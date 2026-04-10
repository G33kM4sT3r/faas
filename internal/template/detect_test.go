package template

import "testing"

func TestDetect(t *testing.T) {
	tests := []struct {
		filename string
		want     string
		wantErr  bool
	}{
		{"hello.py", "python", false},
		{"main.go", "go", false},
		{"handler.rs", "rust", false},
		{"server.php", "php", false},
		{"handler.ts", "typescript", false},
		{"handler.js", "javascript", false},
		{"unknown.lua", "", true},
		{"noext", "", true},
		{"handler.GO", "go", false},
		{"handler.Py", "python", false},
		{"handler.test.go", "go", false},
		{"handler.spec.ts", "typescript", false},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got, err := Detect(tt.filename)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
