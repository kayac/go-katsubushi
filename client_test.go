package katsubushi

import (
	"context"
	"testing"
	"time"
)

func TestClientFetch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app := newTestAppAndListenTCP(ctx, t)
	c := NewClient(app.Listener.Addr().String())

	id, err := c.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Error("could not fetch id > 0")
	}
	t.Logf("fetched id %d", id)
}

func TestClientMulti(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app := newTestAppAndListenTCP(ctx, t)
	c := NewClient(app.Listener.Addr().String())

	ids, err := c.FetchMulti(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 {
		t.Error("FetchMulti != 3")
	}
	for _, id := range ids {
		if id == 0 {
			t.Error("could not fetch id > 0")
		}
		t.Logf("fetched id %d", id)
	}
}

func TestClientFetchRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app := newTestAppAndListenTCP(ctx, t)
	app.SetIdleTimeout(1)

	c := NewClient(app.Listener.Addr().String())

	for i := 0; i < 3; i++ {
		id, err := c.Fetch()
		if err != nil {
			t.Fatal(err)
		}
		if id == 0 {
			t.Error("could not fetch id > 0")
		}
		t.Logf("fetched id %d", id)
		time.Sleep(2 * time.Second) // reset by peer by idle timeout
	}
}

func TestClientFetchBackup(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	app1 := newTestAppAndListenTCP(ctx1, t)

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	app2 := newTestAppAndListenTCP(ctx2, t)

	c := NewClient(
		app1.Listener.Addr().String(),
		app2.Listener.Addr().String(),
	)

	{
		// fetched from app1
		id, err := c.Fetch()
		if err != nil {
			t.Fatal(err)
		}
		if id == 0 {
			t.Error("could not fetch id > 0")
		}
	}

	// shutdown app1
	cancel1()

	{
		// fetched from app2
		id, err := c.Fetch()
		if err != nil {
			t.Fatal(err)
		}
		if id == 0 {
			t.Error("could not fetch id > 0")
		}
	}
}

func TestClientFail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	app := newTestAppAndListenTCP(ctx, t)

	c := NewClient(
		app.Listener.Addr().String(),
	)

	cancel()

	_, err := c.Fetch()
	if err == nil {
		t.Error("must be failed")
	}
	t.Logf("error: %s", err)
}

func TestClientFailMulti(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	app := newTestAppAndListenTCP(ctx, t)

	c := NewClient(
		app.Listener.Addr().String(),
	)

	cancel()

	_, err := c.FetchMulti(3)
	if err == nil {
		t.Error("must be failed")
	}
	t.Logf("error: %s", err)
}

func TestClientFailBackup(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	app1 := newTestAppAndListenTCP(ctx1, t)

	ctx2, cancel2 := context.WithCancel(context.Background())
	app2 := newTestAppAndListenTCP(ctx2, t)

	c := NewClient(
		app1.Listener.Addr().String(),
		app2.Listener.Addr().String(),
	)

	cancel1()
	cancel2()

	_, err := c.Fetch()
	if err == nil {
		t.Error("must be failed")
	}
	t.Logf("error: %s", err)
}

func TestClientFailBackupMulti(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	app1 := newTestAppAndListenTCP(ctx1, t)

	ctx2, cancel2 := context.WithCancel(context.Background())
	app2 := newTestAppAndListenTCP(ctx2, t)

	c := NewClient(
		app1.Listener.Addr().String(),
		app2.Listener.Addr().String(),
	)

	cancel1()
	cancel2()

	_, err := c.FetchMulti(3)
	if err == nil {
		t.Error("must be failed")
	}
	t.Logf("error: %s", err)
}
