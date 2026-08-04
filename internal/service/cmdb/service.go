package cmdb

import (
	"context"
	"crypto/cipher"
	"sync"
	"time"

	"yunshu/internal/interfaces"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

// Service CMDB 服务器资产业务（主机、分组、云账号、SSH/终端）。
type Service struct {
	db               *gorm.DB
	serverRepo       interfaces.ServerRepository
	serverGroupRepo  interfaces.ServerGroupRepository
	cloudAccountRepo interfaces.CloudAccountRepository
	memberRepo       interfaces.ProjectMemberRepository
	aead             cipher.AEAD
	ensureMu         sync.Mutex
	ensuredProjectAt map[uint]time.Time
}

// NewService 创建 CMDB 服务。
func NewService(
	db *gorm.DB,
	serverRepo interfaces.ServerRepository,
	serverGroupRepo interfaces.ServerGroupRepository,
	cloudAccountRepo interfaces.CloudAccountRepository,
	memberRepo interfaces.ProjectMemberRepository,
	encryptionKey string,
) (*Service, error) {
	aead, err := cryptox.NewAESGCMFromKeyString(encryptionKey)
	if err != nil {
		return nil, bizerrors.Pass(context.Background(), "cmdb", "NewService", err)
	}
	return &Service{
		db:               db,
		serverRepo:       serverRepo,
		serverGroupRepo:  serverGroupRepo,
		cloudAccountRepo: cloudAccountRepo,
		memberRepo:       memberRepo,
		aead:             aead,
		ensuredProjectAt: make(map[uint]time.Time),
	}, nil
}

// EncryptionReady 校验加密密钥是否可用（导入/凭证保存前）。
func (s *Service) EncryptionReady() error {
	if s == nil || s.aead == nil {
		return bizerrors.Internalf(context.Background(), "cmdb", "EncryptionReady", nil, "encryption not configured")
	}
	return nil
}
