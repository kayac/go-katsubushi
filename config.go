package katsubushi

import (
	"net"
	"time"
)

type Config struct {
	IdleTimeout time.Duration
	LogLevel    string

	Port     int
	Sockpath string

	HTTPPort       int
	HTTPPathPrefix string
	HTTPListener   net.Listener

	GRPCPort     int
	GRPCListener net.Listener
}
