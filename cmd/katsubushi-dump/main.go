package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	katsubushi "github.com/kayac/go-katsubushi"
)

type Dump struct {
	Time     time.Time `json:"time"`
	WorkerID uint64    `json:"worker_id"`
	Sequence uint64    `json:"sequence"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "no id")
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	for _, s := range os.Args[1:] {
		if id, err := strconv.ParseUint(s, 10, 64); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		} else {
			t, wid, seq := katsubushi.Dump(id)
			enc.Encode(Dump{t, wid, seq})
		}
	}
}
