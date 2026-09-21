package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/promapi"

	"gorm.io/gorm"
)

// DatasourceHealthResult 数据源采集健康探测结果。
type DatasourceHealthResult struct {
	DatasourceID uint       `json:"datasource_id"`
	Status       string     `json:"status"` // ok|degraded|down|unknown
	Message      string     `json:"message"`
	LatencyMs    int64      `json:"latency_ms"`
	UpTotal      int64      `json:"up_total"`
	UpDown       int64      `json:"up_down"`
	CheckedAt    time.Time  `json:"checked_at"`
	StaleSeconds int64      `json:"stale_seconds,omitempty"`
}

// datasourceHealthBlockTTL 评测闸门：近期探测为 down 则跳过规则评测。
const datasourceHealthBlockTTL = 15 * time.Minute

func (s *AlertDatasourceService) GetHealth(ctx context.Context, id uint) (*DatasourceHealthResult, error) {
	row, err := s.getRaw(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg2f3e2fbecdc5)
		}
		return nil, bizerrors.Pass(ctx, "alert.datasource", "GetHealth", err)
	}
	return healthFromDatasource(row), nil
}

func (s *AlertDatasourceService) ListHealth(ctx context.Context, projectID uint) ([]DatasourceHealthResult, error) {
	if projectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 必填")
	}
	list, _, _, _, err := s.List(ctx, AlertDatasourceListQuery{
		ProjectID: projectID,
		Page:      1,
		PageSize:  500,
	})
	if err != nil {
		return nil, err
	}
	out := make([]DatasourceHealthResult, 0, len(list))
	for i := range list {
		out = append(out, *healthFromDatasource(&list[i].AlertDatasource))
	}
	return out, nil
}

// CheckHealth 即时探测连通性 + up 覆盖，并写回 alert_datasources 缓存字段。
func (s *AlertDatasourceService) CheckHealth(ctx context.Context, id uint) (*DatasourceHealthResult, error) {
	row, err := s.getRaw(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg2f3e2fbecdc5)
		}
		return nil, bizerrors.Pass(ctx, "alert.datasource", "CheckHealth", err)
	}
	res := probeDatasourceHealth(ctx, row)
	now := res.CheckedAt
	row.LastHealthStatus = res.Status
	row.LastHealthAt = &now
	row.LastHealthLatencyMs = res.LatencyMs
	row.LastHealthError = trimErr(res.Message, 512)
	if res.Status != model.DatasourceHealthDown {
		row.LastHealthError = ""
		if res.Status == model.DatasourceHealthDegraded {
			row.LastHealthError = trimErr(res.Message, 512)
		}
	}
	row.LastUpTotal = res.UpTotal
	row.LastUpDown = res.UpDown
	if err := s.repo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.datasource", "CheckHealth.save", err)
	}
	res.DatasourceID = id
	return res, nil
}

func healthFromDatasource(row *model.AlertDatasource) *DatasourceHealthResult {
	if row == nil {
		return &DatasourceHealthResult{Status: model.DatasourceHealthUnknown, Message: "empty"}
	}
	status := strings.TrimSpace(row.LastHealthStatus)
	if status == "" {
		status = model.DatasourceHealthUnknown
	}
	msg := strings.TrimSpace(row.LastHealthError)
	if msg == "" {
		switch status {
		case model.DatasourceHealthOK:
			msg = "ok"
		case model.DatasourceHealthDegraded:
			msg = fmt.Sprintf("部分 target down: %d/%d", row.LastUpDown, row.LastUpTotal)
		case model.DatasourceHealthDown:
			msg = "unreachable"
		default:
			msg = "尚未探测"
		}
	}
	out := &DatasourceHealthResult{
		DatasourceID: row.ID,
		Status:       status,
		Message:      msg,
		LatencyMs:    row.LastHealthLatencyMs,
		UpTotal:      row.LastUpTotal,
		UpDown:       row.LastUpDown,
	}
	if row.LastHealthAt != nil {
		out.CheckedAt = *row.LastHealthAt
		out.StaleSeconds = int64(time.Since(*row.LastHealthAt).Seconds())
	}
	return out
}

func probeDatasourceHealth(ctx context.Context, row *model.AlertDatasource) *DatasourceHealthResult {
	now := time.Now().UTC()
	out := &DatasourceHealthResult{
		DatasourceID: row.ID,
		Status:       model.DatasourceHealthDown,
		CheckedAt:    now,
	}
	t := strings.TrimSpace(row.Type)
	if t == "" {
		t = "prometheus"
	}
	if t == "victoriametrics" {
		t = "victoria"
	}
	if !isPromCompatibleDatasourceType(t) {
		out.Message = "仅 prometheus/victoria 支持健康探测"
		return out
	}
	if strings.TrimSpace(row.BaseURL) == "" {
		out.Message = "base_url 为空"
		return out
	}
	cli := &promapi.Client{
		BaseURL:       row.BaseURL,
		BearerToken:   row.BearerToken,
		BasicUser:     row.BasicUser,
		BasicPassword: row.BasicPassword,
		SkipTLSVerify: row.SkipTLSVerify,
	}
	qctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	start := time.Now()
	body, _, err := cli.QueryInstant(qctx, pingPromQL, "")
	out.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		out.Message = err.Error()
		return out
	}
	if !promapi.QueryResponseStatusSuccess(body) {
		out.Message = "Prometheus 返回非 success 状态"
		return out
	}

	upTotal, upDown := queryUpCoverage(qctx, cli)
	out.UpTotal = upTotal
	out.UpDown = upDown
	if upTotal > 0 && upDown > 0 {
		out.Status = model.DatasourceHealthDegraded
		out.Message = fmt.Sprintf("连通正常，但有 %d/%d 个 up==0", upDown, upTotal)
		return out
	}
	out.Status = model.DatasourceHealthOK
	if upTotal == 0 {
		out.Message = "连通正常（暂无 up 序列，可能尚未接入采集）"
	} else {
		out.Message = fmt.Sprintf("ok, up targets=%d", upTotal)
	}
	return out
}

func queryUpCoverage(ctx context.Context, cli *promapi.Client) (total, down int64) {
	if cli == nil {
		return 0, 0
	}
	total = promQLScalarCount(ctx, cli, "count(up)")
	down = promQLScalarCount(ctx, cli, "count(up == 0)")
	return total, down
}

func promQLScalarCount(ctx context.Context, cli *promapi.Client, expr string) int64 {
	body, _, err := cli.QueryInstant(ctx, expr, "")
	if err != nil || !promapi.QueryResponseStatusSuccess(body) {
		return 0
	}
	return firstVectorFloat(body)
}

func firstVectorFloat(body json.RawMessage) int64 {
	var wrap struct {
		Data struct {
			Result []struct {
				Value []any `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil || len(wrap.Data.Result) == 0 {
		return 0
	}
	if len(wrap.Data.Result[0].Value) < 2 {
		return 0
	}
	switch v := wrap.Data.Result[0].Value[1].(type) {
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return int64(f)
	case float64:
		return int64(v)
	default:
		f, _ := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
		return int64(f)
	}
}

func trimErr(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}

// IsDatasourceHealthBlocking 近期探测为 down 时阻止 PromQL 评测（避免假阴性当「健康」）。
func IsDatasourceHealthBlocking(ds *model.AlertDatasource, now time.Time) bool {
	if ds == nil {
		return false
	}
	if strings.TrimSpace(ds.LastHealthStatus) != model.DatasourceHealthDown {
		return false
	}
	if ds.LastHealthAt == nil {
		return false
	}
	return now.Sub(*ds.LastHealthAt) <= datasourceHealthBlockTTL
}
