package config

import "testing"

func TestLoadFromEnv_RequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := LoadFromEnv()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadFromEnv_ParsesP2PFields(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("P2P_LISTEN_ADDRS", "/ip4/0.0.0.0/tcp/4001, /ip4/127.0.0.1/tcp/4002")
	t.Setenv("P2P_BOOTSTRAP_PEERS", " /ip4/1.2.3.4/tcp/4001/p2p/12D3KooWJ5hGQ9N9xqgWmKp5rGf9oA3Y4h8y4d1qX2b1qZ9y2a2a ")
	t.Setenv("P2P_ENABLE_MDNS", "false")
	t.Setenv("P2P_PROTOCOL_ID", "/x/1.0.0")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if len(cfg.P2PListenAddrs) != 2 {
		t.Fatalf("expected 2 listen addrs, got %d", len(cfg.P2PListenAddrs))
	}
	if cfg.P2PEnableMDNS != false {
		t.Fatalf("expected mdns false")
	}
	if cfg.P2PProtocolID != "/x/1.0.0" {
		t.Fatalf("unexpected protocol id: %s", cfg.P2PProtocolID)
	}
	if len(cfg.P2PBootstrapPeers) != 1 {
		t.Fatalf("expected 1 bootstrap peer, got %d", len(cfg.P2PBootstrapPeers))
	}
}

