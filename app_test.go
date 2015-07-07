package katsubushi

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

func newTestApp(t *testing.T) *App {
	app, _ := NewApp(getNextWorkerID(), 0)

	if testing.Verbose() {
		app.SetLogLevel("debug")
	} else {
		app.SetLogLevel("panic")
	}

	return app
}

func newTestAppAndListen(t *testing.T) *App {
	app := newTestApp(t)

	go app.Listen()

	for {
		if app.IsReady() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return app
}

func newTestAppAndListenSock(t *testing.T) *App {
	app := newTestApp(t)

	tmpDir, err := ioutil.TempDir(os.TempDir(), "katsubushi")
	if err != nil {
		t.Fatal("Can't create temp directory:", err)
	}

	go app.ListenSock(filepath.Join(tmpDir, "katsubushi.sock"))

	for {
		if app.IsReady() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return app
}

func TestApp(t *testing.T) {
	app := newTestAppAndListen(t)
	mc := memcache.New(app.Listener.Addr().String())

	item, err := mc.Get("hoge")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("key = %s", item.Key)
	t.Logf("flags = %d", item.Flags)
	t.Logf("id = %s", item.Value)

	if k := item.Key; k != "hoge" {
		t.Errorf("Unexpected key: %s", k)
	}

	if f := item.Flags; f != 0 {
		t.Errorf("Unexpected flags: %d", f)
	}

	if _, err := strconv.ParseInt(string(item.Value), 10, 64); err != nil {
		t.Errorf("Invalid id: %s", err)
	}
}

func TestAppSock(t *testing.T) {
	app := newTestAppAndListenSock(t)
	mc := memcache.New(app.Listener.Addr().String())

	item, err := mc.Get("hoge")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("key = %s", item.Key)
	t.Logf("flags = %d", item.Flags)
	t.Logf("id = %s", item.Value)

	if k := item.Key; k != "hoge" {
		t.Errorf("Unexpected key: %s", k)
	}

	if f := item.Flags; f != 0 {
		t.Errorf("Unexpected flags: %d", f)
	}

	if _, err := strconv.ParseInt(string(item.Value), 10, 64); err != nil {
		t.Errorf("Invalid id: %s", err)
	}
}

func TestAppError(t *testing.T) {
	app := newTestAppAndListen(t)
	mc := memcache.New(app.Listener.Addr().String())

	err := mc.Set(&memcache.Item{
		Key:   "hoge",
		Value: []byte("fuga"),
	})

	if err == nil {
		t.Fatal("Must be error")
	}

	if r := regexp.MustCompile(`ERROR`); !r.MatchString(err.Error()) {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestAppIdleTimeout(t *testing.T) {
	app := newTestAppAndListen(t)
	app.SetIdleTimeout(1)

	mc := memcache.New(app.Listener.Addr().String())

	t.Log("Before timeout")
	{
		item, err := mc.Get("hoge")
		if err != nil {
			t.Fatal(err)
		}

		if _, err := strconv.ParseInt(string(item.Value), 10, 64); err != nil {
			t.Errorf("Invalid id: %s", err)
		}
	}

	time.Sleep(2 * time.Second)

	t.Log("After timeout")
	{
		_, err := mc.Get("hoge")
		if err == nil {
			t.Fatal("Connection must be disconnected")
		}
	}
}

func BenchmarkApp(b *testing.B) {
	app, _ := NewApp(getNextWorkerID(), 0)
	go app.Listen()

	for {
		if app.IsReady() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	errorPattern := regexp.MustCompile(`ERROR`)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		client, err := newTestClient(app.Listener.Addr().String())
		if err != nil {
			b.Fatalf("Failed to connect to app: %s", err)
		}
		for pb.Next() {
			resp, err := client.Command("GET hoge")
			if err != nil {
				b.Fatalf("Error on write: %s", err)
			}
			if errorPattern.Match(resp) {
				b.Fatalf("Got ERROR")
			}
		}
	})
}

func TestStats(t *testing.T) {
	s := MemdStats{
		Pid:              12345,
		Uptime:           10,
		Time:             1432714475,
		Version:          "0.0.1",
		CurrConnections:  10,
		TotalConnections: 123,
		CmdGet:           399,
		GetHits:          396,
		GetMisses:        3,
	}
	var b bytes.Buffer
	buf := bufio.NewWriter(&b)
	s.WriteTo(buf)
	buf.Flush()
	expected := `STAT pid 12345
STAT uptime 10
STAT time 1432714475
STAT version 0.0.1
STAT curr_connections 10
STAT total_connections 123
STAT cmd_get 399
STAT get_hits 396
STAT get_misses 3
END
`
	expected = strings.Replace(expected, "\n", "\r\n", -1)
	if res := b.String(); res != expected {
		t.Error("unexpected STATS output", res, expected)
	}
}

type testClient struct {
	conn net.Conn
}

func (c *testClient) Command(str string) ([]byte, error) {
	resp := make([]byte, 1024)
	_, err := c.conn.Write([]byte(str + "\r\n"))
	if err != nil {
		return nil, err
	}
	n, err := c.conn.Read(resp)
	if err != nil {
		return nil, err
	}
	return resp[0:n], nil
}

func newTestClient(addr string) (*testClient, error) {
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return nil, err
	}
	return &testClient{conn}, nil
}

func TestAppVersion(t *testing.T) {
	app := newTestAppAndListen(t)
	client, err := newTestClient(app.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_resp, err := client.Command("VERSION")
	if string(_resp) != "VERSION "+Version+"\r\n" {
		t.Error("invalid version", string(_resp))
	}
}

func TestAppQuit(t *testing.T) {
	app := newTestAppAndListen(t)
	client, err := newTestClient(app.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Command("QUIT")
	// quitしたら切断されるのでreadしたらEOFがくるはず
	if err != io.EOF {
		t.Error("QUIT failed", err)
	}
}

func TestAppStats(t *testing.T) {
	app := newTestAppAndListen(t)
	client, err := newTestClient(app.Listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to app: %s", err)
	}
	{
		_resp, err := client.Command("STATS")
		if err != nil {
			t.Fatal(err)
		}
		stats, err := parseStats(string(_resp))
		if err != nil {
			t.Fatal(err)
		}
		if stats["total_connections"] != 1 ||
			stats["curr_connections"] != 1 ||
			stats["cmd_get"] != 0 ||
			stats["get_hits"] != 0 ||
			stats["get_misses"] != 0 {
			t.Error("invalid stats", stats)
		}
	}

	_, _ = client.Command("GET id")
	{
		// get したあとは get_hits, cmd_get が増えてる
		_resp, err := client.Command("STATS")
		if err != nil {
			t.Fatal(err)
		}
		stats, err := parseStats(string(_resp))
		if err != nil {
			t.Fatal(err)
		}
		if stats["total_connections"] != 1 ||
			stats["curr_connections"] != 1 ||
			stats["cmd_get"] != 1 ||
			stats["get_hits"] != 1 ||
			stats["get_misses"] != 0 {
			t.Error("invalid stats", stats)
		}
	}

	time.Sleep(2 * time.Second)
	{
		// uptimeが増えてるはず
		_resp, err := client.Command("STATS")
		if err != nil {
			t.Fatal(err)
		}
		stats, err := parseStats(string(_resp))
		if err != nil {
			t.Fatal(err)
		}
		if stats["uptime"] < 2 {
			t.Error("invalid stats", stats)
		}
	}
}

func parseStats(str string) (map[string]int64, error) {
	lines := strings.Split(str, "\r\n")
	stats := make(map[string]int64, len(lines))
	for _, line := range lines {
		col := strings.Split(line, " ")
		if len(col) < 3 {
			continue
		}
		if col[0] == "STAT" {
			stats[col[1]], _ = strconv.ParseInt(col[2], 10, 64)
		}
	}
	if lines[len(lines)-2] != "END" {
		return nil, fmt.Errorf("end of result != END %#v", lines)
	}
	if len(lines)-2 != len(stats) {
		return nil, fmt.Errorf("includes invalid line %s", stats)
	}
	return stats, nil
}
