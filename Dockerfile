FROM golang:1.9
WORKDIR /go/src/github.com/coldog/etcd-aws-cluster
COPY . .
ENV CGO_ENABLED=0 GOOS=linux BUILD_FLAGS="-a -installsuffix cgo"
RUN go build ${BUILD_FLAGS} \
  -o /go/bin/etcd-aws-cluster \
  github.com/coldog/etcd-aws-cluster/cmd/etcd-aws-cluster

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=0 /go/bin/ /bin/
VOLUME ["/root/.aws", "/etc/etcd/"]
ENTRYPOINT ["/bin/etcd-aws-cluster"]
