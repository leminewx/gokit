package tree

import "testing"

func TestTreeInterface(t *testing.T) {
	var tree Tree[string] = NewTrie[string]()

	tree.Insert("/api/:id", "detail")
	tree.Insert("/api/list", "list")

	tests := []struct {
		name       string
		path       string
		wantValue  string
		wantParams map[string]string
		wantExists bool
	}{
		{"静态路径", "/api/list", "list", map[string]string{}, true},
		{"参数路径", "/api/42", "detail", map[string]string{"id": "42"}, true},
		{"未注册路径", "/api", "", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, params, exists := tree.Search(tt.path)
			if exists != tt.wantExists {
				t.Fatalf("Search(%q) exists = %v, want %v", tt.path, exists, tt.wantExists)
			}
			if value != tt.wantValue {
				t.Fatalf("Search(%q) value = %q, want %q", tt.path, value, tt.wantValue)
			}
			for key, want := range tt.wantParams {
				if params[key] != want {
					t.Fatalf("Search(%q) params[%q] = %q, want %q", tt.path, key, params[key], want)
				}
			}
		})
	}
}
