package main

import (
	"net"
	"testing"
)

func TestConvertServerList(t *testing.T) {
	l := ServerList{Server{net.IP{}, 1}, Server{net.IP{}, 2}, Server{net.IP{}, 3}}

	table := []struct {
		offset int
		want   []int32
	}{
		{0, []int32{1, 2, 3}},
		{1, []int32{2, 3, 1}},
		{2, []int32{3, 1, 2}},
		{3, []int32{1, 2, 3}},
		{4, []int32{2, 3, 1}},
	}

	for _, tt := range table {
		res := convertServerList(l, tt.offset)

		if len(res) != len(tt.want) {
			t.Errorf("expected len %d, got %d", len(tt.want), len(res))
			continue
		}

		for i := range res {
			if res[i].Port != tt.want[i] {
				t.Errorf("expected %d at %d, got %d", tt.want[i], i, res[i].Port)
			}
		}
	}
}
