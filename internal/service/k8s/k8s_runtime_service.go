package k8s

import (
	"context"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/eventbus"
	"yunshu/internal/pkg/extension"

	"github.com/weibaohui/kom/callbacks"
	kom "github.com/weibaohui/kom/kom"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sRuntimeService struct {
	repo       interfaces.K8sClusterRepository
	nsDeny     interfaces.K8sNamespaceDenyRepository
	nsAllow    interfaces.K8sNamespaceAllowRepository
	memberRepo interfaces.ProjectMemberRepository
	aead       cipher.AEAD

	komInitOnce    sync.Once
	komMu          sync.Mutex
	registeredHash map[string]string
	connState      map[string]ClusterConnState
	regLocks       map[string]*sync.Mutex
}

type ClusterConnState struct {
	State               string    `json:"state"`
	LastError           string    `json:"last_error,omitempty"`
	LastAttemptAt       time.Time `json:"last_attempt_at,omitempty"`
	LastSuccessAt       time.Time `json:"last_success_at,omitempty"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
}

// NewK8sRuntimeService 创建运行时；encryptionKey 用于解密库内 kubeconfig/direct_config。
func NewK8sRuntimeService(
	repo interfaces.K8sClusterRepository,
	nsDeny interfaces.K8sNamespaceDenyRepository,
	nsAllow interfaces.K8sNamespaceAllowRepository,
	memberRepo interfaces.ProjectMemberRepository,
	encryptionKey string,
) (*K8sRuntimeService, error) {
	var aead cipher.AEAD
	if strings.TrimSpace(encryptionKey) != "" {
		a, err := cryptox.NewAESGCMFromKeyString(encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("k8s runtime encryption_key: %w", err)
		}
		aead = a
	}
	return &K8sRuntimeService{
		repo:           repo,
		nsDeny:         nsDeny,
		nsAllow:        nsAllow,
		memberRepo:     memberRepo,
		aead:           aead,
		registeredHash: map[string]string{},
		connState:      map[string]ClusterConnState{},
		regLocks:       map[string]*sync.Mutex{},
	}, nil
}

// SealCredential 加密集群凭证字段。
func (s *K8sRuntimeService) SealCredential(plain string) (string, error) {
	if s == nil {
		return plain, nil
	}
	return sealClusterSecret(s.aead, plain)
}

// OpenCredential 解密集群凭证字段（兼容明文存量）。
func (s *K8sRuntimeService) OpenCredential(stored string) (string, error) {
	if s == nil {
		return stored, nil
	}
	return openClusterSecret(s.aead, stored)
}

// NamespaceDenyRepo 供 DynamicResourceService 做列表过滤。
func (s *K8sRuntimeService) NamespaceDenyRepo() interfaces.K8sNamespaceDenyRepository {
	if s == nil {
		return nil
	}
	return s.nsDeny
}

// NamespaceAllowRepo 供 DynamicResourceService 做列表过滤。
func (s *K8sRuntimeService) NamespaceAllowRepo() interfaces.K8sNamespaceAllowRepository {
	if s == nil {
		return nil
	}
	return s.nsAllow
}

func (s *K8sRuntimeService) getRegLock(clusterID string) *sync.Mutex {
	s.komMu.Lock()
	defer s.komMu.Unlock()
	if lk, ok := s.regLocks[clusterID]; ok && lk != nil {
		return lk
	}
	lk := &sync.Mutex{}
	s.regLocks[clusterID] = lk
	return lk
}

func (s *K8sRuntimeService) ensureKomInit() {
	s.komInitOnce.Do(func() {
		callbacks.RegisterInit()
	})
}

func (s *K8sRuntimeService) registerClusterIfNeeded(clusterID string, kubeconfig string, force bool) error {
	s.ensureKomInit()
	sum := sha256.Sum256([]byte(kubeconfig))
	hash := hex.EncodeToString(sum[:])

	lk := s.getRegLock(clusterID)
	lk.Lock()
	defer lk.Unlock()

	s.komMu.Lock()
	st := s.connState[clusterID]
	st.LastAttemptAt = time.Now()
	st.State = "connecting"
	s.connState[clusterID] = st
	prev := s.registeredHash[clusterID]
	if !force && prev != "" && prev == hash {
		st.State = "ready"
		st.LastError = ""
		st.LastSuccessAt = time.Now()
		st.ConsecutiveFailures = 0
		s.connState[clusterID] = st
		s.komMu.Unlock()
		return nil
	}
	s.komMu.Unlock()

	// kom RegisterByConfigWithID 在 clusterID 已存在且 Kubectl 非空时会直接返回旧实例，不更新 server/凭据。
	// 切换 kubeconfig/direct 或修改 API 地址后必须先 Remove，否则仍会连到旧地址（如 10.10.10.103）。
	kom.Clusters().RemoveClusterById(clusterID)

	// 使用 kom 原生 RegisterByStringWithID，与 RegisterByPathWithID 等价，避免临时文件在容器/Windows 下的路径问题。
	// RegisterTimeout 防止不可达 APIServer 在 dial/握手阶段无限阻塞（总览等多集群并发时尤甚）。
	_, err := kom.Clusters().RegisterByStringWithID(
		kubeconfig,
		clusterID,
		kom.RegisterTimeout(defaultKomRegisterTimeout),
		kom.RegisterQPS(defaultK8sQPS),
		kom.RegisterBurst(defaultK8sBurst),
	)
	if err != nil {
		_ = bizerrors.Pass(context.Background(), "k8s.runtime", "registerClusterIfNeeded", err, "cluster_id", clusterID)
		s.komMu.Lock()
		st.State = "degraded"
		st.LastError = err.Error()
		st.ConsecutiveFailures++
		s.connState[clusterID] = st
		s.komMu.Unlock()
		extension.NotifyKomRegister(clusterID, false, err.Error())
		eventbus.Default().Publish(eventbus.Event{
			Type:    eventbus.ClusterKomRegisterFail,
			Payload: map[string]any{"cluster_id": clusterID, "error": err.Error()},
		})
		return bizerrors.MarkLogged(err)
	}
	s.komMu.Lock()
	s.registeredHash[clusterID] = hash
	st.State = "ready"
	st.LastError = ""
	st.LastSuccessAt = time.Now()
	st.ConsecutiveFailures = 0
	s.connState[clusterID] = st
	s.komMu.Unlock()
	extension.NotifyKomRegister(clusterID, true, "")
	eventbus.Default().Publish(eventbus.Event{
		Type:    eventbus.ClusterKomRegisterOK,
		Payload: map[string]any{"cluster_id": clusterID},
	})
	return nil
}

// DeleteRegisterCache 删除相关的业务逻辑（含 :ro/:w/:x 与 impersonation 后缀）。
func (s *K8sRuntimeService) DeleteRegisterCache(clusterID uint) {
	prefix := strconv.FormatUint(uint64(clusterID), 10)
	kom.Clusters().RemoveClusterById(prefix)
	s.komMu.Lock()
	for key := range s.registeredHash {
		if key == prefix || strings.HasPrefix(key, prefix+":") {
			kom.Clusters().RemoveClusterById(key)
			delete(s.registeredHash, key)
			delete(s.regLocks, key)
			delete(s.connState, key)
		}
	}
	s.komMu.Unlock()
}

// PeekRegisteredKubectl 返回进程内已注册的 kubectl（bare / :ro / :r / :w / :x），不触发冷注册。
// 供总览等批量只读路径复用既有连接，避免默认 write 意图反复 Register。
func (s *K8sRuntimeService) PeekRegisteredKubectl(id uint) *kom.Kubectl {
	base := strconv.FormatUint(uint64(id), 10)
	for _, key := range []string{base, base + ":ro", base + ":r", base + ":w", base + ":x"} {
		if k := kom.Cluster(key); k != nil {
			return k
		}
	}
	return nil
}

// GetClusterKubectl 按请求 AccessIntent 选择只读/可写凭证（平台侧授权；不使用 Impersonation）。
func (s *K8sRuntimeService) GetClusterKubectl(ctx context.Context, id uint) (*model.K8sCluster, *kom.Kubectl, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, k8sRepoErr(ctx, "k8s.runtime", "GetClusterKubectl", err, "cluster_id", id)
	}
	if cluster.Status != 1 {
		return nil, nil, constants.ErrForbiddenWithMsg(constants.ErrMsgb0e556f1ccc5)
	}
	if err := s.ensureOwningProjectAccess(ctx, cluster); err != nil {
		return nil, nil, err
	}
	intent := accessIntentFromContext(ctx)
	kubeconfig, regID, kerr := s.resolveKubeconfigForIntent(cluster, intent)
	if kerr != nil {
		return nil, nil, constants.ErrBadRequestWithMsg(kerr.Error())
	}
	if err := s.registerClusterIfNeeded(regID, kubeconfig, false); err != nil {
		return nil, nil, bizerrors.Internalf(ctx, "k8s.runtime", "GetClusterKubectl", err, constants.ErrFmtac130d1176b3, "cluster_id", id)
	}
	k := kom.Cluster(regID)
	if k == nil {
		return nil, nil, bizerrors.InternalMsg(ctx, "k8s.runtime", "GetClusterKubectl", constants.ErrMsg5248c9e19a3f, "cluster_id", id)
	}
	return cluster, k, nil
}

// ensureOwningProjectAccess 项目归属集群：有用户上下文时须为项目成员（后台任务无用户则跳过）。
func (s *K8sRuntimeService) ensureOwningProjectAccess(ctx context.Context, cl *model.K8sCluster) error {
	if cl == nil || cl.OwningProjectID == nil || *cl.OwningProjectID == 0 {
		return nil
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return nil
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return nil
	}
	if s.memberRepo == nil {
		return constants.ErrInternal
	}
	_, err := s.memberRepo.GetByProjectAndUser(ctx, *cl.OwningProjectID, u.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrK8sClusterProjectAccessDenied
		}
		return bizerrors.Pass(ctx, "k8s.runtime", "ensureOwningProjectAccess", err)
	}
	return nil
}

// EnsureClusterRegistered 将集群注册到 kom（供 Event 转发等后台任务使用）。
// 后台任务使用稳定 ID（纯数字 cluster_id），不走 :r/:w 后缀，避免与控制台意图注册冲突。
func (s *K8sRuntimeService) EnsureClusterRegistered(ctx context.Context, id uint) (*kom.Kubectl, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, k8sRepoErr(ctx, "k8s.runtime", "EnsureClusterRegistered", err, "cluster_id", id)
	}
	if cluster.Status != 1 {
		return nil, constants.ErrForbiddenWithMsg(constants.ErrMsgb0e556f1ccc5)
	}
	kubeconfig, kerr := s.resolveClusterKubeconfig(cluster)
	if kerr != nil {
		return nil, constants.ErrBadRequestWithMsg(kerr.Error())
	}
	clusterID := strconv.FormatUint(uint64(id), 10)
	// 优先只读凭证（后台 watch 只需读）
	if strings.TrimSpace(cluster.KubeconfigReadonly) != "" {
		if raw, oerr := s.OpenCredential(cluster.KubeconfigReadonly); oerr == nil {
			if kc, nerr := normalizeKubeconfigForClientGo(strings.TrimSpace(raw)); nerr == nil && kc != "" {
				kubeconfig = kc
			}
		}
	}
	if err := s.registerClusterIfNeeded(clusterID, kubeconfig, false); err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.runtime", "EnsureClusterRegistered", err, constants.ErrFmtac130d1176b3, "cluster_id", id)
	}
	k := kom.Cluster(clusterID)
	if k == nil {
		// hash 命中但 kom 实例已丢失时强制重注册，避免后台 watch 拿到 nil
		if err := s.registerClusterIfNeeded(clusterID, kubeconfig, true); err != nil {
			return nil, bizerrors.Internalf(ctx, "k8s.runtime", "EnsureClusterRegistered", err, constants.ErrFmtac130d1176b3, "cluster_id", id)
		}
		k = kom.Cluster(clusterID)
	}
	if k == nil {
		return nil, bizerrors.InternalMsg(ctx, "k8s.runtime", "EnsureClusterRegistered", constants.ErrMsg5248c9e19a3f, "cluster_id", id)
	}
	return k, nil
}

// serverGitVersionFromKubeconfig 使用 client-go Discovery 拉取 GitVersion（与 kubectl 一致）。
// kom 在进程重启后偶发 Status().ServerVersion() 为空，不能作为心跳唯一依据。
func serverGitVersionFromKubeconfig(kubeconfig string) (string, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return "", err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", err
	}
	sv, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	if sv == nil || strings.TrimSpace(sv.GitVersion) == "" {
		return "", fmt.Errorf("APIServer 返回的版本信息为空")
	}
	return strings.TrimSpace(sv.GitVersion), nil
}

// CheckClusterHeartbeat 执行对应的业务逻辑。
func (s *K8sRuntimeService) CheckClusterHeartbeat(ctx context.Context, id uint) (string, ClusterConnState, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", ClusterConnState{}, k8sRepoErr(ctx, "k8s.runtime", "CheckClusterHeartbeat", err, "cluster_id", id)
	}
	if cluster.Status != 1 {
		// 停用集群：不做心跳/重连，直接标记 disabled
		s.komMu.Lock()
		key := strconv.FormatUint(uint64(id), 10)
		st := s.connState[key]
		st.State = "disabled"
		s.connState[key] = st
		s.komMu.Unlock()
		return "", s.GetClusterConnState(id), nil
	}
	clusterID := strconv.FormatUint(uint64(id), 10)
	kubeconfig, kerr := s.resolveClusterKubeconfig(cluster)
	if kerr != nil {
		return "", s.GetClusterConnState(id), constants.ErrBadRequestWithMsg(kerr.Error())
	}
	if err := s.registerClusterIfNeeded(clusterID, kubeconfig, false); err != nil {
		return "", s.GetClusterConnState(id), err
	}
	k := kom.Cluster(clusterID)
	if k == nil {
		return "", s.GetClusterConnState(id), constants.ErrInternalWithMsg(constants.ErrMsg5248c9e19a3f)
	}
	gitVer, verr := serverGitVersionFromKubeconfig(kubeconfig)
	if verr != nil || gitVer == "" {
		s.DeleteRegisterCache(id)
		if e := s.registerClusterIfNeeded(clusterID, kubeconfig, true); e != nil {
			return "", s.GetClusterConnState(id), bizerrors.Internalf(ctx, "k8s.runtime", "heartbeat_reregister", e, constants.ErrFmt8648d0eaa652)
		}
		if kom.Cluster(clusterID) == nil {
			return "", s.GetClusterConnState(id), constants.ErrInternalWithMsg(constants.ErrMsgb9cf6d1a2c2e)
		}
		gitVer, verr = serverGitVersionFromKubeconfig(kubeconfig)
		if verr != nil || gitVer == "" {
			errMsg := "server version empty"
			if verr != nil {
				errMsg = classifyClusterConnectError(verr)
			}
			s.komMu.Lock()
			st := s.connState[clusterID]
			st.State = "degraded"
			st.LastAttemptAt = time.Now()
			st.LastError = errMsg
			st.ConsecutiveFailures++
			s.connState[clusterID] = st
			s.komMu.Unlock()
			return "", s.GetClusterConnState(id), bizerrors.InternalMsg(ctx, "k8s.runtime", "CheckClusterHeartbeat", fmt.Sprintf(constants.ErrFmt5d75fe17f8ef, errMsg), "cluster_id", id, "detail", errMsg)
		}
	}
	if probeErr := s.probeClusterListNamespacesKom(ctx, id); probeErr != nil {
		errMsg := classifyClusterConnectError(probeErr)
		s.komMu.Lock()
		st := s.connState[clusterID]
		st.State = "degraded"
		st.LastAttemptAt = time.Now()
		st.LastError = errMsg
		st.ConsecutiveFailures++
		s.connState[clusterID] = st
		s.komMu.Unlock()
		if _, ok := bizerrors.As(probeErr); ok {
			return "", s.GetClusterConnState(id), probeErr
		}
		return "", s.GetClusterConnState(id), bizerrors.InternalMsg(ctx, "k8s.runtime", "CheckClusterHeartbeat", errMsg, "cluster_id", id)
	}
	s.komMu.Lock()
	st := s.connState[clusterID]
	st.State = "ready"
	st.LastAttemptAt = time.Now()
	st.LastSuccessAt = time.Now()
	st.LastError = ""
	st.ConsecutiveFailures = 0
	s.connState[clusterID] = st
	s.komMu.Unlock()
	return gitVer, s.GetClusterConnState(id), nil
}

// GetClusterConnState 获取相关的业务逻辑。
func (s *K8sRuntimeService) GetClusterConnState(id uint) ClusterConnState {
	s.komMu.Lock()
	defer s.komMu.Unlock()
	key := strconv.FormatUint(uint64(id), 10)
	st := s.connState[key]
	if strings.TrimSpace(st.State) == "" {
		st.State = "unknown"
	}
	return st
}

// GetClusterRestConfig 与 GetClusterKubectl 对齐：项目归属校验、凭证意图、QPS（不使用 Impersonation）。
func (s *K8sRuntimeService) GetClusterRestConfig(ctx context.Context, id uint) (*model.K8sCluster, *rest.Config, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, k8sRepoErr(ctx, "k8s.runtime", "GetClusterRestConfig", err, "cluster_id", id)
	}
	if cluster.Status != 1 {
		return nil, nil, constants.ErrForbiddenWithMsg(constants.ErrMsgb0e556f1ccc5)
	}
	if err := s.ensureOwningProjectAccess(ctx, cluster); err != nil {
		return nil, nil, err
	}
	intent := accessIntentFromContext(ctx)
	kubeconfig, _, kerr := s.resolveKubeconfigForIntent(cluster, intent)
	if kerr != nil {
		return nil, nil, constants.ErrBadRequestWithMsg(kerr.Error())
	}
	cfg, err := restConfigFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, nil, bizerrors.Internalf(ctx, "k8s.runtime", "api", err, constants.ErrFmtd7f0c3fe8497)
	}
	return cluster, cfg, nil
}
