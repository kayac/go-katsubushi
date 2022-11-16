package katsubushi_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/kayac/go-katsubushi/v2"
	"github.com/kayac/go-katsubushi/v2/grpc"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var grpcApp *katsubushi.App
var grpcPort int

func init() {
	var err error
	grpcApp, err = katsubushi.New(88)
	if err != nil {
		panic(err)
	}
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	grpcPort = listener.Addr().(*net.TCPAddr).Port
	go grpcApp.RunGRPCServer(context.Background(), &katsubushi.Config{GRPCListener: listener})
	time.Sleep(3 * time.Second)
}

func newgRPCClient() (grpc.GeneratorClient, func(), error) {
	addr := fmt.Sprintf("localhost:%d", grpcPort)
	conn, err := gogrpc.Dial(
		addr,
		gogrpc.WithTransportCredentials(insecure.NewCredentials()),
		gogrpc.WithBlock(),
	)
	if err != nil {
		return nil, func() {}, err
	}
	c := grpc.NewGeneratorClient(conn)
	return c, func() { conn.Close() }, nil
}

func TestGRPCSingle(t *testing.T) {
	client, close, err := newgRPCClient()
	defer close()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		res, err := client.Fetch(context.Background(), &grpc.FetchRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Id == 0 {
			t.Fatal("id should not be 0")
		}
		t.Logf("HTTP fetched single ID: %d", res.Id)
	}
}

func TestGRPCMulti(t *testing.T) {
	client, close, err := newgRPCClient()
	defer close()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		res, err := client.FetchMulti(context.Background(), &grpc.FetchMultiRequest{N: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Ids) != 10 {
			t.Fatalf("ids should contain 10 elements %v", res.Ids)
		}
		for _, id := range res.Ids {
			if id == 0 {
				t.Fatal("id should not be 0")
			}
		}
		t.Logf("HTTP fetched IDs: %v", res.Ids)
	}
}

func BenchmarkGRPCClientFetch(b *testing.B) {
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		c, close, _ := newgRPCClient()
		defer close()
		for pb.Next() {
			res, err := c.Fetch(context.Background(), &grpc.FetchRequest{})
			if err != nil {
				b.Fatal(err)
			}
			if res.Id == 0 {
				b.Error("could not fetch id > 0")
			}
		}
	})
}
