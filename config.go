package katsubushi

type Config struct {
	WorkerID    uint
	Port        int
	IdleTimeout int
	LogLevel    string
	Sockpath    string

	HTTPPort       int
	HTTPPathPrefix string
}
