package katsubushi

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Verbose() {
		SetLogLevel("debug")
	} else {
		SetLogLevel("panic")
	}
	os.Exit(m.Run())
}

func newTestApp(t testing.TB) *App {
	app, err := NewApp(getNextWorkerID())
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func newTestAppAndListenTCP(ctx context.Context, t testing.TB) *App {
	app := newTestApp(t)

	go app.ListenTCP(ctx, ":0")
	<-app.Ready()

	return app
}

func newTestAppAndListenSock(ctx context.Context, t testing.TB) (*App, string) {
	app := newTestApp(t)

	tmpDir, _ := ioutil.TempDir("", "go-katsubushi-")

	go app.ListenSock(ctx, filepath.Join(tmpDir, "katsubushi.sock"))
	<-app.Ready()

	return app, tmpDir
}

func TestApp(t *testing.T) {
	ctx := context.Background()
	app := newTestAppAndListenTCP(ctx, t)
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

func TestAppMulti(t *testing.T) {
	ctx := context.Background()
	app := newTestAppAndListenTCP(ctx, t)
	mc := memcache.New(app.Listener.Addr().String())
	keys := []string{"foo", "bar", "baz"}
	items, err := mc.GetMulti(keys)
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range keys {
		item := items[key]
		if k := item.Key; k != key {
			t.Errorf("Unexpected key: %s", k)
		}

		if f := item.Flags; f != 0 {
			t.Errorf("Unexpected flags: %d", f)
		}

		if _, err := strconv.ParseInt(string(item.Value), 10, 64); err != nil {
			t.Errorf("Invalid id: %s", err)
		}
	}
}

func TestAppSock(t *testing.T) {
	ctx := context.Background()
	app, tmpDir := newTestAppAndListenSock(ctx, t)
	mc := memcache.New(app.Listener.Addr().String())
	defer os.RemoveAll(tmpDir)

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
	ctx := context.Background()
	app := newTestAppAndListenTCP(ctx, t)
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
	ctx := context.Background()
	app := newTestAppAndListenTCP(ctx, t)
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
	app, _ := NewApp(getNextWorkerID())
	go app.ListenTCP(context.Background(), ":0")
	<-app.Ready()

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

func BenchmarkAppSock(b *testing.B) {
	app, _ := NewApp(getNextWorkerID())
	tmpDir, _ := ioutil.TempDir("", "go-katsubushi-")
	defer os.RemoveAll(tmpDir)

	go app.ListenSock(
		context.Background(),
		filepath.Join(tmpDir, "katsubushi.sock"),
	)
	<-app.Ready()

	errorPattern := regexp.MustCompile(`ERROR`)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		client, err := newTestClientSock(filepath.Join(tmpDir, "katsubushi.sock"))
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

func newTestClientSock(path string) (*testClient, error) {
	conn, err := net.DialTimeout("unix", path, 1*time.Second)
	if err != nil {
		return nil, err
	}
	return &testClient{conn}, nil
}

func TestAppVersion(t *testing.T) {
	ctx := context.Background()
	app := newTestAppAndListenTCP(ctx, t)
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
	ctx := context.Background()
	app := newTestAppAndListenTCP(ctx, t)
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
	ctx := context.Background()
	app := newTestAppAndListenTCP(ctx, t)
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

func TestAppEmptyCommand(t *testing.T) {
	ctx := context.Background()
	app := newTestAppAndListenTCP(ctx, t)
	client, err := newTestClient(app.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_resp, err := client.Command("") // empty string
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(_resp), "ERROR") {
		t.Errorf("expected ERROR got %s", _resp)
	}
}

func TestAppStatsReaceCondition(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	app := newTestAppAndListenTCP(ctx, t)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		client, err := newTestClient(app.Listener.Addr().String())
		if err != nil {
			t.Fatalf("Failed to connect to app: %s", err)
		}
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			client.Command("GET id")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		client, err := newTestClient(app.Listener.Addr().String())
		if err != nil {
			t.Fatalf("Failed to connect to app: %s", err)
		}
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			client.Command("STATS")
		}
	}()

	wg.Wait()
}

func TestAppCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	app := newTestAppAndListenTCP(ctx, t)
	{
		client, err := newTestClient(app.Listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Command("VERSION")
		if err != nil {
			t.Fatal(err)
		}
		cancelAndWait(cancel)
		// disconnect by peer after canceled
		res, err := client.Command("VERSION")
		if err == nil && len(res) > 0 { // response returned
			t.Fatal(err, res)
		}
		t.Log(res, err)
	}
	{
		// failed to conenct after canceled
		_, err := newTestClient(app.Listener.Addr().String())
		if err == nil {
			t.Fatal(err)
		}
	}
}
