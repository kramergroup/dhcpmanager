FROM node:latest as ui-builder

WORKDIR /ui

RUN npm install react-scripts ajv react react-dom react-websocket \
								@material-ui/core @material-ui/icons prop-types d3 whatwg-fetch

COPY ./cmd/ui/frontend /ui

RUN npm run build

# ---------------------------------------------------------------------------------------------

FROM golang:latest as backend-builder

# go get most dependencies before copying in the source to cache them
RUN go get github.com/gorilla/mux github.com/gorilla/websocket github.com/gorilla/handlers \
	         github.com/spf13/viper

RUN go get github.com/coreos/etcd github.com/vishvananda/netlink github.com/google/uuid
RUN go get github.com/digineo/go-dhclient

# Copy sources in
COPY . /go/src/github.com/kramergroup/dhcpmanager
WORKDIR /go/src/github.com/kramergroup/dhcpmanager/cmd/ui/backend

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/backend .

# ---------------------------------------------------------------------------------------------

FROM alpine:latest

COPY --from=backend-builder /go/bin/backend /ui
COPY --from=ui-builder /ui/build /static
CMD ["/ui"]
