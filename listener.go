package katsubushi

import (
	"net"
	"sync"
)

func (app *App) wrapListener(l net.Listener) net.Listener {
	if _, wrapped := l.(*monitListener); wrapped {
		// already wrapped
		return l
	}
	return &monitListener{
		Listener: l,
		app:      app,
	}
}

type monitListener struct {
	net.Listener
	app *App
}

func (l *monitListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	l.app.currConnections.Add(1)
	l.app.totalConnections.Add(1)
	return &monitConn{conn, l.app, &sync.Once{}}, nil
}

type monitConn struct {
	net.Conn
	app  *App
	once *sync.Once
}

func (c *monitConn) Close() error {
	c.once.Do(func() {
		c.app.currConnections.Add(-1)
	})
	return c.Conn.Close()
}
