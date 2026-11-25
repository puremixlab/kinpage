package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const dWebServicesBaseURL = "https://api.dwebservices.xyz/v1"

// UpdateDWebServicesCID updates a hosted IPNS record on dWebServices.xyz with
// the newly published CID.
func UpdateDWebServicesCID(ctx context.Context, baseURL, domain, apiKey, cid string, client *http.Client) error {
	domain = strings.TrimSpace(domain)
	apiKey = strings.TrimSpace(apiKey)
	cid = strings.TrimSpace(cid)
	if domain == "" || apiKey == "" {
		return errors.New("dWebServices 域名或 API Key 不能为空")
	}
	if cid == "" {
		return errors.New("CID 不能为空")
	}
	if baseURL == "" {
		baseURL = dWebServicesBaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	payload, err := json.Marshal(map[string]string{
		"domain": domain,
		"cid":    cid,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/ipns/update", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("dWebServices 返回 HTTP %d", resp.StatusCode)
	}
	return nil
}

// SyncDWebServices checks if dWebServices.xyz is configured for siteID and
// pushes the latest published CID. Returns nil when not configured.
func SyncDWebServices(root, siteID, cid string, client *http.Client) error {
	cfg, err := LoadConfig(root, siteID, "dwebservices")
	if err != nil || cfg.Name == "" || cfg.Token == "" {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return UpdateDWebServicesCID(ctx, dWebServicesBaseURL, cfg.Name, cfg.Token, cid, client)
}
