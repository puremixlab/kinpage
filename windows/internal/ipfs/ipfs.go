// Package ipfs provides a Kubo command boundary using a dedicated repository.
package ipfs

import (
	"context"
	"errors"
	"fmt"
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
