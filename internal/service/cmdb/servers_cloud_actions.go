package cmdb

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/sshclient"

	"gorm.io/gorm"
)

func (s *Service) RunCloudServerAction(ctx context.Context, projectID, serverID uint, req CloudServerActionRequest) (*CloudServerActionResult, error) {
	if strings.TrimSpace(req.Action) == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg62812cadd1e4)
	}
	sv, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrServerNotFound
		}
		return nil, bizerrors.Pass(ctx, "cmdb", "RunCloudServerAction", err)
	}
	if sv.ProjectID != projectID {
		return nil, constants.ErrServerNotInCurrentProject
	}
	if strings.TrimSpace(sv.SourceType) != model.ServerGroupCategoryCloud && strings.TrimSpace(sv.SourceType) != "cloud" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg1f9244d53fae)
	}
	if sv.GroupID == nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg5f86cac1154b)
	}
	groupID := *sv.GroupID
	accounts, err := s.cloudAccountRepo.ListByProjectAndGroup(ctx, sv.ProjectID, &groupID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "RunCloudServerAction", err)
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
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg38ffb6c1fcca)
	}
	if account.EncAK == nil || account.EncSK == nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgedfdf2d93904)
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
		return nil, bizerrors.Pass(ctx, "cmdb", "RunCloudServerAction", err)
	}

	instances, err := provider.ListInstances(ctx, ak, sk, account.RegionScope)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgCloudSDKPrefix + err.Error())
	}
	// Ensure instance id and region present.
	instanceID := strings.TrimSpace(sv.CloudInstanceID)
	region := strings.TrimSpace(sv.CloudRegion)
	if instanceID == "" || region == "" {
		host := strings.TrimSpace(sv.Host)
		for _, ins := range instances {
			if instanceID != "" && strings.EqualFold(strings.TrimSpace(ins.InstanceID), instanceID) {
				region = strings.TrimSpace(ins.Region)
				break
			}
			if instanceID == "" && host != "" &&
				(strings.EqualFold(strings.TrimSpace(ins.Host), host) ||
					strings.EqualFold(strings.TrimSpace(ins.PublicIP), host) ||
					strings.EqualFold(strings.TrimSpace(ins.PrivateIP), host)) {
				instanceID = strings.TrimSpace(ins.InstanceID)
				region = strings.TrimSpace(ins.Region)
				break
			}
		}
	}
	if instanceID == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg7d2f738475f2)
	}
	if region == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgd5a14bd7dce0)
	}
	// Persist inferred fields for later operations.
	changed := false
	if strings.TrimSpace(sv.CloudInstanceID) == "" && instanceID != "" {
		sv.CloudInstanceID = instanceID
		changed = true
	}
	if strings.TrimSpace(sv.CloudRegion) == "" && region != "" {
		sv.CloudRegion = region
		changed = true
	}
	if changed {
		_ = s.serverRepo.Save(ctx, sv)
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case "reset_password":
		pw := strings.TrimSpace(req.NewPassword)
		if pw == "" {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgaef828b00100)
		}
		if err := provider.ResetInstancePassword(ctx, ak, sk, region, instanceID, pw); err != nil {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgCloudSDKPrefix + err.Error())
		}
		return &CloudServerActionResult{ServerID: serverID, Action: action, Message: "密码重置请求已提交"}, nil
	case "reboot":
		if err := provider.RebootInstance(ctx, ak, sk, region, instanceID); err != nil {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgCloudSDKPrefix + err.Error())
		}
		return &CloudServerActionResult{ServerID: serverID, Action: action, Message: "重启请求已提交"}, nil
	case "shutdown":
		if err := provider.ShutdownInstance(ctx, ak, sk, region, instanceID); err != nil {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgCloudSDKPrefix + err.Error())
		}
		return &CloudServerActionResult{ServerID: serverID, Action: action, Message: "关机请求已提交"}, nil
	default:
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg1707715d174a)
	}
}

func (s *Service) detectRemoteOSAndArch(ctx context.Context, cli *sshclient.Client) (string, string, string) {
	// Linux/macOS path
	if res, err := cli.Exec(ctx, "uname -s && uname -m && uname -a", 8192); err == nil && res.ExitCode == 0 {
		lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
		osType := "linux"
		if len(lines) > 0 {
			v := strings.ToLower(strings.TrimSpace(lines[0]))
			if strings.Contains(v, "darwin") {
				osType = "darwin"
			} else if strings.Contains(v, "linux") {
				osType = "linux"
			}
		}
		arch := ""
		if len(lines) > 1 {
			arch = strings.TrimSpace(lines[1])
		}
		msg := strings.TrimSpace(res.Stdout)
		return osType, arch, msg
	}

	// Windows path (OpenSSH + powershell)
	psCmd := "powershell -NoProfile -Command \"[System.Runtime.InteropServices.RuntimeInformation]::OSDescription; [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture\""
	if res, err := cli.Exec(ctx, psCmd, 8192); err == nil && res.ExitCode == 0 {
		lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
		arch := ""
		if len(lines) > 1 {
			arch = strings.TrimSpace(lines[len(lines)-1])
		}
		return "windows", arch, strings.TrimSpace(res.Stdout)
	}

	return "", "", "connected"
}
