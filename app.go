package idg

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	log "gopkg.in/Sirupsen/logrus.v0"
)

const (
	// Version number
	Version = "1.0.0"
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

// App : ID Generator本体。
type App struct {
	Port     int
	Listener net.Listener

	gen   *Generator
	ready bool

	// 一定時間以上通信がなければタイムアウトする
	idleTimeout time.Duration

	startedAt        time.Time
	currConnections  int64
	totalConnections int64
	cmdGet           int64
	getHits          int64
	getMisses        int64
}

// NewApp : 新しいAppオブジェクトを作って返す。
// `port`の指定を行うが、この時点ではlistenしないので
// ほかのプロセスがそのportを使っていたとしてもエラーにはならない。
func NewApp(workerID uint32, port int) (*App, error) {
	gen, err := NewGenerator(workerID)
	if err != nil {
		return nil, err
	}

	return &App{
		Port:        port,
		idleTimeout: DefaultIdleTimeout,
		gen:         gen,
		startedAt:   time.Now(),
	}, nil
}

// SetIdleTimeout : アイドル期間が指定時間以上継続した場合はコネクションを切断する。単位は秒。
// 0を指定した場合はアイドルタイムアウトが無効になる。
func (app *App) SetIdleTimeout(timeout int) error {
	if timeout < 0 {
		return fmt.Errorf("timeout must be positive")
	}

	app.idleTimeout = time.Duration(timeout) * time.Second

	return nil
}

// SetLogLevel : ログ出力のレベルを指定する。
func (app *App) SetLogLevel(str string) error {
	level, err := log.ParseLevel(str)
	if err != nil {
		return err
	}

	log.SetLevel(level)

	return nil
}

// Listen : 接続の待受を開始する。
func (app *App) Listen() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", app.Port))
	if err != nil {
		return err
	}

	log.Infof("Listening at %s", l.Addr().String())
	log.Infof("Worker ID = %d", app.gen.WorkerID)

	app.Listener = l
	app.ready = true

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Warnf("Error on accept connection: %s", err)
			continue
		}
		log.Debugf("Connected by %s", conn.RemoteAddr().String())

		go app.handleConn(conn)
	}
}

// IsReady : 接続を受け付けられる状態かどうかを返す。
func (app *App) IsReady() bool {
	return app.ready
}

func (app *App) handleConn(conn net.Conn) {
	atomic.AddInt64(&(app.totalConnections), 1)
	atomic.AddInt64(&(app.currConnections), 1)
	defer atomic.AddInt64(&(app.currConnections), -1)
	defer conn.Close()

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

// GetStats : returns MemdStats of app
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

// NextID : generatorでIDを発番する
func (app *App) NextID() (uint64, error) {
	id, err := app.gen.NextID()
	if err != nil {
		atomic.AddInt64(&(app.getMisses), 1)
	} else {
		atomic.AddInt64(&(app.getHits), 1)
	}
	return id, err
}

// BytesToCmd : 1コマンド分のバイト列を`MemdCmd`に変換して返す。
func (app *App) BytesToCmd(data []byte) (cmd MemdCmd, err error) {
	if len(data) == 0 {
		return nil, nil
	}

	fields := bytes.Fields(data)
	switch name := string(bytes.ToUpper(fields[0])); name {
	case "GET", "GETS":
		atomic.AddInt64(&(app.cmdGet), 1)
		if len(fields) < 2 {
			err = fmt.Errorf("GET command needs key as second parameter")
			return
		}

		cmd = &MemdCmdGet{
			Name: name,
			Key:  string(fields[1]),
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
	if app.idleTimeout == InfiniteIdleTimeout {
		return
	}

	conn.SetDeadline(time.Now().Add(app.idleTimeout))
}

// MemdCmd : memdの1コマンドを表すinterface
type MemdCmd interface {
	Execute(*App, net.Conn) error
}

// MemdCmdGet : memdのGetコマンドを表す
type MemdCmdGet struct {
	Name string
	Key  string
}

// Execute : generates new ID by generator
func (cmd *MemdCmdGet) Execute(app *App, conn net.Conn) error {
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

	_, err = MemdValue{
		Key:   cmd.Key,
		Flags: 0,
		Value: strconv.FormatUint(id, 10),
	}.WriteTo(conn)
	return nil
}

// MemdCmdQuit : QUIT command
type MemdCmdQuit int

// Execute : disconnect by server
func (cmd MemdCmdQuit) Execute(app *App, conn net.Conn) error {
	return io.EOF
}

// MemdCmdStats : STATS command
type MemdCmdStats int

// Execute : writes STATS response
func (cmd MemdCmdStats) Execute(app *App, conn net.Conn) error {
	_, err := app.GetStats().WriteTo(conn)
	return err
}

// MemdCmdVersion : VERSION command
type MemdCmdVersion int

// Execute : writes Version number
func (cmd MemdCmdVersion) Execute(app *App, conn net.Conn) error {
	var b bytes.Buffer
	b.Write(memdVersionHeader)
	b.WriteString(Version)
	b.Write(memdSep)
	_, err := b.WriteTo(conn)
	return err
}

// MemdValue : memdのレスポンス用の値を表す
type MemdValue struct {
	Key   string
	Flags int
	Value string
}

// MemdStats : STATSコマンドレスポンス用の値を表す
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

// WriteTo : MemdValueの内容をmemcachedプロトコル互換のフォーマットでio.Writerに書き込む
func (v MemdValue) WriteTo(w io.Writer) (int64, error) {
	var b bytes.Buffer
	b.Write(memdValHeader)
	b.WriteString(v.Key)
	b.Write(memdSpc)
	b.WriteString(strconv.Itoa(v.Flags))
	b.Write(memdSpc)
	b.WriteString(strconv.Itoa(len(v.Value)))
	b.Write(memdSep)
	b.WriteString(v.Value)
	b.Write(memdSep)
	b.Write(memdValFooter)
	return b.WriteTo(w)
}

// WriteTo : STATSコマンドのレスポンスを io.Writer に書き込む
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
