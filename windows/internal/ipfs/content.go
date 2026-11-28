package ipfs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

func (c *Client) Add(dir string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.run("add", "-Qr", dir)
}

// Read fetches a bounded public file without using an HTTP gateway.
func (c *Client) Read(cid, path string) ([]byte, error) {
	return c.ReadContext(context.Background(), cid, path)
}

func (c *Client) ReadContext(parent context.Context, cid, path string) ([]byte, error) {
	if !validContentPath(cid, path) {
		return nil, errors.New("IPFS 内容路径无效")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := parent.Err(); err != nil {
		return nil, err
	}
	temporary, err := c.ensureCommandDaemon()
	if err != nil {
		return nil, err
	}
	if temporary {
		defer func() { _ = c.closeLocked() }()
	}
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
	return c.PinContext(context.Background(), cid)
}

func (c *Client) PinContext(ctx context.Context, cid string) error {
	if !safeName.MatchString(cid) {
		return errors.New("CID 无效")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.runContext(ctx, 3*time.Minute, "pin", "add", "--progress=false", "/ipfs/"+cid)
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
