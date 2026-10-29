package logplatform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strconv"
	"strings"

	"yunshu/internal/pkg/sshclient"
)

// 上传中的临时文件后缀：先写临时文件再 mv 原子替换。
// 直接 Create 目标路径会先把正在运行的旧二进制清空，上传中途失败就只剩半个 ELF。
const loggieBinaryUploadPart = ".loggie.uploading"

// 上传所需空间之外的余量，避免把分区撑到 100% 后 systemd/journald 也写不进去。
const loggieUploadSpaceReserve = int64(16 << 20)

// pushLoggieBinary 把离线二进制推送到目标机。
// 三步：远端校验和相同则跳过；磁盘空间不足直接给出可执行的提示（SFTP 对 ENOSPC/EDQUOT
// 只回 SSH_FX_FAILURE，看不出原因）；正式写入走临时文件 + mv，失败时旧二进制不受影响。
func pushLoggieBinary(ctx context.Context, cli *sshclient.Client, remotePath string, data []byte) (string, error) {
	sum := sha256.Sum256(data)
	want := hex.EncodeToString(sum[:])

	if remoteSHA256(ctx, cli, remotePath) == want {
		return fmt.Sprintf("binary unchanged (sha256 %s), skip upload\n", want[:12]), nil
	}
	dir := path.Dir(remotePath)
	if err := ensureRemoteSpace(ctx, cli, dir, int64(len(data))); err != nil {
		return "", err
	}

	part := remotePath + loggieBinaryUploadPart
	if err := cli.UploadBytes(ctx, part, data, 0o600); err != nil {
		sftpErr := err
		// SFTP 子系统缺失/受限时退回 SSH exec 通道；顺带把 shell 的 ENOSPC 原文带出来。
		if streamErr := streamFileViaExec(ctx, cli, part, data); streamErr != nil {
			_ = cli.RemoveRemoteFile(part)
			return "", fmt.Errorf("%s；SFTP 失败原因: %v；SSH 通道重试: %v", remoteSpaceHint(ctx, cli, dir), sftpErr, streamErr)
		}
	}
	res, err := cli.Exec(ctx, fmt.Sprintf("chmod 0755 %q && mv -f %q %q", part, part, remotePath), 4096)
	if err != nil {
		_ = cli.RemoveRemoteFile(part)
		return "", err
	}
	if res.ExitCode != 0 {
		_ = cli.RemoveRemoteFile(part)
		return "", fmt.Errorf("替换 %s 失败: %s", remotePath, strings.TrimSpace(res.Stdout+res.Stderr))
	}
	return fmt.Sprintf("uploaded binary -> %s (%d bytes)\n", remotePath, len(data)), nil
}

func remoteSHA256(ctx context.Context, cli *sshclient.Client, remotePath string) string {
	if cli == nil {
		return ""
	}
	res, err := cli.Exec(ctx, fmt.Sprintf(
		`[ -f %q ] && command -v sha256sum >/dev/null 2>&1 && sha256sum %q 2>/dev/null | awk '{print $1}'`,
		remotePath, remotePath), 4096)
	if err != nil || res.ExitCode != 0 {
		return ""
	}
	return strings.TrimSpace(res.Stdout)
}

func ensureRemoteSpace(ctx context.Context, cli *sshclient.Client, dir string, need int64) error {
	avail, ok := remoteFreeBytes(ctx, cli, dir)
	if !ok {
		// df 不可用时不阻断上传，让真正的写入错误暴露出来
		return nil
	}
	return checkRemoteSpace(dir, avail, need)
}

func checkRemoteSpace(dir string, avail, need int64) error {
	if avail >= need+loggieUploadSpaceReserve {
		return nil
	}
	return fmt.Errorf("目标机 %s 可用空间不足：剩余 %.1f MB，上传需要 %.1f MB（另需 %d MB 余量），请清理该分区后重试",
		dir, megaBytes(avail), megaBytes(need), loggieUploadSpaceReserve>>20)
}

func remoteSpaceHint(ctx context.Context, cli *sshclient.Client, dir string) string {
	avail, ok := remoteFreeBytes(ctx, cli, dir)
	if !ok {
		return fmt.Sprintf("上传失败，请检查目标机 %s 的磁盘空间与配额（df -h、quota -s）", dir)
	}
	return fmt.Sprintf("上传失败：目标机 %s 剩余 %.1f MB，请检查磁盘空间与配额（df -h、quota -s）", dir, megaBytes(avail))
}

func remoteFreeBytes(ctx context.Context, cli *sshclient.Client, dir string) (int64, bool) {
	if cli == nil {
		return 0, false
	}
	res, err := cli.Exec(ctx, fmt.Sprintf(`df -Pk %q 2>/dev/null | awk 'NR==2{print $4}'`, dir), 1024)
	if err != nil || res.ExitCode != 0 {
		return 0, false
	}
	kb, err := strconv.ParseInt(strings.TrimSpace(res.Stdout), 10, 64)
	if err != nil || kb <= 0 {
		return 0, false
	}
	return kb * 1024, true
}

func streamFileViaExec(ctx context.Context, cli *sshclient.Client, remotePath string, data []byte) error {
	if cli == nil {
		return fmt.Errorf("ssh client nil")
	}
	res, err := cli.ExecWithStdin(ctx, fmt.Sprintf("umask 077; cat > %q", remotePath), bytes.NewReader(data), 4096)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		msg := strings.TrimSpace(res.Stderr)
		if msg == "" {
			msg = strings.TrimSpace(res.Stdout)
		}
		if msg == "" {
			msg = fmt.Sprintf("exit=%d", res.ExitCode)
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func megaBytes(v int64) float64 {
	return float64(v) / (1 << 20)
}
