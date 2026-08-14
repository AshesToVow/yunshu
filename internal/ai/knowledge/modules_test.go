package knowledge

import "testing"

func TestInferModules(t *testing.T) {
	cases := []struct {
		q    string
		want string
	}{
		{"Pod CrashLoop 怎么排", ModuleK8s},
		{"构建失败看日志", ModuleCICD},
		{"告警没收到", ModuleAlert},
		{"项目日志查不到", ModuleLog},
		{"磁盘快满了", ModuleLinux},
	}
	for _, tc := range cases {
		got := InferModules(tc.q)
		ok := false
		for _, m := range got {
			if m == tc.want {
				ok = true
				break
			}
		}
		if !ok {
			t.Fatalf("query=%q want module %s in %v", tc.q, tc.want, got)
		}
	}
}

func TestModuleDocsNotEmpty(t *testing.T) {
	docs := ModuleDocs()
	if len(docs) < 10 {
		t.Fatalf("expected playbooks+modules, got %d", len(docs))
	}
	for _, d := range docs {
		if d.Module == "" || d.Source == "" || d.Content == "" {
			t.Fatalf("incomplete doc: %+v", d)
		}
	}
}
