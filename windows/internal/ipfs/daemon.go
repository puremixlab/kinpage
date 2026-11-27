package ipfs

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"windows/internal/sitecleanup"
)

type daemonSnapshot struct {
	running bool
	online  bool
	api     string
	gateway string
}

func (c *Client) clearDaemonState() {
	c.stateMu.Lock()
	c.process = nil
	c.ownedAPI = ""
	c.ownedGateway = ""
	c.offline = false
	c.stateMu.Unlock()
}

func (c *Client) setDaemonState(process *exec.Cmd, exited chan error, offline bool, api, gateway string) {
	c.stateMu.Lock()
	c.process = process
	c.exited = exited
	c.offline = offline
	c.ownedAPI = api
	c.ownedGateway = gateway
	c.stateMu.Unlock()
}

func (c *Client) snapshot() daemonSnapshot {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if c.process == nil || c.ownedAPI == "" {
		return daemonSnapshot{}
	}
	select {
	case <-c.exited:
		c.process = nil
		c.ownedAPI = ""
		c.ownedGateway = ""
		c.offline = false
		return daemonSnapshot{}
	default:
		return daemonSnapshot{
			running: true,
			online:  !c.offline,
			api:     c.ownedAPI,
			gateway: c.ownedGateway,
		}
	}
}

func (c *Client) Alive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshot().running
}

func (c *Client) ensureCommandDaemon() (bool, error) {
	if present, err := sitecleanup.KuboRepoPresent(c.repo); err != nil || !present {
		if err != nil {
			return false, err
		}
		return false, errors.New("Kubo 仓库不存在")
	}
	if c.process != nil {
		if c.ownedAPI == "" {
			return false, errors.New("Kubo API 身份未经核验")
		}
		return false, nil
	}
	if err := c.validateKeyAPI(); err != nil {
		return false, err
	}
	if err := sitecleanup.CheckOfflineKuboRepo(c.repo); err != nil {
		return false, err
	}
	if err := c.startLocked(true); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Client) Init() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.process != nil {
		return errors.New("Kubo 正在运行，不能修改仓库配置")
	}
	repoInfo, repoErr := os.Lstat(c.repo)
	if repoErr == nil {
		if !repoInfo.IsDir() || repoInfo.Mode()&os.ModeSymlink != 0 {
			return errors.New("Kubo 仓库目录无效")
		}
	} else if !errors.Is(repoErr, os.ErrNotExist) {
		return repoErr
	}
	if err := c.validateKeyAPI(); err != nil {
		return err
	}
	path := filepath.Join(c.repo, "config")
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, c.executable, "--offline", "init", "--empty-repo")
		c.configure(cmd)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Kubo 初始化失败：%s: %w", strings.TrimSpace(string(out)), err)
		}
	} else if err != nil {
		return err
	}
	if present, err := sitecleanup.KuboRepoPresent(c.repo); err != nil || !present {
		if err != nil {
			return err
		}
		return errors.New("Kubo 仓库不存在")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 16<<20 {
		return errors.New("Kubo 配置无效")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	actual, err := f.Stat()
	if err != nil || !actual.Mode().IsRegular() || !os.SameFile(info, actual) {
		f.Close()
		return errors.New("Kubo 配置已改变")
	}
	b, err := io.ReadAll(io.LimitReader(f, (16<<20)+1))
	closeErr := f.Close()
	if err != nil || closeErr != nil || len(b) > 16<<20 {
		return errors.New("Kubo 配置读取失败")
	}
	var config map[string]json.RawMessage
	if json.Unmarshal(b, &config) != nil || config == nil {
		return errors.New("Kubo 配置无效")
	}
	var addresses map[string]json.RawMessage
	if len(config["Addresses"]) > 0 && json.Unmarshal(config["Addresses"], &addresses) != nil {
		return errors.New("Kubo 地址配置无效")
	}
	if addresses == nil {
		addresses = map[string]json.RawMessage{}
	}
	addresses["API"] = json.RawMessage(`"/ip4/127.0.0.1/tcp/0"`)
	addresses["Gateway"] = json.RawMessage(`"/ip4/127.0.0.1/tcp/0"`)
	// Keep local RPC and gateway private while allowing peers to reach the node
	// over both IP families and QUIC, including during NAT hole punching.
	addresses["Swarm"] = json.RawMessage(`["/ip4/0.0.0.0/tcp/0","/ip6/::/tcp/0","/ip4/0.0.0.0/udp/0/quic-v1","/ip6/::/udp/0/quic-v1"]`)
	config["Addresses"], err = json.Marshal(addresses)
	if err != nil {
		return err
	}
	var dns map[string]json.RawMessage
	if len(config["DNS"]) > 0 && json.Unmarshal(config["DNS"], &dns) != nil {
		return errors.New("Kubo DNS 配置无效")
	}
	if dns == nil {
		dns = map[string]json.RawMessage{}
	}
	var resolvers map[string]json.RawMessage
	if len(dns["Resolvers"]) > 0 && json.Unmarshal(dns["Resolvers"], &resolvers) != nil {
		return errors.New("Kubo DNS 解析配置无效")
	}
	if resolvers == nil {
		resolvers = map[string]json.RawMessage{}
	}
	if _, exists := resolvers["eth."]; !exists {
		resolvers["eth."] = json.RawMessage(`"https://dns.eth.limo/dns-query"`)
	}
	if _, exists := resolvers["bit."]; !exists {
		resolvers["bit."] = json.RawMessage(`"https://dns.eth.limo/dns-query"`)
	}
	if _, exists := resolvers["sol."]; !exists {
		resolvers["sol."] = json.RawMessage(`"https://dns.eth.limo/dns-query"`)
	}
	if _, exists := resolvers["fc."]; !exists {
		resolvers["fc."] = json.RawMessage(`"https://dns.eth.limo/dns-query"`)
	}
	dns["Resolvers"], err = json.Marshal(resolvers)
	if err != nil {
		return err
	}
	config["DNS"], err = json.Marshal(dns)
	if err != nil {
		return err
	}
	if err = applyLowPowerDefaults(config); err != nil {
		return err
	}
	if err = applyWebUICORS(config); err != nil {
		return err
	}
	b, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(c.repo, ".config-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(b)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if e := tmp.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}

// applyLowPowerDefaults applies Kubo's desktop-friendly lowpower profile
// settings while preserving explicit user overrides and unknown fields.
func applyLowPowerDefaults(config map[string]json.RawMessage) error {
	unmarshalObject := func(raw json.RawMessage, label string) (map[string]json.RawMessage, error) {
		var obj map[string]json.RawMessage
		if len(raw) > 0 && string(bytes.TrimSpace(raw)) != "null" {
			if json.Unmarshal(raw, &obj) != nil {
				return nil, fmt.Errorf("Kubo %s配置无效", label)
			}
		}
		if obj == nil {
			obj = map[string]json.RawMessage{}
		}
		return obj, nil
	}

	autoNAT, err := unmarshalObject(config["AutoNAT"], "AutoNAT ")
	if err != nil {
		return err
	}
	if _, exists := autoNAT["ServiceMode"]; !exists {
		autoNAT["ServiceMode"] = json.RawMessage(`"disabled"`)
	}
	if config["AutoNAT"], err = json.Marshal(autoNAT); err != nil {
		return err
	}

	routing, err := unmarshalObject(config["Routing"], "Routing ")
	if err != nil {
		return err
	}
	routingType := ""
	if raw, exists := routing["Type"]; exists {
		_ = json.Unmarshal(raw, &routingType)
	}
	if routingType == "" || routingType == "auto" {
		routing["Type"] = json.RawMessage(`"autoclient"`)
	}
	if config["Routing"], err = json.Marshal(routing); err != nil {
		return err
	}

	swarm, err := unmarshalObject(config["Swarm"], "Swarm ")
	if err != nil {
		return err
	}
	connMgr, err := unmarshalObject(swarm["ConnMgr"], "Swarm.ConnMgr ")
	if err != nil {
		return err
	}
	if _, exists := connMgr["Type"]; !exists {
		connMgr["Type"] = json.RawMessage(`"basic"`)
	}
	if _, exists := connMgr["LowWater"]; !exists {
		connMgr["LowWater"] = json.RawMessage(`20`)
	}
	if _, exists := connMgr["HighWater"]; !exists {
		connMgr["HighWater"] = json.RawMessage(`40`)
	}
	if _, exists := connMgr["GracePeriod"]; !exists {
		connMgr["GracePeriod"] = json.RawMessage(`"1m0s"`)
	}
	if swarm["ConnMgr"], err = json.Marshal(connMgr); err != nil {
		return err
	}

	relayService, err := unmarshalObject(swarm["RelayService"], "Swarm.RelayService ")
	if err != nil {
		return err
	}
	if _, exists := relayService["Enabled"]; !exists {
		relayService["Enabled"] = json.RawMessage(`false`)
	}
	if swarm["RelayService"], err = json.Marshal(relayService); err != nil {
		return err
	}

	config["Swarm"], err = json.Marshal(swarm)
	return err
}

// Start owns the daemon process it creates. Offline mode is intended for tests.
func (c *Client) Start(offline bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.startLocked(offline)
}

func (c *Client) startLocked(offline bool) error {
	if c.process != nil {
		return errors.New("Kubo is already running")
	}
	if err := c.validateKeyAPI(); err != nil {
		return err
	}
	if present, err := sitecleanup.KuboRepoPresent(c.repo); err != nil || !present {
		if err != nil {
			return err
		}
		return errors.New("Kubo 仓库不存在")
	}
	if err := sitecleanup.CheckOfflineKuboRepo(c.repo); err != nil {
		return err
	}
	c.clearDaemonState()
	args := []string{"daemon"}
	if offline {
		args = append(args, "--offline")
	}
	cmd := exec.Command(c.executable, args...)
	c.configure(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	type daemonAddresses struct{ api, gateway string }
	ready := make(chan daemonAddresses, 1)
	if err := cmd.Start(); err != nil {
		return err
	}
	exited := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		announcedAPI := ""
		announcedGateway := ""
		for scanner.Scan() {
			line := scanner.Text()
			if index := strings.Index(line, "RPC API server listening on "); index >= 0 {
				announcedAPI = strings.TrimSpace(line[index+len("RPC API server listening on "):])
			}
			if index := strings.Index(line, "Gateway server listening on "); index >= 0 {
				announcedGateway = strings.TrimSpace(line[index+len("Gateway server listening on "):])
			}
			if strings.Contains(line, "Daemon is ready") {
				select {
				case ready <- daemonAddresses{api: announcedAPI, gateway: announcedGateway}:
				default:
				}
			}
		}
	}()
	go func() { exited <- cmd.Wait() }()
	select {
	case announced := <-ready:
		api := ""
		gateway := ""
		if b, err := os.ReadFile(filepath.Join(c.repo, "api")); err == nil && len(b) <= 512 && loopbackAPI.MatchString(announced.api) && strings.TrimSpace(string(b)) == announced.api {
			api = announced.api
		}
		if loopbackAPI.MatchString(announced.gateway) && !strings.HasSuffix(announced.gateway, "/tcp/0") {
			gateway = "http://127.0.0.1:" + strings.TrimPrefix(announced.gateway, "/ip4/127.0.0.1/tcp/")
		}
		if api == "" {
			_ = cmd.Process.Kill()
			<-exited
			c.clearDaemonState()
			return errors.New("Kubo API 地址未通过核验")
		}
		c.setDaemonState(cmd, exited, offline, api, gateway)
		return nil
	case err := <-exited:
		return fmt.Errorf("Kubo daemon exited before startup: %v", err)
	case <-time.After(45 * time.Second):
		_ = cmd.Process.Kill()
		<-exited
		return errors.New("Kubo daemon startup timed out")
	}
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeLocked()
}

func (c *Client) closeLocked() error {
	if c.process == nil {
		return nil
	}
	select {
	case <-c.exited:
		c.clearDaemonState()
		return nil
	default:
	}
	if err := c.validateKeyAPI(); err != nil {
		// Never send shutdown to an API address we did not start. Terminate only
		// our own child process if the repository's API file was redirected.
		if c.process != nil {
			_ = c.process.Process.Kill()
			<-c.exited
		}
		c.clearDaemonState()
		return fmt.Errorf("Kubo API 无法验证，已停止本程序的 Kubo 进程：%w", err)
	}
	if c.process == nil {
		c.clearDaemonState()
		return nil
	}
	_, err := c.runTimeout(5*time.Second, c.ownedAPIArgs("shutdown")...)
	select {
	case <-c.exited:
	case <-time.After(10 * time.Second):
		_ = c.process.Process.Kill()
		<-c.exited
		if err == nil {
			err = errors.New("Kubo did not shut down cleanly")
		}
	}
	c.clearDaemonState()
	return err
}
