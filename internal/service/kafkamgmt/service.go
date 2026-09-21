package kafkamgmt

import (
	"context"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/service/logplatform"

	"gorm.io/gorm"
)

// Service Kafka 管理控制台（与日志管道 KafkaToES 隔离）。
type Service struct {
	repo  interfaces.KafkamgmtRepository
	aead  cipher.AEAD
	kafka *logplatform.KafkaProvider
}

// NewService 创建 Kafka 管理服务。encryptionKey 为空时仍可启动，但写入密码会失败。
func NewService(repo interfaces.KafkamgmtRepository, encryptionKey string, kafka *logplatform.KafkaProvider) (*Service, error) {
	s := &Service{repo: repo, kafka: kafka}
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

// BrokersInput 接受 JSON 字符串或字符串数组。
type BrokersInput []string

func (a *BrokersInput) UnmarshalJSON(b []byte) error {
	b = []byte(strings.TrimSpace(string(b)))
	if len(b) == 0 || string(b) == "null" {
		*a = nil
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*a = splitBrokers(s)
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

// ConnectionUpsertRequest 创建/更新连接。
type ConnectionUpsertRequest struct {
	Name          string       `json:"name"`
	Brokers       BrokersInput `json:"brokers"`
	Username      string       `json:"username"`
	Password      string       `json:"password"`
	SASLMechanism string       `json:"sasl_mechanism"`
	TimeoutSec    int          `json:"timeout_sec"`
	IsDefault     bool         `json:"is_default"`
	Remark        string       `json:"remark"`
}

// PingResult 连通性探测结果。
type PingResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func (s *Service) ListConnections(ctx context.Context) ([]model.KafkamgmtConnection, error) {
	list, err := s.repo.ListConnections(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].HasPassword = strings.TrimSpace(list[i].PasswordEnc) != ""
		list[i].PasswordEnc = ""
	}
	return list, nil
}

const dictImportConnectionName = "日志平台（数据字典）"
const dictImportRemarkMarker = "imported_from:kafka_dict"

// ImportConnectionFromDict 从数据字典 kafka_*（及 YAML 兜底）导入/更新一条连接。
func (s *Service) ImportConnectionFromDict(ctx context.Context, actor *auth.CurrentUser) (*model.KafkamgmtConnection, error) {
	if s.kafka == nil {
		return nil, constants.ErrBadRequestWithMsg("日志平台 Kafka Provider 未就绪")
	}
	cfg, err := s.kafka.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	cfg = cfg.Normalized()
	if len(cfg.Brokers) == 0 {
		return nil, constants.ErrBadRequestWithMsg("数据字典未配置 kafka_brokers")
	}
	existing, findErr := s.repo.FindDictImportConnection(ctx, dictImportRemarkMarker, dictImportConnectionName)
	req := ConnectionUpsertRequest{
		Name:          dictImportConnectionName,
		Brokers:       BrokersInput(cfg.Brokers),
		Username:      cfg.Username,
		Password:      cfg.Password,
		SASLMechanism: cfg.SASLMechanism,
		TimeoutSec:    10,
		Remark:        dictImportRemarkMarker + "；来自 kafka_* 字典，可再编辑",
	}
	if findErr == nil {
		if strings.TrimSpace(req.Password) == "" {
			req.Password = ""
		}
		req.IsDefault = existing.IsDefault
		return s.UpdateConnection(ctx, existing.ID, req, actor)
	}
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return nil, findErr
	}
	defCount, _ := s.repo.CountDefaultConnections(ctx)
	req.IsDefault = defCount == 0
	return s.CreateConnection(ctx, req, actor)
}

func (s *Service) CreateConnection(ctx context.Context, req ConnectionUpsertRequest, actor *auth.CurrentUser) (*model.KafkamgmtConnection, error) {
	name := strings.TrimSpace(req.Name)
	brokers := []string(req.Brokers)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg("连接名称不能为空")
	}
	if len(brokers) == 0 {
		return nil, constants.ErrBadRequestWithMsg("brokers 不能为空")
	}
	timeout := req.TimeoutSec
	if timeout <= 0 {
		timeout = 10
	}
	isDefault := req.IsDefault
	if isDefault && !isSuperAdmin(actor) {
		defCount, _ := s.repo.CountDefaultConnections(ctx)
		if defCount > 0 {
			isDefault = false
		}
	}
	row := model.KafkamgmtConnection{
		Name:          name,
		Brokers:       joinBrokers(brokers),
		Username:      strings.TrimSpace(req.Username),
		SASLMechanism: normalizeSASL(req.SASLMechanism, req.Username),
		TimeoutSec:    timeout,
		IsDefault:     isDefault,
		OwnerUserID:   actorID(actor),
		Remark:        strings.TrimSpace(req.Remark),
	}
	if pw := strings.TrimSpace(req.Password); pw != "" {
		enc, err := s.encryptPassword(pw)
		if err != nil {
			return nil, err
		}
		row.PasswordEnc = enc
	}
	if err := s.repo.CreateConnectionClearDefaults(ctx, &row); err != nil {
		return nil, err
	}
	row.HasPassword = strings.TrimSpace(row.PasswordEnc) != ""
	row.PasswordEnc = ""
	return &row, nil
}

func (s *Service) UpdateConnection(ctx context.Context, id uint, req ConnectionUpsertRequest, actor *auth.CurrentUser) (*model.KafkamgmtConnection, error) {
	if err := s.assertConnectionManage(ctx, id, actor); err != nil {
		return nil, err
	}
	row, err := s.repo.GetConnection(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		row.Name = name
	}
	if len(req.Brokers) > 0 {
		row.Brokers = joinBrokers([]string(req.Brokers))
	}
	if strings.TrimSpace(row.Brokers) == "" {
		return nil, constants.ErrBadRequestWithMsg("brokers 不能为空")
	}
	row.Username = strings.TrimSpace(req.Username)
	row.SASLMechanism = normalizeSASL(req.SASLMechanism, row.Username)
	if req.TimeoutSec > 0 {
		row.TimeoutSec = req.TimeoutSec
	}
	if req.IsDefault && !row.IsDefault && !isSuperAdmin(actor) {
		return nil, constants.ErrForbiddenWithMsg("仅超级管理员可将连接设为默认")
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
	if err := s.repo.UpdateConnectionClearDefaults(ctx, id, row); err != nil {
		return nil, err
	}
	row.HasPassword = strings.TrimSpace(row.PasswordEnc) != ""
	row.PasswordEnc = ""
	return row, nil
}

func (s *Service) DeleteConnection(ctx context.Context, id uint, actor *auth.CurrentUser) error {
	if err := s.assertConnectionManage(ctx, id, actor); err != nil {
		return err
	}
	n, err := s.repo.DeleteConnection(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return constants.ErrNotFound
	}
	return nil
}

func (s *Service) PingConnection(ctx context.Context, id uint) (*PingResult, error) {
	ep, err := s.resolveEndpoint(ctx, id)
	if err != nil {
		return &PingResult{OK: false, Message: err.Error()}, nil
	}
	conn, err := ep.dialLeader(ctx)
	if err != nil {
		return &PingResult{OK: false, Message: err.Error()}, nil
	}
	defer conn.Close()
	if _, err := conn.Brokers(); err != nil {
		return &PingResult{OK: false, Message: err.Error()}, nil
	}
	return &PingResult{OK: true, Message: "ok"}, nil
}

// TestConnectionRequest 未保存前的连通性探测。
type TestConnectionRequest struct {
	Brokers       BrokersInput `json:"brokers"`
	Username      string       `json:"username"`
	Password      string       `json:"password"`
	SASLMechanism string       `json:"sasl_mechanism"`
	TimeoutSec    int          `json:"timeout_sec"`
	ConnectionID  uint         `json:"connection_id"`
}

func (s *Service) TestConnection(ctx context.Context, req TestConnectionRequest) (*PingResult, error) {
	brokers := []string(req.Brokers)
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	saslMech := req.SASLMechanism
	timeout := req.TimeoutSec
	if timeout <= 0 {
		timeout = 10
	}
	if req.ConnectionID > 0 {
		row, err := s.repo.GetConnection(ctx, req.ConnectionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &PingResult{OK: false, Message: "连接不存在"}, nil
			}
			return nil, err
		}
		if len(brokers) == 0 {
			brokers = splitBrokers(row.Brokers)
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
		if strings.TrimSpace(saslMech) == "" {
			saslMech = row.SASLMechanism
		}
		if timeout <= 0 && row.TimeoutSec > 0 {
			timeout = row.TimeoutSec
		}
	}
	if len(brokers) == 0 {
		return &PingResult{OK: false, Message: "brokers 不能为空"}, nil
	}
	if timeout <= 0 {
		timeout = 10
	}
	ep := brokerEndpoint{
		brokers:       brokers,
		username:      username,
		password:      password,
		saslMechanism: normalizeSASL(saslMech, username),
		timeout:       time.Duration(timeout) * time.Second,
	}
	conn, err := ep.dialLeader(ctx)
	if err != nil {
		return &PingResult{OK: false, Message: err.Error()}, nil
	}
	defer conn.Close()
	if _, err := conn.Brokers(); err != nil {
		return &PingResult{OK: false, Message: err.Error()}, nil
	}
	return &PingResult{OK: true, Message: "ok"}, nil
}

func (s *Service) encryptPassword(plain string) (string, error) {
	if s.aead == nil {
		return "", constants.ErrBadRequestWithMsg("未配置 security.encryption_key，拒绝明文存储 Kafka 密码")
	}
	return cryptox.EncryptString(s.aead, plain)
}

func (s *Service) decryptPassword(enc string) (string, error) {
	enc = strings.TrimSpace(enc)
	if enc == "" {
		return "", nil
	}
	if s.aead == nil {
		return "", constants.ErrBadRequestWithMsg("未配置 security.encryption_key，无法解密 Kafka 密码")
	}
	return cryptox.DecryptString(s.aead, enc)
}

func (s *Service) resolveEndpoint(ctx context.Context, connectionID uint) (brokerEndpoint, error) {
	var row *model.KafkamgmtConnection
	var err error
	if connectionID > 0 {
		row, err = s.repo.GetConnection(ctx, connectionID)
	} else {
		row, err = s.repo.GetDefaultConnection(ctx)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return brokerEndpoint{}, constants.ErrBadRequestWithMsg("请先在「Kafka 管理控制台 → 连接管理」中配置连接")
		}
		return brokerEndpoint{}, err
	}
	pw, err := s.decryptPassword(row.PasswordEnc)
	if err != nil {
		return brokerEndpoint{}, constants.ErrBadRequestWithMsg("解密连接密码失败")
	}
	ep := endpointFromRow(row, pw)
	if len(ep.brokers) == 0 {
		return brokerEndpoint{}, constants.ErrBadRequestWithMsg("连接 brokers 为空")
	}
	return ep, nil
}
