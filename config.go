package katsubushi

import (
	"net"
	"time"
)

type Config struct {
	IdleTimeout time.Duration
	LogLevel    string
	LogFormat   string

	Port     int
	Sockpath string

	HTTPPort       int
	HTTPPathPrefix string
	HTTPListener   net.Listener

	GRPCPort     int
	GRPCListener net.Listener
}

// server names used to track readiness of each server.
const (
	serverMemcached = "memcached"
	serverHTTP      = "http"
	serverGRPC      = "grpc"
)

// enabledServers returns the names of the servers enabled in the config.
func (cfg *Config) enabledServers() []string {
	var names []string
	if cfg.Port != 0 || cfg.Sockpath != "" {
		names = append(names, serverMemcached)
	}
	if cfg.HTTPPort != 0 || cfg.HTTPListener != nil {
		names = append(names, serverHTTP)
	}
	if cfg.GRPCPort != 0 || cfg.GRPCListener != nil {
		names = append(names, serverGRPC)
	}
	return names
}
