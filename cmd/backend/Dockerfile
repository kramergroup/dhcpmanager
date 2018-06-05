FROM golang:latest as Builder

# go get most dependencies before copying in the source to cache them
RUN go get github.com/gorilla/mux github.com/gorilla/websocket \
	         github.com/kramergroup/dhcpmanager github.com/spf13/viper

# Copy sources in
COPY . /go/src/github.com/kramergroup/dhcpmanager
WORKDIR /go/src/github.com/kramergroup/dhcpmanager/cmd/backend

RUN go get
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/backend .

FROM alpine:latest

COPY --from=builder /go/bin/backend /backend
CMD ["/backend"]
