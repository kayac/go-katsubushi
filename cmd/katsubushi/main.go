package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/context"

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

func main() {
	var (
		showVersion bool
		redisURL    string
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
	flag.Parse()

	if showVersion {
		fmt.Println("katsubushi version:", katsubushi.Version)
		return
	}

	if kc.workerID == 0 {
		fmt.Println("please set -worker-id")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go signalHandler(ctx, cancel, &wg)

	// for profiling
	if pc.enabled() {
		log.Println("Enabling profiler")
		wg.Add(1)
		go profiler(ctx, cancel, &wg, pc)
	}

	// main listener
	app, err := newKatsubushi(kc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if kc.sockpath != "" {
			err := app.ListenSock(ctx, kc.sockpath)
			if err != nil {
				log.Println(err)
				os.Exit(1)
			}
		} else {
			err := app.ListenTCP(ctx, fmt.Sprintf(":%d", kc.port))
			if err != nil {
				log.Println(err)
				os.Exit(1)
			}
		}
	}()
	wg.Wait()
}

func newKatsubushi(kc *katsubushiConfig) (*katsubushi.App, error) {
	app, err := katsubushi.NewApp(uint32(kc.workerID))
	if err != nil {
		return nil, err
	}
	if err := app.SetIdleTimeout(kc.idleTimeout); err != nil {
		return nil, err
	}
	if err := app.SetLogLevel(kc.logLevel); err != nil {
		return nil, err
	}
	return app, nil
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
