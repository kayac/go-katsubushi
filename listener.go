package katsubushi

import (
	"fmt"
	"time"
)

func NewListenerFunc(kc *Config) (*App, ListenFunc, string, error) {
	var timeout time.Duration
	if kc.IdleTimeout == 0 {
		timeout = InfiniteIdleTimeout
	} else {
		timeout = time.Duration(kc.IdleTimeout) * time.Second
	}

	app, err := NewApp(Option{
		WorkerID:    kc.WorkerID,
		IdleTimeout: &timeout,
	})
	if err != nil {
		return nil, nil, "", err
	}
	if kc.Sockpath != "" {
		return app, app.ListenSock, kc.Sockpath, nil
	} else {
		return app, app.ListenTCP, fmt.Sprintf(":%d", kc.Port), nil
	}
}
