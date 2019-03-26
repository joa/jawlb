package main

func filterEqualServerList(in <-chan ServerList) <-chan ServerList {
	out := make(chan ServerList)

	go func() {
		defer close(out)

		var prev ServerList

		for next := range in {
			if len(prev) == len(next) {
				equal := true

				for i := range prev {
					if !prev[i].Equal(next[i]) {
						equal = false
						break
					}
				}

				if equal {
					continue
				}
			}

			prev = next
			out <- next
		}
	}()

	return out
}
