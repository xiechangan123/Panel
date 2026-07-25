package biz

import "testing"

// 多域名证书的目标是逗号分隔的域名列表，用其中任一域名都应能命中规则
func TestFilterTargetMatchesAnyCertDomain(t *testing.T) {
	metrics := []*AlertMetric{
		{Target: "a.example.com,b.example.com", Value: 5},
		{Target: "c.example.com", Value: 30},
	}

	cases := []struct {
		target string
		want   int
	}{
		{"", 2},
		{"a.example.com", 1},
		{"b.example.com", 1},
		{"c.example.com", 1},
		{"a.example.com,b.example.com", 1},
		{"missing.example.com", 0},
	}

	for _, c := range cases {
		if got := filterTarget(c.target, metrics); len(got) != c.want {
			t.Fatalf("target %q matched %d metrics, want %d", c.target, len(got), c.want)
		}
	}
}
