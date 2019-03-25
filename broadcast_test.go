package main

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestBroadcast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	src := make(chan ServerList)
	broadcast := newBroadcast(ctx, src)
	listener := make(chan ServerList)

	// adding a listener should not cause trouble and do not trigger the listener
	// as long as the broadcast never received a server list
	addDummyListeners(ctx, broadcast)
	broadcast.addListener(listener)
	addDummyListeners(ctx, broadcast)
	assertNoDataReceived(listener, t)

	// broadcasting should work
	srv := Server{net.IP{127, 0, 0, 1}, 1234}
	src <- ServerList{srv}
	assertDataReceived(listener, t, srv)

	// broadcasting the same data twice shouldn't happen
	src <- ServerList{srv}
	assertNoDataReceived(listener, t)

	// broadcasting different data should work
	srv = Server{net.IP{127, 0, 0, 1}, 5678}
	src <- ServerList{srv}
	assertDataReceived(listener, t, srv)

	// removing a listener should mean we won't receive further updates
	broadcast.remListener(listener)
	src <- ServerList{srv}
	assertNoDataReceived(listener, t)

	// adding the listener should result in known state being sent
	broadcast.addListener(listener)
	assertDataReceived(listener, t, srv)
}

func addDummyListeners(ctx context.Context, b *broadcast) {
	for i := 0; i < 10; i++ {
		l := make(chan ServerList)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-l:
				}
			}
		}()
		b.addListener(l)
	}
}

func assertDataReceived(listener chan ServerList, t *testing.T, expected Server) {
	select {
	case list := <-listener:
		if len(list) != 1 {
			t.Errorf("expected server list with len 1, got %d", len(list))
		}

		if list[0].IP.String() != expected.IP.String() {
			t.Errorf("expected ip %s, got %s", expected.IP, list[0].IP)
		}

		if list[0].Port != expected.Port {
			t.Errorf("expected port %d, got %d", expected.Port, list[0].Port)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected listener to receive data")
	}
}

func assertNoDataReceived(listener chan ServerList, t *testing.T) {
	select {
	case <-listener:
		t.Error("listener received data but shouldn't")
	case <-time.After(100 * time.Millisecond):
	}
}
