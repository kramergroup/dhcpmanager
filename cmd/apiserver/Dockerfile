FROM golang:latest as Builder

# go get most dependencies before copying in the source to cache them
RUN go get github.com/digineo/go-dhclient github.com/gorilla/mux \
           github.com/coreos/etcd/clientv3 github.com/spf13/viper \
           github.com/digineo/go-dhclient github.com/vishvananda/netlink

# Copy sources in
COPY . /go/src/github.com/kramergroup/dhcpmanager
WORKDIR /go/src/github.com/kramergroup/dhcpmanager/cmd/apiserver

RUN go get
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/apiserver .

FROM alpine:latest

COPY --from=builder /go/bin/apiserver /apiserver
CMD ["/apiserver"]
