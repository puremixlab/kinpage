package localapi

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"windows/internal/store"
)

func TestLocalAPIV0IDWebClientAndSitePublic(t *testing.T) {
	root := t.TempDir()
	lib, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	site, err := lib.CreateSite("Web Site", "About")
	if err != nil {
		t.Fatal(err)
	}
	siteDir := filepath.Join(root, "sites", site.ID)
	if err := os.MkdirAll(siteDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "index.html"), []byte("<h1>Hello Public</h1>"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := SaveConfig(root, true, 18086, false, "Planet", ""); err != nil {
		t.Fatal(err)
	}
	srv := NewServer(Backend{
		OpenStore: func() (*store.Store, error) { return lib, nil },
	})
	if err := srv.Sync(root); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	st, err := srv.Status(root)
	if err != nil || !st.Running || st.Port == 0 {
		t.Fatalf("expected running server, got %+v err=%v", st, err)
	}
	base := "http://127.0.0.1:" + strconvItoa(st.Port)

	// 1. GET /v0/id
	resp, err := http.Get(base + "/v0/id")
	if err != nil {
		t.Fatal(err)
	}
	var idPayload map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&idPayload)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || idPayload["webClient"] != "/app/" {
		t.Fatalf("unexpected /v0/id response: status=%d body=%+v", resp.StatusCode, idPayload)
	}

	// 2. GET /app/ (built-in Web Client)
	resp, err = http.Get(base + "/app/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Kinpage Web Client") {
		t.Fatalf("unexpected /app/ response: status=%d", resp.StatusCode)
	}

	// 3. GET /v0/planets/my/:id/public/
	resp, err = http.Get(base + "/v0/planets/my/" + site.ID + "/public/")
	if err != nil {
		t.Fatal(err)
	}
	pubBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(pubBody), "Hello Public") {
		t.Fatalf("unexpected /public/ response: status=%d body=%s", resp.StatusCode, string(pubBody))
	}
}

func strconvItoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
