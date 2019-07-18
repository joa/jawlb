package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	grpclb "google.golang.org/grpc/balancer/grpclb/grpc_lb_v1"
)

var cfg = struct {
	Host                string        `default:"" desc:"Hostname to listen on"`
	Port                int           `default:"8000" desc:"Port of the grpclb server"`
	ShutdownGracePeriod time.Duration `default:"25s" desc:"Duration of graceful shutdown period"` // during this time, we try to answer open reqs but won't accept new ones

	Namespace     string `default:"default" desc:"Kubernetes namespace in which to operate; empty for all namespaces"`
	Service       string `desc:"Name of the service in Kubernetes" required:"true"`
	LabelSelector string `desc:"Label selector for the service (foo=bar,baz=bang)"`
	TargetPort    string `default:"grpc" desc:"Target port name to forward to"`
	MaxServers    int    `default:"0" desc:"Maximum number of servers to return in response, 0 means unlimited"`

	WatchMaxRetries int           `default:"60" desc:"Number of times to retry establishing the Kubernetess watch"`
	WatchRetryDelay time.Duration `default:"1s" desc:"Delay between retries"`
	WatchTimeout    time.Duration `default:"30s" desc:"Timeout for the watch until we acquire a new one"`
}{}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configure()

	ch, err := watchService(ctx)

	if err != nil {
		log.Panic(err)
	}

	bc := newBroadcast(ctx, ch)

	srv := startServer(bc)

	logChanges(ctx, bc)

	log.Println("waiting for TERM")
	awaitTerm()

	awaitShutdown(srv)
	log.Println("bye")
}

func logChanges(ctx context.Context, bc *broadcast) {
	ch := make(chan ServerList)
	bc.addListener(ch)

	go func() {
		for {
			select {
			case <-ctx.Done():
				bc.remListener(ch)
				close(ch)
				return
			case msg := <-ch:
				log.Print("endpoints:")
				for _, server := range msg {
					log.Printf("\t%s:%d", server.IP, server.Port)
				}
			}
		}
	}()
}

func configure() {
	envconfig.MustProcess("JAWLB", &cfg)
	if len(os.Args) > 1 && os.Args[1] == "help" {
		if err := envconfig.Usage("JAWLB", &cfg); err != nil {
			log.Panic(err)
		}
	}
}

func startServer(bc *broadcast) *grpc.Server {
	// setup listening socket
	conn, err := listen()

	if err != nil {
		log.Panic(err)
	}

	srv := grpc.NewServer()
	grpclb.RegisterLoadBalancerServer(srv, &lb{bc, cfg.MaxServers})

	go func() {
		if err := srv.Serve(conn); err != nil {
			log.Println("grpc connection closed", err)
		}
	}()

	return srv
}

func listen() (conn net.Listener, err error) {
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	conn, err = net.Listen("tcp", addr)
	return
}

func awaitTerm() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig,
		syscall.SIGTERM,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT)
	<-sig
}

func awaitShutdown(server *grpc.Server) {
	log.Println("performing graceful shutdown")

	done := make(chan bool)

	go func() {
		server.GracefulStop()
		done <- true
	}()

	select {
	case <-time.After(cfg.ShutdownGracePeriod):
		log.Println("graceful shutdown failed -- hard stop")
		server.Stop()
	case <-done:
	}
}
