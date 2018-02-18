FROM golang:1.9
WORKDIR /go/src/github.com/coldog/etcd-aws-cluster
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
  -o /go/bin/etcd-config \
  -a -tags netgo -ldflags '-w' \
  github.com/coldog/etcd-aws-cluster/cmd/etcd-config
RUN CGO_ENABLED=0 GOOS=linux go build \
  -o /go/bin/etcd-backupd \
  -a -tags netgo -ldflags '-w' \
  github.com/coldog/etcd-aws-cluster/cmd/etcd-backupd
RUN CGO_ENABLED=0 GOOS=linux go build \
  -o /go/bin/etcd-watcherd \
  -a -tags netgo -ldflags '-w' \
  github.com/coldog/etcd-aws-cluster/cmd/etcd-watcherd

FROM alpine:3.4
RUN apk --update --no-cache add ca-certificates
COPY --from=0 /go/bin/ /bin/
VOLUME ["/root/.aws", "/etc/etcd/", "/etc/ssl/"]
CMD ["/bin/etcd-config"]
