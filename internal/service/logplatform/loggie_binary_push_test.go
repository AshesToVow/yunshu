package logplatform

import (
	"strings"
	"testing"
)

// 分区剩余空间不足时必须在上传前拦住：SFTP 对 ENOSPC/EDQUOT 只回 SSH_FX_FAILURE，
// 用户在页面上只能看到 sftp: "Failure"。
func TestCheckRemoteSpace(t *testing.T) {
	t.Parallel()
	const need = int64(73 << 20)
	cases := []struct {
		name    string
		avail   int64
		wantErr bool
	}{
		{"刚够二进制本身", need, true},
		{"留不足余量", need + loggieUploadSpaceReserve - 1, true},
		{"刚好留足余量", need + loggieUploadSpaceReserve, false},
		{"空间充裕", 10 << 30, false},
	}
	for _, tc := range cases {
		err := checkRemoteSpace("/export/loggie", tc.avail, need)
		if tc.wantErr != (err != nil) {
			t.Fatalf("%s: err=%v wantErr=%v", tc.name, err, tc.wantErr)
		}
		if err != nil && !strings.Contains(err.Error(), "/export/loggie") {
			t.Fatalf("%s: error should name the directory: %v", tc.name, err)
		}
	}
}
