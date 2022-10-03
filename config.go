package katsubushi

import "net"

type Config struct {
	WorkerID    uint
	Port        int
	IdleTimeout int
	LogLevel    string
	Sockpath    string

	HTTPPort       int
	HTTPPathPrefix string
	HTTPListener   net.Listener
}
