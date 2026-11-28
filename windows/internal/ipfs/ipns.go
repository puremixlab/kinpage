package ipfs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PublishOffline creates a local IPNS record; it does not publish to the network.
func (c *Client) PublishOffline(key, cid string) (string, error) {
	if !safeName.MatchString(key) || !safeName.MatchString(cid) {
		return "", errors.New("invalid key or CID")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.run("--offline", "name", "publish", "--allow-offline", "--quieter", "--key="+key, "/ipfs/"+cid)
}

// PublishOnline asks the running daemon to publish an IPNS record via loopback RPC.
func (c *Client) PublishOnline(key, cid string) (string, error) {
	return c.PublishOnlineContext(context.Background(), key, cid)
}

func (c *Client) PublishOnlineContext(ctx context.Context, key, cid string) (string, error) {
	if !safeName.MatchString(key) || !safeName.MatchString(cid) {
		return "", errors.New("invalid key or CID")
	}
	snap := c.snapshot()
	if !snap.running {
		return "", errors.New("Kubo daemon is not running")
	}
	if !snap.online {
		return "", errors.New("Kubo 在线节点未运行")
	}
	query := url.Values{
		"arg":      []string{"/ipfs/" + cid},
		"key":      []string{key},
		"lifetime": []string{"7200h"},
	}
	var result struct {
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}
	if err := postRPCAPIContext(ctx, snap.api, 3*time.Minute, "/api/v0/name/publish?"+query.Encode(), 64<<10, &result); err != nil {
		return "", err
	}
	if !safeName.MatchString(result.Name) || result.Value != "/ipfs/"+cid {
		return "", errors.New("Kubo 发布响应无效")
	}
	return result.Name, nil
}

const MaxIPNSRecordBytes = 2048

// LocalIPNSRecord retrieves the raw signed IPNS record bytes for /ipns/<ipns>
// from the running online daemon's routing table (Type 5 entry in routing/get).
func (c *Client) LocalIPNSRecord(ipns string) ([]byte, error) {
	return c.LocalIPNSRecordContext(context.Background(), ipns)
}

func (c *Client) LocalIPNSRecordContext(parent context.Context, ipns string) ([]byte, error) {
	if !safeName.MatchString(ipns) {
		return nil, errors.New("invalid IPNS name")
	}
	snap := c.snapshot()
	if !snap.running {
		return nil, errors.New("Kubo daemon is not running")
	}
	if !snap.online {
		return nil, errors.New("Kubo 在线节点未运行")
	}
	if !loopbackAPI.MatchString(snap.api) {
		return nil, errors.New("Kubo API 地址无效")
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	port := strings.TrimPrefix(snap.api, "/ip4/127.0.0.1/tcp/")
	query := url.Values{"arg": []string{"/ipns/" + ipns}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://127.0.0.1:"+port+"/api/v0/routing/get?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	client := http.Client{
		Transport:     &http.Transport{Proxy: nil},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	defer client.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, formatRPCError("/api/v0/routing/get", resp.StatusCode, resp.Body)
	}
	const maxRoutingBytes = 256 << 10
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxRoutingBytes+1))
	if err != nil {
		return nil, fmt.Errorf("无法读取 Kubo 路由记录: %w", err)
	}
	if len(b) > maxRoutingBytes {
		return nil, errors.New("Kubo 路由响应超过大小限制")
	}
	return extractIPNSRecordFromRoutingGet(b)
}

func extractIPNSRecordFromRoutingGet(data []byte) ([]byte, error) {
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var entry struct {
			Type  int    `json:"Type"`
			Extra string `json:"Extra"`
		}
		if json.Unmarshal(line, &entry) != nil || entry.Type != 5 || entry.Extra == "" {
			continue
		}
		record, err := base64.StdEncoding.DecodeString(entry.Extra)
		if err != nil || len(record) == 0 || len(record) > MaxIPNSRecordBytes {
			continue
		}
		return record, nil
	}
	return nil, errors.New("未找到本地签名的 IPNS 记录")
}

func (c *Client) ResolveOffline(name string) (string, error) {
	if !safeName.MatchString(name) {
		return "", errors.New("invalid IPNS name")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.run("--offline", "name", "resolve", "/ipns/"+name)
}

// Resolve resolves an IPNS peer ID or DNSLink/ENS name to its root CID.
func (c *Client) Resolve(name string) (string, error) {
	return c.ResolveContext(context.Background(), name)
}

func (c *Client) ResolveContext(ctx context.Context, name string) (string, error) {
	if name == "" || len(name) > 255 || strings.ContainsAny(name, `/\: `) {
		return "", errors.New("IPNS 名称无效")
	}
	value := ""
	var err error
	target := "/ipns/" + name
	if strings.HasSuffix(strings.ToLower(name), ".eth") {
		value, err = resolveENS(ctx, ensRPC(), name)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(value, "/ipns/") {
			target = value
			value = ""
		}
	}
	if value == "" {
		c.mu.Lock()
		value, err = c.runContext(ctx, 90*time.Second, "name", "resolve", "--nocache", target)
		c.mu.Unlock()
	}
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(value, "/ipfs/") {
		return "", errors.New("站点解析结果不是 IPFS 内容")
	}
	resolved := strings.TrimPrefix(value, "/ipfs/")
	parts := strings.SplitN(resolved, "/", 2)
	if !safeName.MatchString(parts[0]) || (len(parts) > 1 && !validContentPath(parts[0], parts[1])) {
		return "", errors.New("站点解析路径无效")
	}
	return resolved, nil
}
