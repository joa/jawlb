package main

import (
	grpclb "google.golang.org/grpc/balancer/grpclb/grpc_lb_v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type lb struct {
	b *broadcast
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
			var servers []*grpclb.Server

			for _, server := range msg {
				servers = append(servers, &grpclb.Server{IpAddress: server.IP, Port: server.Port})
			}

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
