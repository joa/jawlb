package main

import "context"

func newServerListFilter(ctx context.Context, in <-chan ServerList) <-chan ServerList {
	out := make(chan ServerList)
	go doFilter(ctx, in, out)
	return out
}

func doFilter(ctx context.Context, in <-chan ServerList, out chan ServerList) {
	var prev ServerList

looping:
	for {
		select {
		case <-ctx.Done():
			return
		case next := <-in:
			if len(prev) == len(next) {
				equal := true

				// make a sacrifice for the ssa range check elimination gods
				n := len(prev)
				_ = prev[n-1]
				_ = next[n-1]

				for i := range prev {
					if !prev[i].Equal(next[i]) {
						equal = false
						break
					}
				}

				if equal {
					continue looping
				}
			}

			prev = next
			out <- next
		}
	}
}
