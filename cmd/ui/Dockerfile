FROM node:latest as ui-builder

RUN npm install -g react-scripts && \
		npm install -g ajv && \
		npm install -g react && \
		npm install -g react-dom && \
		npm install -g react-websocket && \
		npm install -g @material-ui && \
		npm install -g d3

COPY ./cmd/ui/frontend /ui
WORKDIR /ui

RUN npm install --save
RUN npm run build

# ---------------------------------------------------------------------------------------------

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

FROM alpine:latest

COPY --from=backend-builder /go/bin/backend /ui
COPY --from=ui-builder /ui/build /static
CMD ["/ui"]
