package cmdb

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"
)

// TestServerConnectivity 测试单台服务器连通性。
func (s *Service) TestServerConnectivity(ctx context.Context, req ServerTestRequest) (*ServerTestResult, error) {
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "TestServerConnectivity", err)
	}
	if strings.TrimSpace(sv.SourceType) == model.ServerGroupCategoryCloud {
		return s.testCloudServerConnectivityBySDK(ctx, sv)
	}
	return s.testSelfHostedServerConnectivityByTCP(ctx, sv)
}

// BatchTestServerConnectivity 批量测试服务器连通性。
func (s *Service) BatchTestServerConnectivity(ctx context.Context, req BatchServerTestRequest) (*BatchServerTestResult, error) {
	serverIDs, err := s.resolveProjectServerIDs(ctx, req.ProjectID, req.ServerIDs)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "BatchTestServerConnectivity", err)
	}
	if len(serverIDs) == 0 {
		return &BatchServerTestResult{Total: 0, Results: []ServerTestResult{}}, nil
	}
	parallel := req.Parallel
	if parallel <= 0 {
		parallel = 5
	}
	if parallel > 20 {
		parallel = 20
	}
	results := s.runServerConnectivityTests(ctx, serverIDs, parallel)
	out := &BatchServerTestResult{Total: len(results), Results: results}
	for _, r := range results {
		if r.OK {
			out.Success++
		} else {
			out.Failed++
		}
	}
	return out, nil
}

// SyncProjectServers 批量探测并刷新项目服务器在线状态。
func (s *Service) SyncProjectServers(ctx context.Context, req ServerSyncRequest) (*ServerSyncResult, error) {
	serverIDs, err := s.resolveProjectServerIDs(ctx, req.ProjectID, req.ServerIDs)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "SyncProjectServers", err)
	}
	if len(serverIDs) == 0 {
		return &ServerSyncResult{UpdatedAt: time.Now().Format(time.RFC3339), TestResults: []ServerTestResult{}}, nil
	}
	parallel := req.Parallel
	if parallel <= 0 {
		parallel = 8
	}
	if parallel > 30 {
		parallel = 30
	}
	results := s.runServerConnectivityTests(ctx, serverIDs, parallel)
	out := &ServerSyncResult{
		Total:       len(results),
		UpdatedAt:   time.Now().Format(time.RFC3339),
		TestResults: results,
	}
	for _, r := range results {
		if r.OK {
			out.Online++
		} else {
			out.Offline++
		}
	}
	return out, nil
}

func (s *Service) resolveProjectServerIDs(ctx context.Context, projectID uint, serverIDs []uint) ([]uint, error) {
	list, _, err := s.serverRepo.List(ctx, repository.ServerListParams{
		ProjectID: projectID,
		Page:      1,
		PageSize:  10000,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "resolveProjectServerIDs", err)
	}
	if len(serverIDs) == 0 {
		out := make([]uint, 0, len(list))
		for _, it := range list {
			out = append(out, it.ID)
		}
		return out, nil
	}
	allowed := make(map[uint]struct{}, len(list))
	for _, it := range list {
		allowed[it.ID] = struct{}{}
	}
	out := make([]uint, 0, len(serverIDs))
	for _, id := range serverIDs {
		if _, ok := allowed[id]; ok {
			out = append(out, id)
		}
	}
	return out, nil
}

func (s *Service) runServerConnectivityTests(ctx context.Context, serverIDs []uint, parallel int) []ServerTestResult {
	type job struct {
		id uint
	}
	jobs := make(chan job, len(serverIDs))
	results := make(chan ServerTestResult, len(serverIDs))
	worker := func() {
		for it := range jobs {
			r, err := s.TestServerConnectivity(ctx, ServerTestRequest{ServerID: it.id})
			if err != nil {
				results <- ServerTestResult{ServerID: it.id, OK: false, Message: err.Error()}
				continue
			}
			results <- *r
		}
	}
	var wg sync.WaitGroup
	for range parallel {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Default().With("component", "cmdb").Error("connectivity test worker panic", "recover", r)
				}
			}()
			worker()
		})
	}
	for _, id := range serverIDs {
		jobs <- job{id: id}
	}
	close(jobs)
	wg.Wait()
	close(results)
	out := make([]ServerTestResult, 0, len(serverIDs))
	for it := range results {
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ServerID < out[j].ServerID })
	return out
}

func (s *Service) testSelfHostedServerConnectivityByTCP(ctx context.Context, sv *model.Server) (*ServerTestResult, error) {
	host := strings.TrimSpace(sv.Host)
	if host == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgf2664ad99ec4)
	}
	port := sv.Port
	if port <= 0 {
		port = 22
	}
	target := fmt.Sprintf("%s:%d", host, port)
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(cctx, "tcp", target)
	now := time.Now()
	sv.LastTestAt = &now
	if err != nil {
		msg := "[TCP] " + err.Error()
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}
	_ = conn.Close()
	sv.LastTestError = nil
	_ = s.serverRepo.Save(ctx, sv)
	return &ServerTestResult{ServerID: sv.ID, OK: true, Message: fmt.Sprintf("[TCP] reachable: %s", target)}, nil
}

func (s *Service) testCloudServerConnectivityBySDK(ctx context.Context, sv *model.Server) (*ServerTestResult, error) {
	now := time.Now()
	sv.LastTestAt = &now

	if sv.GroupID == nil {
		msg := "[SDK] cloud server missing group_id"
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}

	groupID := *sv.GroupID
	accounts, err := s.cloudAccountRepo.ListByProjectAndGroup(ctx, sv.ProjectID, &groupID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "testCloudServerConnectivityBySDK", err)
	}
	providerName := strings.TrimSpace(sv.Provider)
	var account *model.CloudAccount
	for i := range accounts {
		it := &accounts[i]
		if it.Status != model.StatusEnabled {
			continue
		}
		if providerName == "" || strings.EqualFold(strings.TrimSpace(it.Provider), providerName) {
			account = it
			break
		}
	}
	if account == nil {
		msg := "[SDK] no enabled cloud account found for this group/provider"
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}
	if account.EncAK == nil || account.EncSK == nil {
		msg := "[SDK] cloud account AK/SK 未配置"
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}

	ak, err := cryptox.DecryptString(s.aead, *account.EncAK)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgCloudCredentialDecryptFailed)
	}
	sk, err := cryptox.DecryptString(s.aead, *account.EncSK)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgCloudCredentialDecryptFailed)
	}
	provider, err := s.providerFor(account.Provider)
	if err != nil {
		msg := "[SDK] " + err.Error()
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}

	instances, err := provider.ListInstances(ctx, ak, sk, account.RegionScope)
	if err != nil {
		msg := "[SDK] " + err.Error()
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}

	instanceID := strings.TrimSpace(sv.CloudInstanceID)
	if instanceID == "" {
		// Best-effort: try to infer instance id by matching server host (public/private IP).
		host := strings.TrimSpace(sv.Host)
		for _, ins := range instances {
			if host == "" {
				break
			}
			if strings.EqualFold(strings.TrimSpace(ins.Host), host) ||
				strings.EqualFold(strings.TrimSpace(ins.PublicIP), host) ||
				strings.EqualFold(strings.TrimSpace(ins.PrivateIP), host) {
				instanceID = strings.TrimSpace(ins.InstanceID)
				if instanceID != "" {
					sv.CloudInstanceID = instanceID
					if strings.TrimSpace(sv.CloudRegion) == "" && strings.TrimSpace(ins.Region) != "" {
						sv.CloudRegion = strings.TrimSpace(ins.Region)
					}
					_ = s.serverRepo.Save(ctx, sv)
				}
				break
			}
		}
		if instanceID == "" {
			msg := "[SDK] cloud_instance_id 为空：请先同步云账号或在服务器编辑中补充实例 ID"
			sv.LastTestError = &msg
			_ = s.serverRepo.Save(ctx, sv)
			return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
		}
	}
	for _, ins := range instances {
		if strings.TrimSpace(ins.InstanceID) != instanceID {
			continue
		}
		if ins.Status == model.StatusEnabled {
			sv.LastTestError = nil
			_ = s.serverRepo.Save(ctx, sv)
			return &ServerTestResult{
				ServerID: sv.ID,
				OK:       true,
				Message:  fmt.Sprintf("[SDK] instance %s is running", instanceID),
			}, nil
		}
		msg := fmt.Sprintf("[SDK] instance %s is not running", instanceID)
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}

	msg := fmt.Sprintf("[SDK] instance %s not found in provider result", instanceID)
	sv.LastTestError = &msg
	_ = s.serverRepo.Save(ctx, sv)
	return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
}

