package katsubushi

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

var (
	// Version number
	Version      = "development"
	zaplogger, _ = zap.NewDevelopment()
	log          = zaplogger.Sugar()
)

var (
	respError         = []byte("ERROR\r\n")
	memdSep           = []byte("\r\n")
	memdSepLen        = len(memdSep)
	memdSpc           = []byte(" ")
	memdValHeader     = []byte("VALUE ")
	memdValFooter     = []byte("END\r\n")
	memdStatHeader    = []byte("STAT ")
	memdVersionHeader = []byte("VERSION ")

	DefaultIdleTimeout  = 600 * time.Second
	InfiniteIdleTimeout = time.Duration(0)
)

// App is main struct of the Application.
type App struct {
	Listener net.Listener

	gen     *Generator
	readyCh chan interface{}
	mu      sync.Mutex

	// App will disconnect connection if there are no commands until idleTimeout.
	idleTimeout time.Duration

	startedAt        time.Time
	currConnections  int64
	totalConnections int64
	cmdGet           int64
	getHits          int64
	getMisses        int64
}

// NewApp create and returns new App instance.
func NewApp(workerID uint) (*App, error) {
	gen, err := NewGenerator(workerID)
	if err != nil {
		return nil, err
	}

	return &App{
		idleTimeout: DefaultIdleTimeout,
		gen:         gen,
		startedAt:   time.Now(),
		readyCh:     make(chan interface{}),
	}, nil
}

// SetIdleTimeout sets duration before disconnect caused by idle networking.
// To disable idle timeout, set 0.
func (app *App) SetIdleTimeout(timeout int) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	if timeout < 0 {
		return fmt.Errorf("timeout must be positive")
	}

	app.idleTimeout = time.Duration(timeout) * time.Second

	return nil
}

// SetLogLevel sets log level.
// Log level must be one of debug, info, warning, error, fatal and panic.
func SetLogLevel(str string) error {
	conf := zap.Config{
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	switch str {
	case "debug":
		conf.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		conf.Development = true
	case "info":
		conf.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warning":
		conf.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		conf.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "fatal":
		conf.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
	case "panic":
		conf.Level = zap.NewAtomicLevelAt(zap.PanicLevel)
	default:
		return fmt.Errorf("invalid log level %s", str)
	}
	zaplogger, _ = conf.Build()
	log = zaplogger.Sugar()
	return nil
}

func StdLogger() *stdlog.Logger {
	return zap.NewStdLog(zaplogger)
}

type ListenFunc func(context.Context, string) error

// ListenSock starts listen Unix Domain Socket on sockpath.
func (app *App) ListenSock(ctx context.Context, sockpath string) error {
	l, err := net.Listen("unix", sockpath)
	if err != nil {
		return err
	}

	return app.Listen(ctx, l)
}

// ListenTCP starts listen on host:port.
func (app *App) ListenTCP(ctx context.Context, addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return app.Listen(ctx, l)
}

// Listen starts listen.
func (app *App) Listen(ctx context.Context, l net.Listener) error {
	log.Infof("Listening at %s", l.Addr().String())
	log.Infof("Worker ID = %d", app.gen.WorkerID)

	app.Listener = l
	close(app.readyCh)

	go func() {
		<-ctx.Done()
		if err := l.Close(); err != nil {
			log.Warn(err)
		}
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				log.Info("Shutting down listener")
				return nil
			default:
				log.Warnf("Error on accept connection: %s", err)
				return err
			}
		}
		log.Debugf("Connected by %s", conn.RemoteAddr().String())

		go app.handleConn(ctx, conn)
	}
	return nil
}

// Ready returns a channel which become readable when the app can accept connections.
func (app *App) Ready() chan interface{} {
	return app.readyCh
}

func (app *App) handleConn(ctx context.Context, conn net.Conn) {
	atomic.AddInt64(&(app.totalConnections), 1)
	atomic.AddInt64(&(app.currConnections), 1)
	defer atomic.AddInt64(&(app.currConnections), -1)
	defer conn.Close()

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	app.extendDeadline(conn)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		app.extendDeadline(conn)
		cmd, err := app.BytesToCmd(scanner.Bytes())
		if err != nil {
			if err := app.writeError(conn); err != nil {
				log.Warn("error on write error: %s", err)
				return
			}
			continue
		}
		err = cmd.Execute(app, conn)
		if err == io.EOF {
			return
		} else if err != nil {
			log.Warn("error on write to conn: %s", err)
			return
		}
	}
}

// GetStats returns MemdStats of app
func (app *App) GetStats() MemdStats {
	now := time.Now()
	return MemdStats{
		Pid:              os.Getpid(),
		Uptime:           int64(now.Sub(app.startedAt).Seconds()),
		Time:             time.Now().Unix(),
		Version:          Version,
		CurrConnections:  app.currConnections,
		TotalConnections: app.totalConnections,
		CmdGet:           app.cmdGet,
		GetHits:          app.getHits,
		GetMisses:        app.getMisses,
	}
}

func (app *App) writeError(conn net.Conn) (err error) {
	_, err = conn.Write(respError)
	if err != nil {
		log.Warn(err)
	}

	return
}

// NextID generates new ID.
func (app *App) NextID() (uint64, error) {
	id, err := app.gen.NextID()
	if err != nil {
		atomic.AddInt64(&(app.getMisses), 1)
	} else {
		atomic.AddInt64(&(app.getHits), 1)
	}
	return id, err
}

// BytesToCmd converts byte array to a MemdCmd and returns it.
func (app *App) BytesToCmd(data []byte) (cmd MemdCmd, err error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("No command")
	}

	fields := strings.Fields(string(data))
	switch name := strings.ToUpper(fields[0]); name {
	case "GET", "GETS":
		atomic.AddInt64(&(app.cmdGet), 1)
		if len(fields) < 2 {
			err = fmt.Errorf("GET command needs key as second parameter")
			return
		}
		cmd = &MemdCmdGet{
			Name: name,
			Keys: fields[1:],
		}
	case "QUIT":
		cmd = MemdCmdQuit(0)
	case "STATS":
		cmd = MemdCmdStats(0)
	case "VERSION":
		cmd = MemdCmdVersion(0)
	default:
		err = fmt.Errorf("Unknown command: %s", name)
	}
	return
}

func (app *App) extendDeadline(conn net.Conn) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.idleTimeout == InfiniteIdleTimeout {
		return
	}

	conn.SetDeadline(time.Now().Add(app.idleTimeout))
}

// MemdCmd defines a command.
type MemdCmd interface {
	Execute(*App, net.Conn) error
}

// MemdCmdGet defines Get command.
type MemdCmdGet struct {
	Name string
	Keys []string
}

// Execute generates new ID.
func (cmd *MemdCmdGet) Execute(app *App, conn net.Conn) error {
	values := make([]string, len(cmd.Keys))
	for i, _ := range cmd.Keys {
		id, err := app.NextID()
		if err != nil {
			log.Warn(err)
			if err = app.writeError(conn); err != nil {
				log.Warn("error on write error: %s", err)
				return err
			}
			return nil
		}
		log.Debugf("Generated ID: %d", id)
		values[i] = strconv.FormatUint(id, 10)
	}
	_, err := MemdValue{
		Keys:   cmd.Keys,
		Flags:  0,
		Values: values,
	}.WriteTo(conn)
	return err
}

// MemdCmdQuit defines QUIT command.
type MemdCmdQuit int

// Execute disconnect by server.
func (cmd MemdCmdQuit) Execute(app *App, conn net.Conn) error {
	return io.EOF
}

// MemdCmdStats defines STATS command.
type MemdCmdStats int

// Execute writes STATS response.
func (cmd MemdCmdStats) Execute(app *App, conn net.Conn) error {
	_, err := app.GetStats().WriteTo(conn)
	return err
}

// MemdCmdVersion defines VERSION command.
type MemdCmdVersion int

// Execute writes Version number.
func (cmd MemdCmdVersion) Execute(app *App, conn net.Conn) error {
	var b bytes.Buffer
	b.Write(memdVersionHeader)
	b.WriteString(Version)
	b.Write(memdSep)
	_, err := b.WriteTo(conn)
	return err
}

// MemdValue defines return value for client.
type MemdValue struct {
	Keys   []string
	Flags  int
	Values []string
}

// MemdStats defines result of STATS command.
type MemdStats struct {
	Pid              int    `memd:"pid"`
	Uptime           int64  `memd:"uptime"`
	Time             int64  `memd:"time"`
	Version          string `memd:"version"`
	CurrConnections  int64  `memd:"curr_connections"`
	TotalConnections int64  `memd:"total_connections"`
	CmdGet           int64  `memd:"cmd_get"`
	GetHits          int64  `memd:"get_hits"`
	GetMisses        int64  `memd:"get_misses"`
}

// WriteTo writes content of MemdValue to io.Writer.
// Its format is compatible to memcached protocol.
func (v MemdValue) WriteTo(w io.Writer) (int64, error) {
	var b bytes.Buffer
	for i, key := range v.Keys {
		b.Write(memdValHeader)
		b.WriteString(key)
		b.Write(memdSpc)
		b.WriteString(strconv.Itoa(v.Flags))
		b.Write(memdSpc)
		b.WriteString(strconv.Itoa(len(v.Values[i])))
		b.Write(memdSep)
		b.WriteString(v.Values[i])
		b.Write(memdSep)
	}
	b.Write(memdValFooter)
	return b.WriteTo(w)
}

// WriteTo writes result of STATS command to io.Writer.
func (s MemdStats) WriteTo(w io.Writer) (int64, error) {
	var b bytes.Buffer
	statsValue := reflect.ValueOf(s)
	statsType := reflect.TypeOf(s)
	for i := 0; i < statsType.NumField(); i++ {
		b.Write(memdStatHeader)
		field := statsType.Field(i)
		if tag := field.Tag.Get("memd"); tag != "" {
			b.WriteString(tag)
		} else {
			b.WriteString(strings.ToUpper(field.Name))
		}
		b.Write(memdSpc)
		v := statsValue.FieldByIndex(field.Index).Interface()
		switch _v := v.(type) {
		case int:
			b.WriteString(strconv.Itoa(_v))
		case int64:
			b.WriteString(strconv.FormatInt(int64(_v), 10))
		case string:
			b.WriteString(string(_v))
		}
		b.Write(memdSep)
	}
	b.Write(memdValFooter)
	return b.WriteTo(w)
}
