package ipfs

import (
	"encoding/json"
	"testing"
)

func TestResolveResourceURL(t *testing.T) {
	gw := "http://127.0.0.1:8080"
	cases := []struct {
		in   string
		want string
	}{
		{"QmQZe48wGBwx6z8DuboTG62AYf8BbKd6XYDENefVZUvjyV", "http://127.0.0.1:8080/ipfs/QmQZe48wGBwx6z8DuboTG62AYf8BbKd6XYDENefVZUvjyV"},
		{"bafybeihyeuqc7nv2zfwt3x6bglgxrca2xglt26jzzqpi4zxci5czxwskku/index.html", "http://127.0.0.1:8080/ipfs/bafybeihyeuqc7nv2zfwt3x6bglgxrca2xglt26jzzqpi4zxci5czxwskku/index.html"},
		{"ipfs://bafybeihyeuqc7nv2zfwt3x6bglgxrca2xglt26jzzqpi4zxci5czxwskku", "http://127.0.0.1:8080/ipfs/bafybeihyeuqc7nv2zfwt3x6bglgxrca2xglt26jzzqpi4zxci5czxwskku"},
		{"ipns://planetable.eth/about/", "http://127.0.0.1:8080/ipns/planetable.eth/about/"},
		{"planetable.eth", "http://127.0.0.1:8080/ipns/planetable.eth"},
		{"k51qzi5uqu5dgv8kzl1anc0m74n6t9ffdjnypdh846ct5wgpljc7rulynxa74a", "http://127.0.0.1:8080/ipns/k51qzi5uqu5dgv8kzl1anc0m74n6t9ffdjnypdh846ct5wgpljc7rulynxa74a"},
	}
	for _, tc := range cases {
		got, err := ResolveResourceURL(gw, tc.in)
		if err != nil {
			t.Fatalf("ResolveResourceURL(%q) error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ResolveResourceURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	for _, bad := range []string{"", "   ", "javascript:alert(1)", "https://evil.com", "../etc/passwd"} {
		if _, err := ResolveResourceURL(gw, bad); err == nil {
			t.Fatalf("expected error for invalid resource %q", bad)
		}
	}
	if _, err := ResolveResourceURL("", "planetable.eth"); err == nil {
		t.Fatal("expected error when gateway is empty")
	}
}

func TestApplyWebUICORS(t *testing.T) {
	cfg := map[string]json.RawMessage{}
	if err := applyWebUICORS(cfg); err != nil {
		t.Fatal(err)
	}
	var api struct {
		HTTPHeaders struct {
			Origin  []string `json:"Access-Control-Allow-Origin"`
			Methods []string `json:"Access-Control-Allow-Methods"`
		} `json:"HTTPHeaders"`
	}
	if err := json.Unmarshal(cfg["API"], &api); err != nil {
		t.Fatal(err)
	}
	if len(api.HTTPHeaders.Origin) != 1 || api.HTTPHeaders.Origin[0] != "https://webui.ipfs.io" {
		t.Fatalf("unexpected CORS origin: %v", api.HTTPHeaders.Origin)
	}
	if len(api.HTTPHeaders.Methods) == 0 {
		t.Fatal("expected CORS methods to be populated")
	}
}
