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
	"sync"
	"syscall"
	"time"

	"github.com/fujiwara/raus"
	"github.com/fukata/golang-stats-api-handler"
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

type katsubushiConfig struct {
	workerID    uint
	port        int
	idleTimeout int
	logLevel    string
	sockpath    string
}

var log *stdlog.Logger

func main() {
	var (
		showVersion bool
		redisURL    string
		minWorkerID uint
		maxWorkerID uint
	)
	pc := &profConfig{}
	kc := &katsubushiConfig{}

	flag.UintVar(&kc.workerID, "worker-id", 0, "worker id. muset be unique.")
	flag.IntVar(&kc.port, "port", 11212, "port to listen.")
	flag.StringVar(&kc.sockpath, "sock", "", "unix domain socket to listen. ignore port option when set this.")
	flag.IntVar(&kc.idleTimeout, "idle-timeout", int(katsubushi.DefaultIdleTimeout/time.Second), "connection will be closed if there are no packets over the seconds. 0 means infinite.")
	flag.StringVar(&kc.logLevel, "log-level", "info", "log level (panic, fatal, error, warn, info = Default, debug)")

	flag.BoolVar(&pc.enablePprof, "enable-pprof", false, "")
	flag.BoolVar(&pc.enableStats, "enable-stats", false, "")
	flag.IntVar(&pc.debugPort, "debug-port", 8080, "port to listen for debug")

	flag.BoolVar(&showVersion, "version", false, "show version number")
	flag.StringVar(&redisURL, "redis", "", "URL of Redis for automated worker id allocation")
	flag.UintVar(&minWorkerID, "min-worker-id", 0, "minimum automated worker id")
	flag.UintVar(&maxWorkerID, "max-worker-id", 0, "maximum automated worker id")
	flag.Parse()

	if showVersion {
		fmt.Println("katsubushi version:", katsubushi.Version)
		return
	}

	if err := katsubushi.SetLogLevel(kc.logLevel); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log = katsubushi.StdLogger()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go signalHandler(ctx, cancel, &wg)

	if kc.workerID == 0 {
		if redisURL == "" {
			fmt.Println("please set -worker-id or -redis")
			os.Exit(1)
		}
		var err error
		wg.Add(1)
		kc.workerID, err = assignWorkerID(ctx, &wg, redisURL, minWorkerID, maxWorkerID)
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
	fn, addr, err := newKatsubushiListenFunc(kc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	wg.Add(1)
	go mainListener(ctx, &wg, fn, addr)

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

func newKatsubushiListenFunc(kc *katsubushiConfig) (katsubushi.ListenFunc, string, error) {
	app, err := katsubushi.NewApp(kc.workerID)
	if err != nil {
		return nil, "", err
	}
	if err := app.SetIdleTimeout(kc.idleTimeout); err != nil {
		return nil, "", err
	}
	if kc.sockpath != "" {
		return app.ListenSock, kc.sockpath, nil
	} else {
		return app.ListenTCP, fmt.Sprintf(":%d", kc.port), nil
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
