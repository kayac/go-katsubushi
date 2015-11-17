package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/fukata/golang-stats-api-handler"
	"github.com/kayac/go-katsubushi"
)

func main() {
	var (
		workerID    uint
		port        int
		idleTimeout int
		logLevel    string
		enablePprof bool
		enableStats bool
		debugPort   int
		sockpath    string
		showVersion bool
	)

	flag.UintVar(&workerID, "worker-id", 0, "worker id. muset be unique.")
	flag.IntVar(&port, "port", 11212, "port to listen.")
	flag.StringVar(&sockpath, "sock", "", "unix domain socket to listen. ignore port option when set this.")
	flag.IntVar(&idleTimeout, "idle-timeout", int(katsubushi.DefaultIdleTimeout/time.Second), "connection will be closed if there are no packets over the seconds. 0 means infinite.")
	flag.StringVar(&logLevel, "log-level", "info", "log level (panic, fatal, error, warn, info = Default, debug)")
	flag.BoolVar(&enablePprof, "enable-pprof", false, "")
	flag.BoolVar(&enableStats, "enable-stats", false, "")
	flag.BoolVar(&showVersion, "version", false, "")
	flag.IntVar(&debugPort, "debug-port", 8080, "port to listen for debug")
	flag.Parse()

	if showVersion {
		fmt.Println("katsubushi version:", katsubushi.Version)
		return
	}

	if workerID == 0 {
		fmt.Println("please set -worker-id")
		os.Exit(1)
	}

	app, err := katsubushi.NewApp(uint32(workerID))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := app.SetIdleTimeout(idleTimeout); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := app.SetLogLevel(logLevel); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// for profiling
	if enablePprof || enableStats {
		mux := http.NewServeMux()

		if enablePprof {
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			mux.HandleFunc("/debug/pprof/", pprof.Index)
		}

		if enableStats {
			mux.HandleFunc("/debug/stats", stats_api.Handler)
		}

		go func() {
			log.Println(http.ListenAndServe(fmt.Sprintf("localhost:%d", debugPort), mux))
		}()
	}

	if sockpath != "" {
		fmt.Println(app.ListenSock(sockpath))
	} else {
		fmt.Println(app.ListenTCP("", port))
	}
}
