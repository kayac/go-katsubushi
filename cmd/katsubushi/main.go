package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fujiwara/raus"
	stats_api "github.com/fukata/golang-stats-api-handler"
	"github.com/kayac/go-katsubushi"
)

type profConfig struct {
	enablePprof bool
	enableStats bool
	debugPort   int
}

func (pc profConfig) enabled() bool {
	return pc.enablePprof || pc.enableStats
}

var log *stdlog.Logger

func init() {
	raus.LockExpires = 600 * time.Second
}

func main() {
	var (
		showVersion bool
		redisURL    string
		minWorkerID uint
		maxWorkerID uint
	)
	pc := &profConfig{}
	kc := &katsubushi.Config{}

	flag.UintVar(&kc.WorkerID, "worker-id", 0, "worker id. muset be unique.")
	flag.IntVar(&kc.Port, "port", 11212, "port to listen.")
	flag.StringVar(&kc.Sockpath, "sock", "", "unix domain socket to listen. ignore port option when set this.")
	flag.IntVar(&kc.IdleTimeout, "idle-timeout", int(katsubushi.DefaultIdleTimeout/time.Second), "connection will be closed if there are no packets over the seconds. 0 means infinite.")
	flag.StringVar(&kc.LogLevel, "log-level", "info", "log level (panic, fatal, error, warn, info = Default, debug)")
	flag.IntVar(&kc.HTTPPort, "http-port", 0, "port to listen http server. 0 means disable.")

	flag.BoolVar(&pc.enablePprof, "enable-pprof", false, "")
	flag.BoolVar(&pc.enableStats, "enable-stats", false, "")
	flag.IntVar(&pc.debugPort, "debug-port", 8080, "port to listen for debug")

	flag.BoolVar(&showVersion, "version", false, "show version number")
	flag.StringVar(&redisURL, "redis", "", "URL of Redis for automated worker id allocation")
	flag.UintVar(&minWorkerID, "min-worker-id", 0, "minimum automated worker id")
	flag.UintVar(&maxWorkerID, "max-worker-id", 0, "maximum automated worker id")
	flag.VisitAll(envToFlag)
	flag.Parse()

	if showVersion {
		fmt.Println("katsubushi version:", katsubushi.Version)
		return
	}

	if err := katsubushi.SetLogLevel(kc.LogLevel); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log = katsubushi.StdLogger()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go signalHandler(ctx, cancel, &wg)

	if kc.WorkerID == 0 {
		if redisURL == "" {
			fmt.Println("please set -worker-id or -redis")
			os.Exit(1)
		}
		var err error
		wg.Add(1)
		kc.WorkerID, err = assignWorkerID(ctx, &wg, redisURL, minWorkerID, maxWorkerID)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}

	// for profiling
	if pc.enabled() {
		log.Println("Enabling profiler")
		wg.Add(1)
		go profiler(ctx, cancel, &wg, pc)
	}

	// main listener
	app, fn, addr, err := katsubushi.NewListenerFunc(kc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	wg.Add(1)
	go mainListener(ctx, &wg, fn, addr)

	// http server
	if kc.HTTPPort != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := app.RunHTTPServer(ctx, kc); err != nil {
				fmt.Println(err)
				cancel()
			}
		}()
	}

	wg.Wait()
	log.Println("Shutdown completed")
}

func mainListener(ctx context.Context, wg *sync.WaitGroup, fn katsubushi.ListenFunc, addr string) {
	defer wg.Done()
	if err := fn(ctx, addr); err != nil {
		log.Println("Listen failed", err)
		os.Exit(1)
	}
}

func profiler(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, pc *profConfig) {
	defer wg.Done()

	mux := http.NewServeMux()
	if pc.enablePprof {
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		log.Println("EnablePprof on /debug/pprof")
	}
	if pc.enableStats {
		mux.HandleFunc("/debug/stats", stats_api.Handler)
		log.Println("EnableStats on /debug/stats")
	}
	addr := fmt.Sprintf("localhost:%d", pc.debugPort)
	log.Println("Listening debugger on", addr)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	if err := http.Serve(ln, mux); err != nil {
		log.Println(err)
		return
	}
}

func signalHandler(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) {
	defer wg.Done()
	trapSignals := []os.Signal{
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, trapSignals...)
	select {
	case sig := <-sigCh:
		log.Printf("Got signal %s", sig)
		cancel()
	case <-ctx.Done():
	}
}

func assignWorkerID(ctx context.Context, wg *sync.WaitGroup, redisURL string, min, max uint) (uint, error) {
	defer wg.Done()
	raus.SetLogger(log)
	defaultMax := uint((1 << katsubushi.WorkerIDBits) - 1)
	if min == 0 {
		min = 1
	}
	if max == 0 {
		max = defaultMax
	}
	if min > max {
		return 0, errors.New("max-worker-id must be larger than min-worker-id")
	}
	if max > defaultMax {
		return 0, fmt.Errorf("max-worker-id must be smaller than %d", defaultMax)
	}
	log.Printf("Waiting for worker-id automated assignment (between %d and %d) with %s", min, max, redisURL)
	r, err := raus.New(redisURL, min, max)
	if err != nil {
		log.Println("Failed to assign worker-id", err)
		return 0, err
	}
	id, ch, err := r.Get(ctx)
	if err != nil {
		return 0, err
	}
	log.Printf("Assigned worker-id: %d", id)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err, more := <-ch
		if err != nil {
			panic(err)
		}
		if !more {
			// shutdown
		}
	}()
	return id, nil
}

func envToFlag(f *flag.Flag) {
	names := []string{
		strings.ToUpper(strings.Replace(f.Name, "-", "_", -1)),
		strings.ToLower(strings.Replace(f.Name, "-", "_", -1)),
	}
	for _, name := range names {
		if s := os.Getenv(name); s != "" {
			f.Value.Set(s)
			break
		}
	}
}
