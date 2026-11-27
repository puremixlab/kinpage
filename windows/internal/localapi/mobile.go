package localapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"windows/internal/store"
)

const mobilePrefix = "/v0/mobile/"

func (srv *Server) dispatchMobile(w http.ResponseWriter, r *http.Request) {
	if srv.backend.PublishMobile == nil {
		writeMobileError(w, http.StatusServiceUnavailable, "publishing_unavailable")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, mobilePrefix)
	if path == "sites" && r.Method == http.MethodGet {
		srv.mobileSites(w)
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) == 4 && parts[0] == "sites" && parts[2] == "articles" && r.Method == http.MethodPut {
		srv.mobilePublish(w, r, parts[1], parts[3])
		return
	}
	writeMobileError(w, http.StatusNotFound, "not_found")
}

func (srv *Server) mobileSites(w http.ResponseWriter) {
	lib, err := srv.backend.OpenStore()
	if err != nil {
		writeMobileError(w, http.StatusInternalServerError, "library_unavailable")
		return
	}
	sites, err := lib.ListSites()
	if err != nil {
		writeMobileError(w, http.StatusInternalServerError, "library_unavailable")
		return
	}
	type mobileSite struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	active := []mobileSite{}
	for _, site := range sites {
		if !site.Archived && site.ShouldPublishIPNS() {
			active = append(active, mobileSite{ID: site.ID, Name: site.Name})
		}
	}
	writeJSON(w, http.StatusOK, active)
}

func (srv *Server) mobilePublish(w http.ResponseWriter, r *http.Request, siteID, articleID string) {
	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil || json.Unmarshal(body, &input) != nil {
		writeMobileError(w, http.StatusBadRequest, "invalid_article")
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" || strings.TrimSpace(input.Content) == "" {
		writeMobileError(w, http.StatusBadRequest, "invalid_article")
		return
	}
	lib, err := srv.backend.OpenStore()
	if err != nil {
		writeMobileError(w, http.StatusInternalServerError, "library_unavailable")
		return
	}
	article, err := lib.SaveMobileArticle(siteID, articleID, input.Title, input.Content)
	switch {
	case errors.Is(err, store.ErrMobileSiteUnavailable):
		writeMobileError(w, http.StatusNotFound, "site_unavailable")
		return
	case errors.Is(err, store.ErrMobileArticleConflict):
		writeMobileError(w, http.StatusConflict, "article_conflict")
		return
	case errors.Is(err, store.ErrMobileArticleID):
		writeMobileError(w, http.StatusBadRequest, "invalid_article")
		return
	case err != nil:
		writeMobileError(w, http.StatusInternalServerError, "article_save_failed")
		return
	}
	result, err := srv.backend.PublishMobile(article)
	if err != nil || result.CID == "" {
		writeMobileError(w, http.StatusBadGateway, "publish_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"articleId": article.ID, "cid": result.CID})
}

func writeMobileError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
