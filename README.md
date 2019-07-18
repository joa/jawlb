### Jawlb
Jawlb (pronounced jolp) is an unsophisticated grpclb implementation for things running in Kubernetes talking to gRPC 
services within that same Kubernetes cluster.

This load balancer performs  service discovery via the Kubernetes API and announces any changes it sees
via the grpclb protocol to its clients.

### Building
You'll want to push the result of `docker build -f Dockerfile .` into your registry.

### grpclb
The grpclb Go implementation (others maybe too) requires you to define a SRV record for the loadbalancer. 
The easy solution is to define a Kubernetes service with a port named `grpclb` and you're all set.

### Example
The example shows a load balancer setup that'll forward requests to `myservice:grpc`. 
We assume that RBAC isn't required.

#### Load Balancer Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myservice-lb
spec:
  selector:
    matchLabels:
      app: jawlb
      service: myservice
  replicas: 3
  template:
    metadata:
      labels:
        app: jawlb
        service: myservice
    spec:
      restartPolicy: Always
      containers:
      - image: "your.regist.ry/jawlb:yolo"
        name: jawlb
        ports:
        - containerPort: 8000
          name: grpclb
        readinessProbe:
          tcpSocket:
            port: grpclb
          initialDelaySeconds: 5
          periodSeconds: 10
        livenessProbe:
          tcpSocket:
            port: grpclb
          initialDelaySeconds: 15
          periodSeconds: 60
        env:
        # Maximum number of servers to return in response, # default is 0 and it means unlimited
        - name: JAWLB_MAXSERVERS
          value: "5"
        # The name of the upstream service we want
        # to balance
        - name: JAWLB_SERVICE
          value: "myservice"
        # The name of the port exposed by this service
        # which we want to forward to
        - name: JAWLB_TARGETPORT
          value: "grpc"
        # The namespace in which jawlb performs the lookup
        # and we use the current one simply
        - name: JAWLB_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
```

#### Load Balancer Service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: myservice-lb
  labels:
    app: jawlb
    service: myservice
spec:
  # Make this a headless service because we'll have to use
  # the dns:// resolver in the Go client
  clusterIP: None
  ports:
  # The port MUST be named grpclb in order to create
  # the proper DNS SRV entry
  - name: grpclb
    port: 8000
    targetPort: grpclb
  selector:
    app: jawlb
    service: myservice
```

#### gRPC Client
```go
import 	_ "google.golang.org/grpc/balancer/grpclb"

// When dialing, gRPC's DNS resolver will issue a SRV lookup and
// because we're so nice to provide the grpclb entry, everything
// works as expected
//
// If no SRV record exists, gRPC will fall back to a vanilla connection
// without the loadbalancer.

conn, err := grpc.Dial(
	"dns:///myservice-lb",        // must use the dns resolver
	grpc.WithInsecure())

// ... magic 🧙‍♀️
```

### Configuration
Everything is passed via environment variables.

- `JAWLB_NAMESPACE` in which namespace service lookup is performed, default `"default"`
- `JAWLB_SERVICE` the name of the Kubernetes service to balance, required
- `JAWLB_TARGETPORT` the name(!) of the target port on that service, default `"grpc"`
- `JAWLB_LABELSELECTOR` an additional label selector, default `""`
- `JAWLB_HOST` the hostname to listen on, default`""`
- `JAWLB_PORT` port to listen on, default `8000`
- `JAWLB_SHUTDOWNGRACEPERIOD` Grace period for open connections during shutdown, default `"25s"`

### What's missing
- Potentially some actual load balancing
- Implementation of [LoadReporter](https://github.com/grpc/grpc/blob/master/src/proto/grpc/lb/v1/load_reporter.proto) service
- Readiness Probe 
- Health Check

### Resources
- [grpclb spec](https://github.com/grpc/grpc/tree/master/src/proto/grpc/lb/v1)
- [Load Balancing in gRPC](https://github.com/grpc/grpc/blob/master/doc/load-balancing.md)
