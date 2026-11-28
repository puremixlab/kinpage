package ipfs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOnlineDHTPublicationAndDelegatedReads(t *testing.T) {
	exe := os.Getenv("PLANET_TEST_KUBO")
	if exe == "" {
		t.Skip("set PLANET_TEST_KUBO to run isolated Kubo routing checks")
	}
	t.Setenv("IPFS_HTTP_ROUTERS", "")
	t.Setenv("LIBP2P_FORCE_PNET", "")
	// Kubo's own test option lets its WAN DHT use loopback peers.
	t.Setenv("TEST_DHT_STUB", "1")
	unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("read router received %s %s", r.Method, r.URL.Path)
		}
		http.Error(w, "read router temporarily unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(unavailable.Close)

	router := startLocalRoutingNode(t, exe, nil, "", true)
	peerID, err := router.run("id", "-f=<id>")
	if err != nil {
		t.Fatal(err)
	}
	listeners, err := router.run("swarm", "addrs", "listen")
	if err != nil {
		t.Fatal(err)
	}
	var bootstrap []string
	for addr := range strings.Lines(listeners) {
		if strings.HasPrefix(addr, "/ip4/127.0.0.1/tcp/") {
			bootstrap = append(bootstrap, strings.TrimSpace(addr)+"/p2p/"+peerID)
		}
	}
	if len(bootstrap) != 1 {
		t.Fatalf("unexpected bootstrap listeners %q", listeners)
	}
	publisher := startLocalRoutingNode(t, exe, bootstrap, unavailable.URL, false)
	dhtReader := startLocalRoutingNode(t, exe, bootstrap, unavailable.URL, false)
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()

	if _, err := publisher.EnsureKeyContext(ctx, "routing-test"); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	const content = "Kinpage isolated IPFS routing test"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cid, err := publisher.AddContext(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	name, err := publisher.PublishOnlineContext(ctx, "routing-test", cid)
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.ProvideContext(ctx, cid); err != nil {
		t.Fatal(err)
	}
	publisherID, err := publisher.run("id", "-f=<id>")
	if err != nil {
		t.Fatal(err)
	}
	providers, err := dhtReader.runContext(ctx, 15*time.Second, "routing", "findprovs", cid)
	if err != nil || !strings.Contains(providers, publisherID) {
		t.Fatalf("DHT does not list the provider: %q, %v", providers, err)
	}

	delegatedReader := startLocalRoutingNode(t, exe, nil, router.snapshot().gateway, false)
	if peers, err := delegatedReader.run("swarm", "peers"); err != nil || peers != "" {
		t.Fatalf("delegated reader must start without connected peers: %q, %v", peers, err)
	}
	for _, reader := range []struct {
		name string
		node *Client
	}{{"DHT with unavailable HTTP router", dhtReader}, {"HTTP router without DHT bootstrap", delegatedReader}} {
		t.Run(reader.name, func(t *testing.T) {
			if got, err := reader.node.ResolveContext(ctx, name); err != nil || got != cid {
				t.Fatalf("resolve = %q, want %q: %v", got, cid, err)
			}
			if got, err := reader.node.ReadContext(ctx, cid, "index.html"); err != nil || string(got) != content {
				t.Fatalf("read = %q, want %q: %v", got, content, err)
			}
		})
	}
}

func startLocalRoutingNode(t *testing.T, exe string, bootstrap []string, readRouter string, serveRouting bool) *Client {
	t.Helper()
	node := New(exe, filepath.Join(t.TempDir(), "repo"))
	if err := node.Init(); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(node.repo, "config")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	cfg["Addresses"].(map[string]any)["Swarm"] = []string{"/ip4/127.0.0.1/tcp/0"}
	cfg["AutoConf"] = map[string]any{"Enabled": false}
	cfg["AutoTLS"] = map[string]any{"Enabled": false}
	cfg["Bootstrap"] = append([]string{}, bootstrap...)
	cfg["DNS"] = map[string]any{}
	cfg["Discovery"] = map[string]any{"MDNS": map[string]any{"Enabled": false}}
	cfg["Internal"] = map[string]any{"NonPublicAddrPublishing": true}
	cfg["Ipns"].(map[string]any)["DelegatedPublishers"] = []string{}
	cfg["Provide"] = map[string]any{"DHT": map[string]any{"SweepEnabled": false}}
	cfg["Swarm"].(map[string]any)["RelayClient"] = map[string]any{"Enabled": false}

	routing := cfg["Routing"].(map[string]any)
	routes := []string{}
	if serveRouting {
		routing["Type"] = "dhtserver"
		cfg["Gateway"].(map[string]any)["ExposeRoutingAPI"] = true
	} else {
		// Use the routes written by Init, with the endpoint moved to loopback.
		const origin = "https://delegated-ipfs.dev"
		for _, entry := range routing["DelegatedRouters"].([]any) {
			if path, found := strings.CutPrefix(entry.(string), origin); found {
				routes = append(routes, readRouter+path)
			}
		}
		if len(routes) != 2 {
			t.Fatalf("expected two built-in read routes, got %v", routes)
		}
	}
	routing["DelegatedRouters"] = routes
	raw, err = json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err = node.Start(false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := node.Close(); err != nil {
			t.Error(err)
		}
	})
	return node
}
