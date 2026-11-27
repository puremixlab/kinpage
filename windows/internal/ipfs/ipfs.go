// Package ipfs provides a Kubo command boundary using a dedicated repository.
package ipfs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var safeName = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
var loopbackAPI = regexp.MustCompile(`^/ip4/127\.0\.0\.1/tcp/[0-9]{1,5}$`)

type Client struct {
	mu           sync.Mutex
	stateMu      sync.Mutex
	executable   string
	repo         string
	process      *exec.Cmd
	exited       chan error
	ownedAPI     string
	ownedGateway string
	offline      bool
}

func New(executable, repo string) *Client { return &Client{executable: executable, repo: repo} }

func (c *Client) run(args ...string) (string, error) {
	return c.runTimeout(3*time.Minute, args...)
}

func (c *Client) runTimeout(timeout time.Duration, args ...string) (string, error) {
	return c.runContext(context.Background(), timeout, args...)
}

func (c *Client) runContext(parent context.Context, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	// Never let the Kubo CLI infer a daemon from a mutable repository api file.
	temporary, err := c.ensureCommandDaemon()
	if err != nil {
		return "", err
	}
	if temporary {
		defer func() { _ = c.closeLocked() }()
	}
	if len(args) > 0 && strings.HasPrefix(args[0], "--api=") && args[0] != "--api="+c.ownedAPI {
		return "", errors.New("Kubo API 与本程序启动的服务不一致")
	}
	if len(args) == 0 || !strings.HasPrefix(args[0], "--api=") {
		args = append([]string{"--api=" + c.ownedAPI}, args...)
	}
	cmd := exec.CommandContext(ctx, c.executable, args...)
	c.configure(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Kubo: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) configure(cmd *exec.Cmd) {
	for _, env := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(env), "IPFS_PATH=") {
			cmd.Env = append(cmd.Env, env)
		}
	}
	cmd.Env = append(cmd.Env, "IPFS_PATH="+c.repo)
	hideWindow(cmd)
}

func (c *Client) ownedAPIArgs(args ...string) []string {
	if c.process != nil && c.ownedAPI != "" {
		return append([]string{"--api=" + c.ownedAPI}, args...)
	}
	return args
}

func (c *Client) Add(dir string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.run("add", "-Qr", dir)
}

// Read fetches a bounded public file without using an HTTP gateway.
func (c *Client) Read(cid, path string) ([]byte, error) {
	if !validContentPath(cid, path) {
		return nil, errors.New("IPFS 内容路径无效")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	temporary, err := c.ensureCommandDaemon()
	if err != nil {
		return nil, err
	}
	if temporary {
		defer func() { _ = c.closeLocked() }()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.executable, "--api="+c.ownedAPI, "cat", "/ipfs/"+cid+"/"+path)
	c.configure(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	const limit = 16 << 20
	data, readErr := io.ReadAll(io.LimitReader(stdout, limit+1))
	if len(data) > limit {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if len(data) > limit {
		return nil, errors.New("站点元数据超过 16 MB")
	}
	if readErr != nil {
		return nil, readErr
	}
	if waitErr != nil {
		return nil, fmt.Errorf("Kubo: %s: %w", strings.TrimSpace(stderr.String()), waitErr)
	}
	return data, nil
}

// Pin stores a full immutable site in the local repository for offline use.
func (c *Client) Pin(cid string) error {
	if !safeName.MatchString(cid) {
		return errors.New("CID 无效")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.run("pin", "add", "--progress=false", "/ipfs/"+cid)
	return err
}

func validContentPath(cid, path string) bool {
	if !safeName.MatchString(cid) || path == "" || strings.Contains(path, `\`) {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `<>:"|?*`) {
			return false
		}
	}
	return true
}

// CopyContent streams a public IPFS file into an already-open private temporary file.
func (c *Client) CopyContent(cid, path string, out *os.File) error {
	if !validContentPath(cid, path) || out == nil {
		return errors.New("IPFS 内容路径无效")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	temporary, err := c.ensureCommandDaemon()
	if err != nil {
		return err
	}
	if temporary {
		defer func() { _ = c.closeLocked() }()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.executable, "--api="+c.ownedAPI, "cat", "/ipfs/"+cid+"/"+path)
	c.configure(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		return err
	}
	const maxBytes int64 = 1 << 30
	n, copyErr := io.Copy(out, io.LimitReader(stdout, maxBytes+1))
	if n > maxBytes || copyErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if n > maxBytes {
		return errors.New("音频文件超过 1 GiB")
	}
	if copyErr != nil {
		return copyErr
	}
	if waitErr != nil {
		return fmt.Errorf("Kubo: %s: %w", strings.TrimSpace(stderr.String()), waitErr)
	}
	return nil
}
