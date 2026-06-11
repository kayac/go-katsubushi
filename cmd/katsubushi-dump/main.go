package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	katsubushi "github.com/kayac/go-katsubushi/v2"
)

type Dump struct {
	Time     time.Time `json:"time"`
	WorkerID uint64    `json:"worker_id"`
	Sequence uint64    `json:"sequence"`
}

func main() {
	var jsSafe bool
	flag.BoolVar(&jsSafe, "js-safe", false, "dump IDs in the JS-safe format")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "no id")
		os.Exit(1)
	}
	dump := katsubushi.Dump
	if jsSafe {
		dump = katsubushi.DumpJSSafe
	}
	enc := json.NewEncoder(os.Stdout)
	for _, s := range flag.Args() {
		if id, err := strconv.ParseUint(s, 10, 64); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		} else {
			if !jsSafe && id <= katsubushi.JSSafeMaxID {
				fmt.Fprintf(os.Stderr, "warning: %d seems to be an ID in the JS-safe format. use -js-safe to dump it correctly\n", id)
			} else if jsSafe && id > katsubushi.JSSafeMaxID {
				fmt.Fprintf(os.Stderr, "warning: %d exceeds 2^53-1, it seems to be an ID in the default format. remove -js-safe to dump it correctly\n", id)
			}
			t, wid, seq := dump(id)
			enc.Encode(Dump{t, wid, seq})
		}
	}
}
