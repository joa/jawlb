FROM golang:alpine as builder

ENV GO111MODULE on
RUN apk update && apk add git && apk add ca-certificates
RUN adduser -D -g '' unprivileged
COPY . $GOPATH/src/github.com/joa/jawlb/
WORKDIR $GOPATH/src/github.com/joa/jawlb/
RUN go get -d -v
RUN go generate main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/app

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /go/bin/app /go/bin/app
USER unprivileged
EXPOSE 8000
ENTRYPOINT ["/go/bin/app"]
