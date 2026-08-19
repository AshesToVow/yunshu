package eventforward

import "testing"

func TestNormalizeK8sEventForwardRule(t *testing.T) {
	t.Parallel()
	enabled := true
	cases := []struct {
		name    string
		req     K8sEventForwardRuleUpsertRequest
		wantErr bool
		wantNS  string
	}{
		{
			name: "empty_filters_become_json_array",
			req: K8sEventForwardRuleUpsertRequest{
				Name:       "prod-warning-forward",
				ClusterIDs: "1",
				Enabled:    &enabled,
			},
			wantNS: "[]",
		},
		{
			name: "keeps_namespace_array",
			req: K8sEventForwardRuleUpsertRequest{
				Name:           "ns-rule",
				ClusterIDs:     "1,2",
				RuleNamespaces: `["yunshu-logging"]`,
				RuleNames:      `["yunshu-loggie"]`,
				RuleReasons:    `["Failed","BackOff"]`,
			},
			wantNS: `["yunshu-logging"]`,
		},
		{
			name:    "rejects_invalid_json",
			req:     K8sEventForwardRuleUpsertRequest{Name: "bad", ClusterIDs: "1", RuleNamespaces: "yunshu-logging"},
			wantErr: true,
		},
		{
			name:    "requires_name",
			req:     K8sEventForwardRuleUpsertRequest{ClusterIDs: "1"},
			wantErr: true,
		},
		{
			name:    "requires_cluster",
			req:     K8sEventForwardRuleUpsertRequest{Name: "x"},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeK8sEventForwardRule(tc.req)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.RuleNamespaces != tc.wantNS {
				t.Fatalf("namespaces=%q want %q", got.RuleNamespaces, tc.wantNS)
			}
		})
	}
}
