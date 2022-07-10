FROM golang:1.18.3-alpine3.16 as builder

WORKDIR /app/

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o golinks .

FROM alpine:3.16.0 as tailscaler

WORKDIR /app/

ENV TSFILE=tailscale_1.26.2_amd64.tgz
RUN wget https://pkgs.tailscale.com/stable/${TSFILE} && tar xzf ${TSFILE} --strip-components=1

FROM alpine:3.16.0

WORKDIR /app/

RUN apk update && apk add ca-certificates iptables ip6tables && rm -rf /var/cache/apk/*

# Copy binary to production image
COPY --from=builder /app/golinks /app/golinks
COPY --from=tailscaler /app/tailscaled /app/tailscaled
COPY --from=tailscaler /app/tailscale /app/tailscale
RUN mkdir -p /var/run/tailscale /var/cache/tailscale /var/lib/tailscale

COPY scripts/start.sh /app/start.sh

CMD "/app/start.sh"