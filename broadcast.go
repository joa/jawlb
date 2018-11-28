package main

import (
	"context"
)

type Listener chan<- ServerList

type broadcast struct {
	ctx   context.Context
	src   <-chan ServerList
	state ServerList
	tgts  map[Listener]bool
	add   chan Listener
	rem   chan Listener
}

func newBroadcast(ctx context.Context, src <-chan ServerList) *broadcast {
	b := &broadcast{
		ctx:  ctx,
		src:  src,
		tgts: make(map[Listener]bool),
		add:  make(chan Listener),
		rem:  make(chan Listener),
	}

	go b.run()

	return b
}

func (b *broadcast) addListener(listener Listener) {
	b.add <- listener

	// send initial state if present once on registration
	if len(b.state) > 0 {
		go func() { listener <- b.state }()
	}
}

func (b *broadcast) remListener(listener Listener) {
	b.rem <- listener
}

func (b *broadcast) run() {
	for {
		select {
		case <-b.ctx.Done():
			close(b.add)
			close(b.rem)
			return
		case l := <-b.add:
			b.tgts[l] = true
		case l := <-b.rem:
			delete(b.tgts, l)
		case state := <-b.src:
			b.state = state
			for tgt := range b.tgts {
				go func() { tgt <- state }()
			}
		}
	}
}
