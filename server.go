package main

import "net"

type ServerList []Server

type Server struct {
	IP   net.IP
	Port int32
}

func (s Server) Equal(x Server) bool {
	return s.Port == x.Port && s.IP.Equal(x.IP)
}
