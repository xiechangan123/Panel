package middleware

import "testing"

func TestTrimEntranceAPIPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		entrance string
		wantPath string
		wantOK   bool
	}{
		{name: "root api", path: "/api/user/info", entrance: "/", wantPath: "/api/user/info", wantOK: true},
		{name: "root api exact", path: "/api", entrance: "/", wantPath: "/api", wantOK: true},
		{name: "root api prefix confusion", path: "/apiary", entrance: "/", wantOK: false},
		{name: "entrance api", path: "/secret/api/user/info", entrance: "/secret", wantPath: "/api/user/info", wantOK: true},
		{name: "entrance api exact", path: "/secret/api", entrance: "/secret", wantPath: "/api", wantOK: true},
		{name: "entrance page", path: "/secret/login", entrance: "/secret", wantOK: false},
		{name: "entrance prefix confusion", path: "/secretx/api/user/info", entrance: "/secret", wantOK: false},
		{name: "api prefix confusion", path: "/secret/apiary", entrance: "/secret", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotOK := trimEntranceAPIPath(tt.path, tt.entrance)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}
