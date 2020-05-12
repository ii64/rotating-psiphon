FROM golang:alpine AS builder
WORKDIR $GOPATH/src/github.com/ii64/rotating-psiphon
RUN apk update && apk add --no-cache git
COPY . .

RUN go get -d -v
RUN GOOS=linux GOARCH=amd64 go build -v -o /go/bin/start_app psi-scale.go


# final
FROM ubuntu:14.04
MAINTAINER ii64 <nekonify@gmail.com>
WORKDIR /app
COPY --from=builder /go/bin/start_app /app/start_app
COPY ./desktop/ /app/desktop/
COPY ./psiphon-tunnel-core /app/psiphon-tunnel-core
COPY ./psiphon-tunnel-core.exe /app/psiphon-tunnel-core.exe

ENV PATH="/app:${PATH}"

RUN chmod +x /app/start_app
RUN chmod +x /app/psiphon-tunnel-core
RUN apt-get update && \
	apt-get install -y haproxy


#ENV INSTANCE_COUNT 5
#ENV HAPROXY_CONFIG /etc/haproxy/haproxy.cfg

# HA Proxy monitor dashboard /haproxy?stats
EXPOSE 4444/tcp

ENTRYPOINT ["/app/start_app"]