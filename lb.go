package main

import (
	"math/rand"
	"time"

	grpclb "google.golang.org/grpc/balancer/grpclb/grpc_lb_v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type lb struct {
	b   *broadcast
	max int
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

	for {
		select {
		case <-req.Context().Done():
			return nil
		case msg := <-ch:
			servers := convertServerList(msg, l.max)

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

func convertServerList(l ServerList, max int) []*grpclb.Server {
	n := len(l)
	if max > n || max == 0 {
		max = n
	}
	servers := make([]*grpclb.Server, max)

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < max; i++ {
		j := rand.Intn(n)
		l[i], l[j] = l[j], l[i]
		servers[i] = convertServer(l[i])
	}

	return servers
}

func convertServer(s Server) *grpclb.Server {
	return &grpclb.Server{IpAddress: s.IP, Port: s.Port}
}
