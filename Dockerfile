# ==============================================================================
# GLOBAL RUNTIME BUILD ARGUMENTS
# ==============================================================================
ARG GANESHA_AUR_URL="https://aur.archlinux.org/nfs-ganesha.git"
ARG ENABLE_DEBUG=true
ARG CGO_ENABLED=1
ARG GOOS=linux
ARG GOARCH=amd64

# ==============================================================================
# STAGE 1: Heavy System, AUR Compilation & Complete Core Staging Prep
# ==============================================================================
# hadolint ignore=DL3007
FROM archlinux:latest AS arch-system-builder

ARG GANESHA_AUR_URL

RUN sed -i 's/^#ParallelDownloads = 5/ParallelDownloads = 10/' /etc/pacman.conf && \
    pacman -Syu --noconfirm --needed \
    base-devel \
    go \
    git \
    libcap \
    samba \
    rpcbind \
    nfs-utils \
    && pacman -Scc --noconfirm

RUN useradd -m -G wheel aur-builder && \
    echo "aur-builder ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

USER aur-builder
WORKDIR /home/aur-builder
RUN git clone "${GANESHA_AUR_URL}" 
WORKDIR /home/aur-builder/nfs-ganesha
RUN makepkg -si --noconfirm --skipchecksums --skippgpcheck

# ==============================================================================
# STAGE 2: Custom Distroless Root Filesystem Setup
# ==============================================================================
FROM arch-system-builder AS distroless-setup
# hadolint ignore=DL3002
USER root
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

RUN mkdir -p /distroless/etc \
    /distroless/usr/bin \
    /distroless/usr/lib \
    /distroless/run/ganesha \
    /distroless/var/run/ganesha \
    /distroless/var/lib/nfs/statd/sm \
    /distroless/var/lib/nfs/statd/sm.bak \
    /distroless/var/lib/nfs/ganesha \
    /distroless/var/lib/samba \
    /distroless/var/log/samba \
    /distroless/var/lock/samba \   
    /distroless/var/cache/samba \  
    /distroless/run/samba \   
    /distroless/run/rpcbind && \
    \
    # 1. Align the filesystem to the Arch Linux standard (Crucial for the dynamic loader)
    ln -s usr/bin /distroless/bin && \
    ln -s usr/bin /distroless/sbin && \
    ln -s bin /distroless/usr/sbin && \
    ln -s usr/lib /distroless/lib && \
    ln -s usr/lib /distroless/lib64 && \
    \
    # 2. Sanitize user accounts via grep (No dbus, no aur-builder)
    grep -E '^root:|^nobody:|^rpc:' /etc/passwd > /distroless/etc/passwd && \
    grep -E '^root:|^nobody:|^rpc:|^wheel:' /etc/group > /distroless/etc/group && \
    \
    # 3. Copy base configurations
    cp -a /etc/hosts /etc/resolv.conf /etc/protocols /etc/services /etc/netconfig /distroless/etc/ && \
    if [ -f /etc/idmapd.conf ]; then cp -a /etc/idmapd.conf /distroless/etc/; fi && \
    echo "hosts: files dns" > /distroless/etc/nsswitch.conf && \
    \
    # 4. Copy system binaries into /usr/bin
    cp /usr/bin/ganesha.nfsd /distroless/usr/bin/ && \
    cp /usr/bin/smbd /distroless/usr/bin/ && \
    cp /usr/bin/nmbd /distroless/usr/bin/ && \
    cp /usr/bin/rpcbind /distroless/usr/bin/ && \
    cp /usr/bin/rpc.statd /distroless/usr/bin/ && \
    \
    # 5. Map native libraries into /usr/lib
    mkdir -p /distroless/usr/lib/ganesha && \
    cp -a /usr/lib/ganesha/* /distroless/usr/lib/ganesha/ && \
    cp -vnP /usr/lib/libnss_files* /distroless/usr/lib/ && \
    cp -vnP /usr/lib/libnss_dns* /distroless/usr/lib/ && \
    cp -vnP /usr/lib/libnfsidmap.so* /distroless/usr/lib/ && \
    mkdir -p /distroless/usr/lib/libnfsidmap && \
    cp -a /usr/lib/libnfsidmap/* /distroless/usr/lib/libnfsidmap/



# ==============================================================================
# STAGE 3: Fast Go Binary Builder & Master Dependency Scanner
# ==============================================================================
FROM arch-system-builder AS go-binary-builder
# hadolint ignore=DL3002
USER root
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG ENABLE_DEBUG
ARG CGO_ENABLED
ARG GOOS
ARG GOARCH

COPY --from=distroless-setup /distroless /distroless

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN if [ "$ENABLE_DEBUG" = "true" ] ; then \
    CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} go build -a -v -tags=debug -ldflags="-s -w" -o /gorogs main.go; \
    else \
    CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} go build -a -v -ldflags="-s -w" -o /gorogs main.go; \
    fi

# Inject gorogs into the correct canonical binary path
RUN cp /gorogs /distroless/usr/bin/gorogs && chmod +x /distroless/usr/bin/gorogs

# Master Scan: Resolves and copies libraries directly into /usr/lib/
# Dereferences symlinks (-L) on the second pass to guarantee files like libresolv.so.2 are actual readable files
RUN for bin in /distroless/usr/bin/* /distroless/usr/lib/ganesha/*.so; do \
    ldd "$bin" 2>/dev/null | awk 'match($0, /\/[^ ]+/) {print substr($0, RSTART, RLENGTH)}' | while read -r lib; do \
    cp -vnL "$lib" "/distroless/usr/lib/$(basename "$lib")"; \
    done; \
    done

# ==============================================================================
# STAGE 4: Final Distroless Runtime Assembly
# ==============================================================================
FROM scratch AS final

WORKDIR /
COPY --from=go-binary-builder /distroless/. /

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/bin/gorogs", "--check-health"]

EXPOSE \
    # ==========================================
    # SAMBA / NETBIOS / WINDOWS DISCOVERY
    # ==========================================
    # Microsoft RPC Endpoint Mapper (Often needed by modern Windows clients)
    135/tcp \
    # NetBIOS Name Service (nmbd)
    137/udp \
    # NetBIOS Datagram Service (nmbd)
    138/udp \
    # NetBIOS Session Service (smbd)
    139/tcp \
    # SMB over TCP / Active Directory Direct Host (smbd)
    445/tcp \
    # WS-Discovery / Web Services Dynamic Discovery (Samba network browsing)
    3702/udp \
    # WSDAPI / Web Services for Devices (Samba network browsing HTTP)
    5357/tcp \
    # ==========================================
    # RPCBIND / NFS INFRASTRUCTURE
    # ==========================================
    # RPC Endpoint Mapper / Portmapper (rpcbind)
    111/tcp 111/udp \
    # Network File System core (nfsd / ganesha.nfsd)
    2049/tcp 2049/udp \
    # NFS Mount Daemon (mountd / ganesha.nfsd MNT)
    20048/tcp 20048/udp \
    # NFS Network Lock Manager (lockd / NLM status)
    32803/tcp 32803/udp \
    # NFS Remote Quota Daemon (rquotad / RQUOTA status)
    875/tcp 875/udp


ENTRYPOINT ["usr/bin/gorogs"]
