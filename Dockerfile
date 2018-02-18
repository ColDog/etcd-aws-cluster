FROM alpine:3.4

RUN apk --update --no-cache add \
  python \
  py-pip \
  jq \
  curl \
  wget \
  bash && \
  pip install --upgrade awscli &&\
  mkdir /root/.aws

COPY etcd-aws-cluster /bin/etcd-aws-cluster
VOLUME ["/root/.aws", "/etc/etcd/"]
ENTRYPOINT [ "/bin/etcd-aws-cluster" ]
