package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"windows/internal/sitecleanup"
)

type KeyInfo struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func (c *Client) ListKeys() ([]KeyInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out, err := c.run("key", "list", "-l")
	if err != nil {
		return nil, err
	}
	var keys []KeyInfo
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			keys = append(keys, KeyInfo{ID: fields[0], Name: fields[1]})
		}
	}
	return keys, nil
}

func (c *Client) EnsureKey(name string) (string, error) {
	if !safeName.MatchString(name) || name == "self" {
		return "", errors.New("invalid key name")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	key, err := c.lookupKey(context.Background(), name)
	if err != nil || key != "" {
		return key, err
	}
	return c.run("key", "gen", "--type=ed25519", name)
}

func (c *Client) LookupKey(name string) (string, error) {
	return c.LookupKeyContext(context.Background(), name)
}

func (c *Client) LookupKeyContext(ctx context.Context, name string) (string, error) {
	if !safeName.MatchString(name) || name == "self" {
		return "", errors.New("invalid key name")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lookupKey(ctx, name)
}

// ExportKey writes a cleartext private key to a new path chosen by the caller.
func (c *Client) ExportKey(name, path string) (result error) {
	if !safeName.MatchString(name) || name == "self" {
		return errors.New("invalid key name")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := os.Lstat(path); err == nil {
		return errors.New("key export destination exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if c.process != nil {
		offline := c.offline
		if err := c.closeLocked(); err != nil {
			return err
		}
		defer func() {
			if err := c.startLocked(offline); err != nil {
				result = errors.Join(result, fmt.Errorf("重启 Kubo 失败：%w", err))
			}
		}()
	}
	if err := c.validateKeyAPI(); err != nil {
		return err
	}
	present, err := sitecleanup.KuboRepoPresent(c.repo)
	if err != nil {
		return err
	}
	if !present {
		return errors.New("Kubo 仓库不存在")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.executable, "--offline", "key", "export", name, "-o", path)
	c.configure(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("导出密钥失败：%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (c *Client) ImportKey(name, path string) (string, error) {
	if !safeName.MatchString(name) || name == "self" {
		return "", errors.New("invalid key name")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if present, err := sitecleanup.KuboRepoPresent(c.repo); err != nil || !present {
		if err != nil {
			return "", err
		}
		return "", errors.New("Kubo 仓库不存在")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return "", errors.New("导入密钥文件无效")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !actual.Mode().IsRegular() || !os.SameFile(info, actual) {
		return "", errors.New("导入密钥文件已改变")
	}
	key, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(key) > 1<<20 {
		return "", errors.New("导入密钥文件无效")
	}
	if c.process == nil {
		if err = c.validateKeyAPI(); err != nil {
			return "", err
		}
		if err = sitecleanup.CheckOfflineKuboRepo(c.repo); err != nil {
			return "", err
		}
		if err = c.startLocked(true); err != nil {
			return "", err
		}
		defer func() { _ = c.closeLocked() }()
	}
	if err = c.validateKeyAPI(); err != nil {
		return "", err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "key")
	if err != nil {
		return "", err
	}
	if _, err = part.Write(key); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	port := strings.TrimPrefix(c.ownedAPI, "/ip4/127.0.0.1/tcp/")
	address := "http://127.0.0.1:" + port + "/api/v0/key/import?" + url.Values{"arg": []string{name}}.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, address, &body)
	if err != nil {
		return "", errors.New("导入密钥请求无效")
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("导入密钥请求失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("Kubo 未接受导入密钥")
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, (64<<10)+1))
	if err != nil || len(b) > 64<<10 {
		return "", errors.New("导入密钥响应无效")
	}
	var result struct {
		ID   string `json:"Id"`
		Name string `json:"Name"`
	}
	if json.Unmarshal(b, &result) != nil || result.Name != name || !safeName.MatchString(result.ID) {
		return "", errors.New("导入密钥响应无效")
	}
	return result.ID, nil
}

func (c *Client) RemoveKey(name string) error {
	if !safeName.MatchString(name) || name == "self" {
		return errors.New("invalid key name")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.run("key", "rm", name)
	return err
}

func (c *Client) KeyMatches(name, identity string) (bool, error) {
	if !safeName.MatchString(name) || name == "self" || !safeName.MatchString(identity) {
		return false, errors.New("invalid key name or identity")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.keyMatches(name, identity)
}

func (c *Client) keyMatches(name, identity string) (bool, error) {
	for _, base := range []string{"base36", "b58mh", "base32"} {
		list, err := c.run(c.ownedAPIArgs("key", "list", "-l", "--ipns-base="+base)...)
		if err != nil {
			return false, err
		}
		for _, line := range strings.Split(list, "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[1] == name && fields[0] == identity {
				return true, nil
			}
		}
	}
	return false, nil
}

// ValidateKeyAPI prevents a repository's api file from routing destructive key
// commands to a daemon that this client did not start.
func (c *Client) ValidateKeyAPI() error { c.mu.Lock(); defer c.mu.Unlock(); return c.validateKeyAPI() }

func (c *Client) validateKeyAPI() error {
	path := filepath.Join(c.repo, "api")
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if c.process != nil {
			select {
			case <-c.exited:
				c.clearDaemonState()
			default:
				return errors.New("运行中的 Kubo API 地址缺失")
			}
		}
		return nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > 512 {
		return errors.New("Kubo API 地址文件无效")
	}
	if c.process == nil || c.ownedAPI == "" {
		return errors.New("Kubo API 不属于本程序，不能删除密钥")
	}
	select {
	case <-c.exited:
		c.clearDaemonState()
		return errors.New("Kubo 服务已停止，不能核对 API")
	default:
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !actual.Mode().IsRegular() || !os.SameFile(info, actual) {
		return errors.New("Kubo API 地址文件已改变")
	}
	b, err := io.ReadAll(io.LimitReader(f, 513))
	if err != nil || len(b) > 512 || strings.TrimSpace(string(b)) != c.ownedAPI {
		return errors.New("Kubo API 地址与本程序启动的服务不一致")
	}
	return nil
}

// RemoveKeyIfMatches makes retries safe and refuses to delete an unrelated key.
func (c *Client) RemoveKeyIfMatches(name, identity string) error {
	if !safeName.MatchString(name) || name == "self" || (identity != "" && !safeName.MatchString(identity)) {
		return errors.New("invalid key name or identity")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if present, err := sitecleanup.KuboRepoPresent(c.repo); err != nil || !present {
		if err != nil {
			return err
		}
		return errors.New("Kubo 仓库不存在")
	}
	if c.process == nil || c.ownedAPI == "" {
		return errors.New("必须使用本程序启动并核验的 Kubo 服务删除密钥")
	}
	if err := c.validateKeyAPI(); err != nil {
		return err
	}
	existing, err := c.lookupKey(context.Background(), name)
	if err != nil {
		return err
	}
	if existing == "" {
		return nil
	}
	if identity == "" {
		return errors.New("站点密钥身份未知，未删除")
	}
	match, err := c.keyMatches(name, identity)
	if err != nil {
		return err
	}
	if !match {
		return errors.New("站点密钥与原 IPNS 不匹配，未删除")
	}
	_, err = c.run(c.ownedAPIArgs("key", "rm", name)...)
	return err
}

func (c *Client) lookupKey(ctx context.Context, name string) (string, error) {
	list, err := c.runContext(ctx, 15*time.Second, c.ownedAPIArgs("key", "list", "-l")...)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(list, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			return fields[0], nil
		}
	}
	return "", nil
}
