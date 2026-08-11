package esmgmt

import (
	"context"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/esclient"
	"yunshu/internal/service/logplatform"

	"gorm.io/gorm"
)

// SchedulerConfigResolver 解析 ES 备份调度字典。
type SchedulerConfigResolver func(ctx context.Context) dictconfig.EsmgmtBackupSchedulerConfig

// Service Elasticsearch 管理服务。
type Service struct {
	db             *gorm.DB
	aead           cipher.AEAD
	logES          *logplatform.ElasticsearchProvider
	newObjectStore ObjectStoreFactory
	resolveSched   SchedulerConfigResolver

	schedMu      sync.Mutex
	schedRunning map[uint]bool
}

// NewService 创建 ES 管理服务。encryptionKey 为空时密码明文写入 PasswordEnc（与部分备份链路兜底一致）。
func NewService(
	db *gorm.DB,
	encryptionKey string,
	logES *logplatform.ElasticsearchProvider,
	newObjectStore ObjectStoreFactory,
	resolveSched SchedulerConfigResolver,
) (*Service, error) {
	s := &Service{
		db:             db,
		logES:          logES,
		newObjectStore: newObjectStore,
		resolveSched:   resolveSched,
		schedRunning:   map[uint]bool{},
	}
	if s.resolveSched == nil {
		s.resolveSched = func(context.Context) dictconfig.EsmgmtBackupSchedulerConfig {
			return dictconfig.ResolveEsmgmtBackupSchedulerConfig(context.Background(), nil, dictconfig.DefaultEsmgmtBackupSchedulerDictTypes())
		}
	}
	key := strings.TrimSpace(encryptionKey)
	if key == "" {
		return s, nil
	}
	aead, err := cryptox.NewAESGCMFromKeyString(key)
	if err != nil {
		return nil, err
	}
	s.aead = aead
	return s, nil
}

// AddressesInput 接受 JSON 字符串或字符串数组。
type AddressesInput []string

func (a *AddressesInput) UnmarshalJSON(b []byte) error {
	b = bytesTrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*a = nil
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*a = splitAddresses(s)
		return nil
	}
	var arr []string
	if err := json.Unmarshal(b, &arr); err != nil {
		return err
	}
	out := make([]string, 0, len(arr))
	for _, it := range arr {
		if v := strings.TrimSpace(it); v != "" {
			out = append(out, v)
		}
	}
	*a = out
	return nil
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func splitAddresses(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func joinAddresses(addrs []string) string {
	return strings.Join(addrs, ",")
}

// ConnectionUpsertRequest 创建/更新连接。
type ConnectionUpsertRequest struct {
	Name       string         `json:"name"`
	Addresses  AddressesInput `json:"addresses"`
	Username   string         `json:"username"`
	Password   string         `json:"password"`
	TimeoutSec int            `json:"timeout_sec"`
	IsDefault  bool           `json:"is_default"`
	Remark     string         `json:"remark"`
}

// PingResult 连通性探测结果。
type PingResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// ProxyRequest 受限 REST 代理请求。
type ProxyRequest struct {
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Body   json.RawMessage `json:"body"`
}

func (s *Service) ListConnections(ctx context.Context) ([]model.EsmgmtConnection, error) {
	var list []model.EsmgmtConnection
	if err := s.db.WithContext(ctx).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		list[i].HasPassword = strings.TrimSpace(list[i].PasswordEnc) != ""
		list[i].PasswordEnc = ""
	}
	return list, nil
}

func (s *Service) CreateConnection(ctx context.Context, req ConnectionUpsertRequest) (*model.EsmgmtConnection, error) {
	name := strings.TrimSpace(req.Name)
	addrs := []string(req.Addresses)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg("连接名称不能为空")
	}
	if len(addrs) == 0 {
		return nil, constants.ErrBadRequestWithMsg("addresses 不能为空")
	}
	timeout := req.TimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	row := model.EsmgmtConnection{
		Name:       name,
		Addresses:  joinAddresses(addrs),
		Username:   strings.TrimSpace(req.Username),
		TimeoutSec: timeout,
		IsDefault:  req.IsDefault,
		Remark:     strings.TrimSpace(req.Remark),
	}
	if pw := strings.TrimSpace(req.Password); pw != "" {
		enc, err := s.encryptPassword(pw)
		if err != nil {
			return nil, err
		}
		row.PasswordEnc = enc
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.IsDefault {
			if err := tx.Model(&model.EsmgmtConnection{}).
				Where("is_default = ?", true).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return nil, err
	}
	row.HasPassword = strings.TrimSpace(row.PasswordEnc) != ""
	row.PasswordEnc = ""
	return &row, nil
}

func (s *Service) UpdateConnection(ctx context.Context, id uint, req ConnectionUpsertRequest) (*model.EsmgmtConnection, error) {
	if id == 0 {
		return nil, constants.ErrBadRequestWithMsg("连接 ID 无效")
	}
	var row model.EsmgmtConnection
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		row.Name = name
	}
	if len(req.Addresses) > 0 {
		row.Addresses = joinAddresses([]string(req.Addresses))
	}
	if strings.TrimSpace(row.Addresses) == "" {
		return nil, constants.ErrBadRequestWithMsg("addresses 不能为空")
	}
	row.Username = strings.TrimSpace(req.Username)
	if req.TimeoutSec > 0 {
		row.TimeoutSec = req.TimeoutSec
	}
	row.IsDefault = req.IsDefault
	row.Remark = strings.TrimSpace(req.Remark)
	if pw := strings.TrimSpace(req.Password); pw != "" {
		enc, err := s.encryptPassword(pw)
		if err != nil {
			return nil, err
		}
		row.PasswordEnc = enc
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.IsDefault {
			if err := tx.Model(&model.EsmgmtConnection{}).
				Where("is_default = ? AND id <> ?", true, id).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(&row).Error
	})
	if err != nil {
		return nil, err
	}
	row.HasPassword = strings.TrimSpace(row.PasswordEnc) != ""
	row.PasswordEnc = ""
	return &row, nil
}

func (s *Service) DeleteConnection(ctx context.Context, id uint) error {
	if id == 0 {
		return constants.ErrBadRequestWithMsg("连接 ID 无效")
	}
	res := s.db.WithContext(ctx).Delete(&model.EsmgmtConnection{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFound
	}
	return nil
}

func (s *Service) PingConnection(ctx context.Context, id uint) (*PingResult, error) {
	cli, err := s.resolveClient(ctx, id)
	if err != nil {
		return &PingResult{OK: false, Message: err.Error()}, nil
	}
	if err := cli.Ping(ctx); err != nil {
		return &PingResult{OK: false, Message: err.Error()}, nil
	}
	return &PingResult{OK: true, Message: "ok"}, nil
}

// TestConnectionRequest 未保存前的连通性探测。
type TestConnectionRequest struct {
	Addresses  AddressesInput `json:"addresses"`
	Username   string         `json:"username"`
	Password   string         `json:"password"`
	TimeoutSec int            `json:"timeout_sec"`
	// ConnectionID>0 且 Password 为空时，复用已存密码。
	ConnectionID uint `json:"connection_id"`
}

// TestConnection 用表单账密探测 ES（可不落库）。
func (s *Service) TestConnection(ctx context.Context, req TestConnectionRequest) (*PingResult, error) {
	addrs := []string(req.Addresses)
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	timeout := req.TimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	if req.ConnectionID > 0 {
		var row model.EsmgmtConnection
		if err := s.db.WithContext(ctx).First(&row, req.ConnectionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &PingResult{OK: false, Message: "连接不存在"}, nil
			}
			return nil, err
		}
		if len(addrs) == 0 {
			addrs = splitAddresses(row.Addresses)
		}
		if username == "" {
			username = row.Username
		}
		if password == "" {
			pw, err := s.decryptPassword(row.PasswordEnc)
			if err != nil {
				return &PingResult{OK: false, Message: "解密密码失败"}, nil
			}
			password = pw
		}
		if timeout <= 0 && row.TimeoutSec > 0 {
			timeout = row.TimeoutSec
		}
	}
	if len(addrs) == 0 {
		return &PingResult{OK: false, Message: "addresses 不能为空"}, nil
	}
	cfg := config.ElasticsearchConfig{
		Enabled:        true,
		Addresses:      addrs,
		Username:       username,
		Password:       password,
		TimeoutSeconds: timeout,
	}
	cli, err := esclient.NewUnmanaged(cfg)
	if err != nil {
		return &PingResult{OK: false, Message: err.Error()}, nil
	}
	if err := cli.Ping(ctx); err != nil {
		return &PingResult{OK: false, Message: err.Error()}, nil
	}
	return &PingResult{OK: true, Message: "ok"}, nil
}

func (s *Service) ClusterHealth(ctx context.Context, connectionID uint) (map[string]any, error) {
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	out, err := cli.ClusterHealth(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("集群健康查询失败: " + err.Error())
	}
	return out, nil
}

func (s *Service) ListIndices(ctx context.Context, connectionID uint, pattern string) ([]esclient.IndexInfo, error) {
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	list, err := cli.CatIndices(ctx, pattern)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("索引列表查询失败: " + err.Error())
	}
	return list, nil
}

// CreateIndexRequest 新建索引。
type CreateIndexRequest struct {
	ConnectionID uint           `json:"connection_id"`
	Name         string         `json:"name"`
	Settings     map[string]any `json:"settings"`
	Mappings     map[string]any `json:"mappings"`
}

// CreateIndex 创建索引（可选 settings/mappings）；禁止系统索引名，已存在则失败。
func (s *Service) CreateIndex(ctx context.Context, req CreateIndexRequest) error {
	index := strings.TrimSpace(req.Name)
	if index == "" {
		return constants.ErrBadRequestWithMsg("索引名不能为空")
	}
	if strings.HasPrefix(index, ".") {
		return constants.ErrBadRequestWithMsg("禁止创建系统索引")
	}
	if strings.ContainsAny(index, `\/?*"<>|, #`) || strings.Contains(index, " ") {
		return constants.ErrBadRequestWithMsg("索引名含非法字符")
	}
	cli, err := s.resolveClient(ctx, req.ConnectionID)
	if err != nil {
		return err
	}
	exists, err := cli.IndexExists(ctx, index)
	if err != nil {
		return constants.ErrBadRequestWithMsg("检查索引失败: " + err.Error())
	}
	if exists {
		return constants.ErrBadRequestWithMsg("索引已存在")
	}
	body := map[string]any{}
	if len(req.Settings) > 0 {
		body["settings"] = req.Settings
	}
	if len(req.Mappings) > 0 {
		body["mappings"] = req.Mappings
	}
	if err := cli.CreateIndex(ctx, index, body); err != nil {
		return constants.ErrBadRequestWithMsg("创建索引失败: " + err.Error())
	}
	return nil
}

// DeleteIndex 删除索引；名称含 yunshu-agent 时须 force=true。
func (s *Service) DeleteIndex(ctx context.Context, connectionID uint, index string, force bool) error {
	index = strings.TrimSpace(index)
	if index == "" {
		return constants.ErrBadRequestWithMsg("索引名不能为空")
	}
	if strings.HasPrefix(index, ".") {
		return constants.ErrBadRequestWithMsg("禁止删除系统索引")
	}
	if strings.Contains(strings.ToLower(index), "yunshu-agent") && !force {
		return constants.ErrBadRequestWithMsg("索引名匹配 yunshu-agent，删除可能影响日志平台，请确认后传 force=true")
	}
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return err
	}
	if err := cli.DeleteIndex(ctx, index); err != nil {
		return constants.ErrBadRequestWithMsg("删除索引失败: " + err.Error())
	}
	return nil
}

func (s *Service) OpenIndex(ctx context.Context, connectionID uint, index string) error {
	index = strings.TrimSpace(index)
	if index == "" {
		return constants.ErrBadRequestWithMsg("索引名不能为空")
	}
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return err
	}
	if err := cli.OpenIndex(ctx, index); err != nil {
		return constants.ErrBadRequestWithMsg("打开索引失败: " + err.Error())
	}
	return nil
}

func (s *Service) CloseIndex(ctx context.Context, connectionID uint, index string) error {
	index = strings.TrimSpace(index)
	if index == "" {
		return constants.ErrBadRequestWithMsg("索引名不能为空")
	}
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return err
	}
	if err := cli.CloseIndex(ctx, index); err != nil {
		return constants.ErrBadRequestWithMsg("关闭索引失败: " + err.Error())
	}
	return nil
}

func (s *Service) CatNodes(ctx context.Context, connectionID uint) ([]map[string]any, error) {
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	out, err := cli.CatNodes(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("节点列表查询失败: " + err.Error())
	}
	return out, nil
}

func (s *Service) ProxyREST(ctx context.Context, connectionID uint, req ProxyRequest) (*esclient.ProxyResult, error) {
	cli, err := s.resolveClient(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	out, err := cli.ProxyREST(ctx, req.Method, req.Path, []byte(req.Body))
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	return out, nil
}

func (s *Service) encryptPassword(plain string) (string, error) {
	if s.aead == nil {
		// encryption_key 未配置时明文落入 password_enc（与部分历史备份兜底一致）。
		return plain, nil
	}
	return cryptox.EncryptString(s.aead, plain)
}

func (s *Service) decryptPassword(enc string) (string, error) {
	enc = strings.TrimSpace(enc)
	if enc == "" {
		return "", nil
	}
	if s.aead == nil {
		return enc, nil
	}
	pt, err := cryptox.DecryptString(s.aead, enc)
	if err != nil {
		// 兼容 encryption_key 变更前写入的明文。
		return enc, nil
	}
	return pt, nil
}

// resolveClient：connectionID>0 用该连接；否则默认连接；再否则回退日志平台 ES。
func (s *Service) resolveClient(ctx context.Context, connectionID uint) (*esclient.Client, error) {
	if connectionID > 0 {
		cli, err := s.clientFromConnectionID(ctx, connectionID)
		if err != nil {
			return nil, err
		}
		return cli, nil
	}
	var def model.EsmgmtConnection
	err := s.db.WithContext(ctx).Where("is_default = ?", true).First(&def).Error
	if err == nil {
		cli, err := s.clientFromRow(ctx, &def)
		if err != nil {
			return nil, err
		}
		return cli, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if s.logES == nil {
		return nil, constants.ErrBadRequestWithMsg("未配置 Elasticsearch 连接，且日志平台 ES 不可用")
	}
	cli, _, err := s.logES.Client(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("Elasticsearch 不可用: " + err.Error())
	}
	return cli, nil
}

func (s *Service) clientFromConnectionID(ctx context.Context, id uint) (*esclient.Client, error) {
	var row model.EsmgmtConnection
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFoundWithMsg("连接不存在")
		}
		return nil, err
	}
	return s.clientFromRow(ctx, &row)
}

func (s *Service) clientFromRow(ctx context.Context, row *model.EsmgmtConnection) (*esclient.Client, error) {
	_ = ctx
	if row == nil {
		return nil, constants.ErrBadRequestWithMsg("连接无效")
	}
	addrs := splitAddresses(row.Addresses)
	if len(addrs) == 0 {
		return nil, constants.ErrBadRequestWithMsg("连接 addresses 为空")
	}
	pw, err := s.decryptPassword(row.PasswordEnc)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("解密连接密码失败")
	}
	timeout := row.TimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	cfg := config.ElasticsearchConfig{
		Enabled:        true,
		Addresses:      addrs,
		Username:       row.Username,
		Password:       pw,
		TimeoutSeconds: timeout,
	}
	cli, err := esclient.NewUnmanaged(cfg)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("创建 ES 客户端失败: %v", err))
	}
	return cli, nil
}
