# build up psi-scale
FROM golang:alpine AS builder
WORKDIR $GOPATH/src/github.com/ii64/rotating-psiphon
RUN apk update && apk add --no-cache git
COPY . .

RUN go get -d -v
RUN GOOS=linux GOARCH=amd64 go build -v -o /go/bin/start_app psi-scale.go

# final img
FROM haproxy:latest
MAINTAINER ii64 <nekonify@gmail.com>
WORKDIR /app
COPY --from=builder /go/bin/start_app /app/start_app
COPY ./desktop/ /app/desktop/
COPY ./psiphon-tunnel-core /app/psiphon-tunnel-core

RUN chmod +x /app/start_app
RUN chmod +x /app/psiphon-tunnel-core
ENV PATH="/app:${PATH}"

# psi-scale configuration - or set on docker-compose
ENV INSTANCE_COUNT 5
ENV HAPROXY_CONFIG /app/haproxy.cfg

# HA Proxy monitor dashboard /haproxy?stats
EXPOSE 4444/tcp
# HA Proxy frontend for psiphons
EXPOSE 4455/tcp
# direct app entrypoint
ENTRYPOINT ["/app/start_app"]
