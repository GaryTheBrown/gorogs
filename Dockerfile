FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /gorogs main.go

FROM debian:trixie-slim
ENV DEBIAN_FRONTEND=noninteractive

# hadolint ignore=DL3008
RUN apt-get update && apt-get install -y --no-install-recommends -t trixie \
    libcap2-bin \
    nfs-ganesha \
    nfs-ganesha-vfs \
    samba \
    rpcbind \
    nfs-common \
    && rm -rf /var/lib/apt/lists/* /var/cache/apt/* /usr/share/doc/* /usr/share/man/* /usr/share/locale/* \
    && usermod -d /tmp -g nogroup nobody

WORKDIR /
COPY --from=builder /gorogs /usr/local/bin/gorogs
RUN setcap cap_dac_read_search+ep /usr/bin/ganesha.nfsd && chmod +x /usr/local/bin/gorogs
# RUN setcap cap_dac_read_search,cap_sys_resource+ep /usr/bin/ganesha.nfsd && chmod +x /usr/local/bin/gorogs

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/usr/local/bin/gorogs", "--check-health"]

EXPOSE 137/udp 138/udp 139/tcp 445/tcp 3702/udp 5357/tcp \
    111/tcp 111/udp 2049/tcp 2049/udp 892/tcp 892/udp 4045/tcp 4045/udp 875/tcp 875/udp

ENTRYPOINT ["/usr/local/bin/gorogs"]
