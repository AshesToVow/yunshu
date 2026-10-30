package repository

import (
	"context"
	"testing"

	"yunshu/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newCicdRunsTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared&_pragma=busy_timeout(5000)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// gorm.io/driver/sqlite 依赖 CGO；Windows/CI 常以 CGO_ENABLED=0 编译。
		t.Skipf("sqlite unavailable: %v", err)
	}
	if err := db.AutoMigrate(&model.CicdBuildRun{}); err != nil {
		t.Fatal(err)
	}
	return db
}

// 镜像浏览的「关联构建」必须忽略 registry 主机别名（Harbor 域名 / IP / 项目级 harbor_url 覆盖），
// 且 tag 精确匹配，不能因为前缀相同（go123 vs go123-v2）或 LIKE 通配符（_）误关联。
func TestListBuildRunsByImageAddress_HostAliasAndExactTag(t *testing.T) {
	t.Parallel()
	db := newCicdRunsTestDB(t, "cicd_runs_image_link")
	repo := NewCicdRepository(db)
	ctx := context.Background()

	seed := []model.CicdBuildRun{
		{ProjectID: 1, ServiceID: 1, BuildNumber: 101, ImageAddress: "harbor.deploy.local/registry/kube-consul-register:go123"},
		{ProjectID: 1, ServiceID: 1, BuildNumber: 102, ImageAddress: "harbor.deploy.local/registry/kube-consul-register:go123-v2"},
		{ProjectID: 1, ServiceID: 1, BuildNumber: 103, ImageAddress: "harbor.deploy.local/registry/other-service:go123"},
		{ProjectID: 1, ServiceID: 1, BuildNumber: 104, ImageAddress: "harbor.deploy.local/registry/kube-consul-register:latest_prod_2026"},
		{ProjectID: 1, ServiceID: 1, BuildNumber: 105, ImageAddress: "harbor.deploy.local/registry/kube-consul-register:latest-prod-2026"},
		{ProjectID: 1, ServiceID: 1, BuildNumber: 106, ImageAddress: "registry/kube-consul-register:go121"},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		query string
		want  []int
	}{
		{
			name:  "IP 与域名别名可关联",
			query: "10.10.10.103/registry/kube-consul-register:go123",
			want:  []int{101},
		},
		{
			name:  "带 scheme 的查询地址可关联",
			query: "https://harbor.deploy.local/registry/kube-consul-register:go123-v2",
			want:  []int{102},
		},
		{
			name:  "下划线按字面量匹配",
			query: "10.10.10.103/registry/kube-consul-register:latest_prod_2026",
			want:  []int{104},
		},
		{
			name:  "构建记录未带主机名时按路径后缀关联",
			query: "10.10.10.103/registry/kube-consul-register:go121",
			want:  []int{106},
		},
		{
			name:  "同 tag 不同仓库不关联",
			query: "10.10.10.103/registry/missing-service:go123",
			want:  nil,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows, err := repo.ListBuildRunsByImageAddress(ctx, tc.query, 20)
			if err != nil {
				t.Fatalf("query: %v", err)
			}
			got := make([]int, 0, len(rows))
			for _, r := range rows {
				got = append(got, r.BuildNumber)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("want build numbers %v, got %v", tc.want, got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("want build numbers %v, got %v", tc.want, got)
				}
			}
		})
	}
}
