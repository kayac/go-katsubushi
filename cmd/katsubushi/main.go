package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
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
	"github.com/kayac/go-katsubushi/v2"
)

type profConfig struct {
	enablePprof bool
	enableStats bool
	debugPort   int
}

func (pc profConfig) enabled() bool {
	return pc.enablePprof || pc.enableStats
}

func init() {
	raus.LockExpires = 600 * time.Second
}

func main() {
	var (
		showVersion bool
		redisURL    string
		minWorkerID uint
		maxWorkerID uint
		workerID    uint
		jsSafeID    bool
	)
	pc := &profConfig{}
	kc := &katsubushi.Config{}

	flag.UintVar(&workerID, "worker-id", 0, "worker id. must be unique.")
	flag.IntVar(&kc.Port, "port", 11212, "port to listen. 0 means disable.")
	flag.StringVar(&kc.Sockpath, "sock", "", "unix domain socket to listen. ignore port option when set this.")
	flag.DurationVar(&kc.IdleTimeout, "idle-timeout", katsubushi.DefaultIdleTimeout, "connection will be closed if there are no packets over the seconds. 0 means infinite.")
	flag.StringVar(&kc.LogLevel, "log-level", "info", "log level (panic, fatal, error, warn, info = Default, debug)")
	flag.StringVar(&kc.LogFormat, "log-format", "text", "log format (text = Default, json)")
	flag.IntVar(&kc.HTTPPort, "http-port", 0, "port to listen http server. 0 means disable.")
	flag.IntVar(&kc.GRPCPort, "grpc-port", 0, "port to listen grpc server. 0 means disable.")

	flag.BoolVar(&pc.enablePprof, "enable-pprof", false, "")
	flag.BoolVar(&pc.enableStats, "enable-stats", false, "")
	flag.IntVar(&pc.debugPort, "debug-port", 8080, "port to listen for debug")

	flag.BoolVar(&showVersion, "version", false, "show version number")
	flag.StringVar(&redisURL, "redis", "", "URL of Redis for automated worker id allocation")
	flag.UintVar(&minWorkerID, "min-worker-id", 0, "minimum automated worker id")
	flag.UintVar(&maxWorkerID, "max-worker-id", 0, "maximum automated worker id")
	flag.BoolVar(&jsSafeID, "js-safe-id", false, "generate IDs that fit within 2^53-1 (Number.MAX_SAFE_INTEGER in JavaScript)")
	flag.VisitAll(envToFlag)
	flag.Parse()

	if showVersion {
		fmt.Println("katsubushi version:", katsubushi.Version)
		return
	}

	if err := katsubushi.SetLogFormat(kc.LogFormat); err != nil {
		slog.Error("failed to set log format", "format", kc.LogFormat, "error", err)
		os.Exit(1)
	}
	if err := katsubushi.SetLogLevel(kc.LogLevel); err != nil {
		slog.Error("failed to set log level", "level", kc.LogLevel, "error", err)
		os.Exit(1)
	}

	if kc.Port == 0 && kc.Sockpath == "" && kc.HTTPPort == 0 && kc.GRPCPort == 0 {
		fmt.Println("no server is enabled. please set -port, -sock, -http-port or -grpc-port")
		os.Exit(1)
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go signalHandler(ctx, cancel, &wg)

	workerIDBits := uint(katsubushi.WorkerIDBits)
	if jsSafeID {
		workerIDBits = katsubushi.JSSafeWorkerIDBits
	}
	if workerID == 0 {
		if redisURL == "" {
			fmt.Println("please set -worker-id or -redis")
			os.Exit(1)
		}
		var err error
		wg.Add(1)
		workerID, err = assignWorkerID(ctx, &wg, redisURL, minWorkerID, maxWorkerID, workerIDBits)
		if err != nil {
			slog.Error("failed to assign worker-id", "error", err)
			os.Exit(1)
		}
	}

	// for profiling
	if pc.enabled() {
		slog.Info("Enabling profiler")
		wg.Add(1)
		go profiler(ctx, cancel, &wg, pc)
	}

	var app *katsubushi.App
	var err error
	if jsSafeID {
		app, err = katsubushi.NewJSSafe(workerID)
	} else {
		app, err = katsubushi.New(workerID)
	}
	if err != nil {
		slog.Error("failed to create app", "error", err)
		os.Exit(1)
	}

	var errs []error
	var errsMu sync.Mutex
	recordErr := func(err error) {
		errsMu.Lock()
		defer errsMu.Unlock()
		errs = append(errs, err)
	}

	// memcached compatible server
	if kc.Port != 0 || kc.Sockpath != "" {
		wg.Go(func() {
			if err := app.RunServer(ctx, kc); err != nil {
				recordErr(err)
				cancel()
			}
		})
	} else {
		// Serve() logs them when the memcached compatible server is enabled.
		idFormat := "default"
		if jsSafeID {
			idFormat = "js-safe"
		}
		slog.Info("Memcached compatible server is disabled",
			"worker_id", uint64(workerID),
			"id_format", idFormat,
		)
	}

	// http server
	if kc.HTTPPort != 0 {
		wg.Go(func() {
			if err := app.RunHTTPServer(ctx, kc); err != nil {
				recordErr(err)
				cancel()
			}
		})
	}

	if kc.GRPCPort != 0 {
		wg.Go(func() {
			if err := app.RunGRPCServer(ctx, kc); err != nil {
				recordErr(err)
				cancel()
			}
		})
	}

	wg.Wait()
	code := 0
	if len(errs) > 0 {
		for _, err := range errs {
			if errors.Is(err, context.Canceled) || errors.Is(err, http.ErrServerClosed) {
				continue
			}
			slog.Error("server error", "error", err)
			code = 1
		}
	}
	slog.Info("Shutdown completed")
	os.Exit(code)
}

func profiler(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, pc *profConfig) {
	defer wg.Done()

	mux := http.NewServeMux()
	if pc.enablePprof {
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		slog.Info("EnablePprof on /debug/pprof")
	}
	if pc.enableStats {
		mux.HandleFunc("/debug/stats", stats_api.Handler)
		slog.Info("EnableStats on /debug/stats")
	}
	addr := fmt.Sprintf("localhost:%d", pc.debugPort)
	slog.Info("Listening debugger on", "addr", addr)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("failed to listen", "addr", addr, "error", err)
		return
	}

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	if err := http.Serve(ln, mux); err != nil {
		slog.Error("failed to serve", "error", err)
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
		slog.Info("Got signal", "signal", sig)
		cancel()
	case <-ctx.Done():
	}
}

func assignWorkerID(ctx context.Context, wg *sync.WaitGroup, redisURL string, min, max, workerIDBits uint) (uint, error) {
	defer wg.Done()
	defaultMax := uint((1 << workerIDBits) - 1)
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
	slog.Info("Waiting for worker-id automated assignment", "min", min, "max", max, "redisURL", redisURL)
	raus.SubscribeTimeout = 0 // skip Discovery
	r, err := raus.New(redisURL, min, max)
	if err != nil {
		slog.Error("failed to assign worker-id", "error", err)
		return 0, err
	}
	r.SetSlogLogger(katsubushi.SlogLogger())
	id, ch, err := r.Get(ctx)
	if err != nil {
		return 0, err
	}
	slog.Info("Assigned worker-id", "id", id)

	wg.Go(func() {
		err, more := <-ch
		if err != nil {
			panic(err)
		}
		if !more {
			// shutdown
		}
	})
	return id, nil
}

func envToFlag(f *flag.Flag) {
	if err := applyEnvToFlag(f); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func applyEnvToFlag(f *flag.Flag) error {
	name := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
	names := []string{
		"KATSUBUSHI_" + name,
		name,
		strings.ToLower(name),
	}
	for _, name := range names {
		if s := os.Getenv(name); s != "" {
			if err := f.Value.Set(s); err != nil {
				return fmt.Errorf("invalid value %q in environment variable %s for flag -%s: %w", s, name, f.Name, err)
			}
			break
		}
	}
	return nil
}
