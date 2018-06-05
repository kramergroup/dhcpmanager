FROM golang:latest as backend-builder

# go get most dependencies before copying in the source to cache them
RUN go get github.com/gorilla/mux github.com/gorilla/websocket \
	         github.com/kramergroup/dhcpmanager github.com/spf13/viper

# Copy sources in
COPY . /go/src/github.com/kramergroup/dhcpmanager
WORKDIR /go/src/github.com/kramergroup/dhcpmanager/cmd/ui/backend

RUN go get
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/backend .

# ---------------------------------------------------------------------------------------------

FROM mhart/alpine-node as ui-builder

COPY ./cmd/ui/frontend /ui
WORKDIR /ui

RUN npm install && npm run build

# ---------------------------------------------------------------------------------------------

FROM alpine:latest

COPY --from=backend-builder /go/bin/backend /ui
COPY --from=ui-builder /ui/build /static
CMD ["/ui"]