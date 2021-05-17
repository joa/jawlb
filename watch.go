package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func watchService(ctx context.Context) (_ <-chan ServerList, err error) {
	icc, err := getConfig()

	if err != nil {
		return
	}

	client, err := kubernetes.NewForConfig(icc)

	if err != nil {
		return
	}

	ep := startWatch(ctx, client)
	ch := make(chan ServerList)
	ticker := time.NewTicker(cfg.WatchTimeout)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				go stopWatch(ep)
				close(ch)
				return
			case <-ticker.C:
				log.Printf("restarting the watch after timeout")
				go stopWatch(ep)
				ep = startWatch(ctx, client)
			case res := <-ep.ResultChan():
				endpoint, ok := res.Object.(*v1.Endpoints)

				if !ok {
					log.Printf("watch encountered an error: %+v", res.Object)
					go stopWatch(ep)
					ep = startWatch(ctx, client)
					continue
				}

				if cfg.Namespace != "" && endpoint.Namespace != cfg.Namespace {
					continue
				}

				if endpoint.Name != cfg.Service {
					continue
				}

				var ips []string
				var ports []int32

				for _, subset := range endpoint.Subsets {
					for _, addr := range subset.Addresses {
						ips = append(ips, addr.IP)
					}

					for _, port := range subset.Ports {
						if cfg.TargetPort == "" || port.Name == cfg.TargetPort {
							ports = append(ports, port.Port)
						}
					}
				}

				var servers []Server

				for _, addr := range ips {
					for _, port := range ports {
						servers = append(servers, Server{IP: net.ParseIP(addr), Port: port})
					}
				}

				ch <- servers
			}
		}
	}()

	return ch, nil
}

func startWatch(ctx context.Context, client *kubernetes.Clientset) watch.Interface {
	log.Printf("start watching endpoints in '%s' with labels '%s'", cfg.Namespace, cfg.LabelSelector)

	for i := 0; i < cfg.WatchMaxRetries; i++ {
		ep, err := client.CoreV1().Endpoints(cfg.Namespace).Watch(ctx, meta_v1.ListOptions{
			LabelSelector: cfg.LabelSelector,
			Watch:         true,
		})

		if err == nil {
			return ep
		}

		log.Println("couldn't start watch:", err.Error())
		log.Printf("retrying in %s ...", cfg.WatchRetryDelay)
		time.Sleep(cfg.WatchRetryDelay)
	}

	panic(fmt.Sprintf("couldn't start watch after %d retries", cfg.WatchMaxRetries))
}

func stopWatch(w watch.Interface) {
	log.Printf("stopping a watch endpoint")
	done := make(chan bool)

	go func() {
		w.Stop()
		done <- true
	}()

	select {
	case <-time.After(5 * time.Minute):
		log.Println("closing a watch timed out after 5m")
	case <-done:
		log.Println("successfully stopped a watch")
	}
}

func getConfig() (cfg *rest.Config, err error) {
	cfg, err = rest.InClusterConfig()

	if err != nil {
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		)

		cfg, err = loader.ClientConfig()

		if err != nil {
			return
		}
	}

	return
}
