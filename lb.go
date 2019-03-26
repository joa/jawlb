package main

import (
	"github.com/joa/jawlb/internal/atomic"
	grpclb "google.golang.org/grpc/balancer/grpclb/grpc_lb_v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type lb struct {
	b  *broadcast
	rr int64
}

func (l *lb) BalanceLoad(req grpclb.LoadBalancer_BalanceLoadServer) error {
	if in, err := req.Recv(); err != nil {
		return err
	} else if init := in.GetInitialRequest(); init != nil {
		err := req.Send(&grpclb.LoadBalanceResponse{
			LoadBalanceResponseType: &grpclb.LoadBalanceResponse_InitialResponse{
				InitialResponse: &grpclb.InitialLoadBalanceResponse{},
			},
		})

		if err != nil {
			return err
		}
	} else {
		return status.Error(codes.InvalidArgument, "expected initial request")
	}

	ch := make(chan ServerList)
	defer close(ch)

	l.b.addListener(ch)
	defer l.b.remListener(ch)

	offset := int(atomic.IncWrapInt64(&l.rr))

	for {
		select {
		case <-req.Context().Done():
			return nil
		case msg := <-ch:
			servers := convertServerList(msg, offset)

			err := req.Send(&grpclb.LoadBalanceResponse{
				LoadBalanceResponseType: &grpclb.LoadBalanceResponse_ServerList{
					ServerList: &grpclb.ServerList{
						Servers: servers,
					},
				},
			})

			if err != nil {
				return err
			}
		}
	}
}

func convertServerList(l ServerList, offset int) []*grpclb.Server {
	var servers []*grpclb.Server

	n := len(l)

	for i := 0; i < n; i++ {
		server := l[(i+offset)%n]
		servers = append(servers, convertServer(server))
	}

	return servers
}

func convertServer(s Server) *grpclb.Server {
	return &grpclb.Server{IpAddress: s.IP, Port: s.Port}
}
