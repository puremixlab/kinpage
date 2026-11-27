package localapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"windows/internal/publisher"
	"windows/internal/store"
)

func TestMobilePublicationRetry(t *testing.T) {
	root := t.TempDir()
	lib, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	site, err := lib.CreateSite("Test journal", "")
	if err != nil {
		t.Fatal(err)
	}

	attempts := 0
	srv := NewServer(Backend{
		OpenStore: func() (*store.Store, error) { return lib, nil },
		PublishMobile: func(post store.Article) (publisher.Result, error) {
			attempts++
			if post.SiteID != site.ID || post.Title != "From the phone" || post.Content != "Write offline." {
				t.Fatalf("unexpected article: %+v", post)
			}
			if attempts == 1 {
				return publisher.Result{}, errors.New("temporary node failure")
			}
			return publisher.Result{CID: "test-cid"}, nil
		},
	})
	handler := srv.buildHandler(root)
	request := func(method, path string, body io.Reader, auth bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, body)
		if auth {
			req.SetBasicAuth("Planet", "test-passcode")
		}
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		if resp.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("mobile response must be non-cacheable")
		}
		return resp
	}
	const list = "/v0/mobile/sites"
	if resp := request(http.MethodGet, list, nil, false); resp.Code != http.StatusForbidden {
		t.Fatalf("auth disabled: %d %s", resp.Code, resp.Body)
	}
	if _, err := SaveConfig(root, true, 8086, true, "Planet", "test-passcode"); err != nil {
		t.Fatal(err)
	}
	if resp := request(http.MethodGet, list, nil, false); resp.Code != http.StatusUnauthorized {
		t.Fatalf("missing credentials: %d %s", resp.Code, resp.Body)
	}
	resp := request(http.MethodGet, list, nil, true)
	var sites []map[string]string
	if resp.Code != http.StatusOK || json.Unmarshal(resp.Body.Bytes(), &sites) != nil ||
		len(sites) != 1 || len(sites[0]) != 2 || sites[0]["id"] != site.ID || sites[0]["name"] != site.Name {
		t.Fatalf("site list: %d %s", resp.Code, resp.Body)
	}

	const id = "ef62d160-8da1-41fb-b397-8b19c246086d"
	const input = `{"title":"From the phone","content":"Write offline."}`
	path := list + "/" + site.ID + "/articles/" + id
	if resp = request(http.MethodPut, path, strings.NewReader(input), true); resp.Code != http.StatusBadGateway {
		t.Fatalf("first attempt: %d %s", resp.Code, resp.Body)
	}
	resp = request(http.MethodPut, path, strings.NewReader(input), true)
	var receipt map[string]string
	if resp.Code != http.StatusOK || json.Unmarshal(resp.Body.Bytes(), &receipt) != nil ||
		receipt["articleId"] != strings.ToUpper(id) || receipt["cid"] != "test-cid" {
		t.Fatalf("retry: %d %s", resp.Code, resp.Body)
	}
	posts, err := lib.ListArticles(site.ID)
	if err != nil || len(posts) != 1 || posts[0].ID != receipt["articleId"] || attempts != 2 {
		t.Fatalf("retry did not reuse the article: %+v %v", posts, err)
	}
	if _, err := lib.SaveArticle(site.ID, posts[0].ID, "Desktop edit", posts[0].Content); err != nil {
		t.Fatal(err)
	}
	if resp = request(http.MethodPut, path, strings.NewReader(input), true); resp.Code != http.StatusConflict || attempts != 2 {
		t.Fatalf("retry after desktop edit: %d %s", resp.Code, resp.Body)
	}
}
