package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	DataDir     string
	JWTSecret   []byte

	P2PListenAddrs   []string
	P2PBootstrapPeers []string
	P2PEnableMDNS    bool
	P2PProtocolID    string
}

func LoadFromEnv() (Config, error) {
	cfg := Config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		DataDir:     getenv("DATA_DIR", "./data"),
		P2PListenAddrs: splitAndTrim(getenv("P2P_LISTEN_ADDRS", "/ip4/0.0.0.0/tcp/4001")),
		P2PBootstrapPeers: splitAndTrim(os.Getenv("P2P_BOOTSTRAP_PEERS")),
		P2PEnableMDNS: parseBool(getenv("P2P_ENABLE_MDNS", "true")),
		P2PProtocolID: getenv("P2P_PROTOCOL_ID", "/p2p-fileshare/1.0.0"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	cfg.JWTSecret = []byte(secret)
	return cfg, nil
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func parseBool(v string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return false
	}
	return b
}

func splitAndTrim(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

