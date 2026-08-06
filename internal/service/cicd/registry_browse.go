package cicd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

type HarborProjectItem struct {
	ProjectID int64  `json:"project_id"`
	Name      string `json:"name"`
}

type HarborRepoItem struct {
	Name          string `json:"name"`
	ArtifactCount int64  `json:"artifact_count"`
}

type HarborTagItem struct {
	Digest    string             `json:"digest"`
	Tags      []string           `json:"tags"`
	Size      int64              `json:"size"`
	PushTime  string             `json:"push_time"`
	Linked    []LinkedBuildBrief `json:"linked_build_runs,omitempty"`
}

type LinkedBuildBrief struct {
	ID          uint   `json:"id"`
	ProjectID   uint   `json:"project_id"`
	ServiceID   uint   `json:"service_id"`
	BuildNumber int    `json:"build_number"`
	ImageAddress string `json:"image_address"`
}

func (s *Service) resolveBrowseRegistry(ctx context.Context, registryID, projectID uint) (ResolvedRegistry, error) {
	if registryID > 0 {
		reg, err := s.getRegistry(ctx, registryID)
		if err != nil {
			return ResolvedRegistry{}, err
		}
		return registryToResolved(reg), nil
	}
	return s.ResolveRegistryForProject(ctx, projectID), nil
}

func (s *Service) ListHarborProjects(ctx context.Context, registryID, projectID uint) ([]HarborProjectItem, error) {
	r, err := s.resolveBrowseRegistry(ctx, registryID, projectID)
	if err != nil {
		return nil, err
	}
	client := registryHTTPClient(r.URL)
	if r.Type == model.ImageRegistryTypeDockerRegistry {
		return []HarborProjectItem{{Name: "_catalog"}}, nil
	}
	code, body, err := s.registryDo(ctx, client, r, "GET", "/api/v2.0/projects?page_size=100")
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	if code < 200 || code >= 300 {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("Harbor projects HTTP %d", code))
	}
	var raw []struct {
		ProjectID int64  `json:"project_id"`
		Name      string `json:"name"`
	}
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil, constants.ErrBadRequestWithMsg("解析 Harbor projects 失败")
	}
	out := make([]HarborProjectItem, 0, len(raw))
	for _, p := range raw {
		out = append(out, HarborProjectItem{ProjectID: p.ProjectID, Name: p.Name})
	}
	return out, nil
}

func (s *Service) ListHarborRepositories(ctx context.Context, registryID, projectID uint, harborProject string) ([]HarborRepoItem, error) {
	r, err := s.resolveBrowseRegistry(ctx, registryID, projectID)
	if err != nil {
		return nil, err
	}
	client := registryHTTPClient(r.URL)
	harborProject = strings.TrimSpace(harborProject)
	if harborProject == "" {
		harborProject = r.ProjectGroup
	}
	if r.Type == model.ImageRegistryTypeDockerRegistry {
		code, body, err := s.registryDo(ctx, client, r, "GET", "/v2/_catalog")
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg(err.Error())
		}
		if code < 200 || code >= 300 {
			return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("catalog HTTP %d", code))
		}
		var cat struct {
			Repositories []string `json:"repositories"`
		}
		_ = json.Unmarshal([]byte(body), &cat)
		out := make([]HarborRepoItem, 0, len(cat.Repositories))
		for _, name := range cat.Repositories {
			out = append(out, HarborRepoItem{Name: name})
		}
		return out, nil
	}
	path := fmt.Sprintf("/api/v2.0/projects/%s/repositories?page_size=100", url.PathEscape(harborProject))
	code, body, err := s.registryDo(ctx, client, r, "GET", path)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	if code < 200 || code >= 300 {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("repositories HTTP %d: %s", code, truncate(body, 200)))
	}
	var raw []struct {
		Name          string `json:"name"`
		ArtifactCount int64  `json:"artifact_count"`
	}
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil, constants.ErrBadRequestWithMsg("解析 repositories 失败")
	}
	out := make([]HarborRepoItem, 0, len(raw))
	for _, it := range raw {
		name := it.Name
		if strings.HasPrefix(name, harborProject+"/") {
			name = strings.TrimPrefix(name, harborProject+"/")
		}
		out = append(out, HarborRepoItem{Name: name, ArtifactCount: it.ArtifactCount})
	}
	return out, nil
}

func (s *Service) ListHarborArtifacts(ctx context.Context, registryID, projectID uint, harborProject, repo string) ([]HarborTagItem, error) {
	r, err := s.resolveBrowseRegistry(ctx, registryID, projectID)
	if err != nil {
		return nil, err
	}
	client := registryHTTPClient(r.URL)
	harborProject = strings.TrimSpace(harborProject)
	repo = strings.TrimSpace(repo)
	if harborProject == "" {
		harborProject = r.ProjectGroup
	}
	if repo == "" {
		return nil, constants.ErrBadRequestWithMsg("repository 不能为空")
	}

	var items []HarborTagItem
	if r.Type == model.ImageRegistryTypeDockerRegistry {
		path := fmt.Sprintf("/v2/%s/tags/list", strings.TrimPrefix(repo, "/"))
		code, body, err := s.registryDo(ctx, client, r, "GET", path)
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg(err.Error())
		}
		if code < 200 || code >= 300 {
			return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("tags HTTP %d", code))
		}
		var tags struct {
			Tags []string `json:"tags"`
		}
		_ = json.Unmarshal([]byte(body), &tags)
		for _, t := range tags.Tags {
			img := fmt.Sprintf("%s/%s:%s", stripHarborHost(r.URL), repo, t)
			items = append(items, HarborTagItem{
				Tags:   []string{t},
				Linked: s.findLinkedBuilds(ctx, img),
			})
		}
		return items, nil
	}

	encodedRepo := url.PathEscape(repo)
	// Harbor 双层 PathEscape：foo/bar → foo%252Fbar 在部分版本；优先 %2F
	path := fmt.Sprintf(
		"/api/v2.0/projects/%s/repositories/%s/artifacts?page_size=100&with_tag=true",
		url.PathEscape(harborProject),
		strings.ReplaceAll(encodedRepo, "%2F", "%252F"),
	)
	// 更通用：repository 名用 double-encode of /
	path = fmt.Sprintf(
		"/api/v2.0/projects/%s/repositories/%s/artifacts?page_size=100&with_tag=true",
		url.PathEscape(harborProject),
		encodeHarborRepoPath(repo),
	)
	code, body, err := s.registryDo(ctx, client, r, "GET", path)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	if code < 200 || code >= 300 {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("artifacts HTTP %d: %s", code, truncate(body, 200)))
	}
	var raw []struct {
		Digest   string `json:"digest"`
		Size     int64  `json:"size"`
		PushTime string `json:"push_time"`
		Tags     []struct {
			Name string `json:"name"`
		} `json:"tags"`
	}
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil, constants.ErrBadRequestWithMsg("解析 artifacts 失败")
	}
	host := stripHarborHost(r.URL)
	for _, a := range raw {
		tags := make([]string, 0, len(a.Tags))
		for _, t := range a.Tags {
			if t.Name != "" {
				tags = append(tags, t.Name)
			}
		}
		var linked []LinkedBuildBrief
		for _, t := range tags {
			img := fmt.Sprintf("%s/%s/%s:%s", host, harborProject, repo, t)
			linked = append(linked, s.findLinkedBuilds(ctx, img)...)
		}
		if len(tags) == 0 && a.Digest != "" {
			img := fmt.Sprintf("%s/%s/%s@%s", host, harborProject, repo, a.Digest)
			linked = s.findLinkedBuilds(ctx, img)
		}
		items = append(items, HarborTagItem{
			Digest:   a.Digest,
			Tags:     tags,
			Size:     a.Size,
			PushTime: a.PushTime,
			Linked:   dedupeLinked(linked),
		})
	}
	return items, nil
}

func encodeHarborRepoPath(repo string) string {
	// Harbor expects single segment with %2F for nested names
	parts := strings.Split(strings.Trim(repo, "/"), "/")
	escaped := make([]string, 0, len(parts))
	for _, p := range parts {
		escaped = append(escaped, url.PathEscape(p))
	}
	return strings.Join(escaped, "%2F")
}

func (s *Service) DeleteHarborArtifact(ctx context.Context, registryID, projectID uint, harborProject, repo, reference string) error {
	r, err := s.resolveBrowseRegistry(ctx, registryID, projectID)
	if err != nil {
		return err
	}
	client := registryHTTPClient(r.URL)
	harborProject = strings.TrimSpace(harborProject)
	repo = strings.TrimSpace(repo)
	reference = strings.TrimSpace(reference)
	if harborProject == "" {
		harborProject = r.ProjectGroup
	}
	if repo == "" || reference == "" {
		return constants.ErrBadRequestWithMsg("repository 与 reference（tag 或 digest）必填")
	}
	if r.Type == model.ImageRegistryTypeDockerRegistry {
		// Docker Registry V2: GET manifest for digest then DELETE
		manifestPath := fmt.Sprintf("/v2/%s/manifests/%s", strings.TrimPrefix(repo, "/"), reference)
		code, _, err := s.registryDo(ctx, client, r, "DELETE", manifestPath)
		if err != nil {
			return constants.ErrBadRequestWithMsg(err.Error())
		}
		if code < 200 || code >= 300 {
			return constants.ErrBadRequestWithMsg(fmt.Sprintf("删除失败 HTTP %d", code))
		}
		return nil
	}
	path := fmt.Sprintf(
		"/api/v2.0/projects/%s/repositories/%s/artifacts/%s",
		url.PathEscape(harborProject),
		encodeHarborRepoPath(repo),
		url.PathEscape(reference),
	)
	code, body, err := s.registryDo(ctx, client, r, "DELETE", path)
	if err != nil {
		return constants.ErrBadRequestWithMsg(err.Error())
	}
	if code < 200 || code >= 300 {
		return constants.ErrBadRequestWithMsg(fmt.Sprintf("删除失败 HTTP %d: %s", code, truncate(body, 200)))
	}
	return nil
}

func (s *Service) findLinkedBuilds(ctx context.Context, imageAddress string) []LinkedBuildBrief {
	imageAddress = strings.TrimSpace(imageAddress)
	if imageAddress == "" || s.db == nil {
		return nil
	}
	var rows []model.CicdBuildRun
	_ = s.db.WithContext(ctx).
		Select("id", "project_id", "service_id", "build_number", "image_address").
		Where("image_address = ? OR image_address LIKE ?", imageAddress, imageAddress+"%").
		Order("id DESC").
		Limit(20).
		Find(&rows).Error
	out := make([]LinkedBuildBrief, 0, len(rows))
	for _, r := range rows {
		out = append(out, LinkedBuildBrief{
			ID:           r.ID,
			ProjectID:    r.ProjectID,
			ServiceID:    r.ServiceID,
			BuildNumber:  r.BuildNumber,
			ImageAddress: r.ImageAddress,
		})
	}
	return out
}

func dedupeLinked(in []LinkedBuildBrief) []LinkedBuildBrief {
	seen := map[uint]struct{}{}
	out := make([]LinkedBuildBrief, 0, len(in))
	for _, it := range in {
		if _, ok := seen[it.ID]; ok {
			continue
		}
		seen[it.ID] = struct{}{}
		out = append(out, it)
	}
	return out
}

func parseInt64Loose(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}
