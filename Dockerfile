# ==============================================================================
# GLOBAL RUNTIME BUILD ARGUMENTS
# ==============================================================================
ARG GANESHA_AUR_URL="https://aur.archlinux.org/nfs-ganesha.git"
ARG ENABLE_DEBUG=false
ARG CGO_ENABLED=1
ARG GOOS=linux
ARG GOARCH=amd64

# ==============================================================================
# STAGE 1: Heavy System, AUR Compilation & Complete Core Staging Prep
# ==============================================================================
# hadolint ignore=DL3007
FROM archlinux:latest AS system-builder

ARG GANESHA_AUR_URL

RUN sed -i 's/^#ParallelDownloads = 5/ParallelDownloads = 10/' /etc/pacman.conf && \
    pacman -Syu --noconfirm --needed \
    base-devel \
    go \
    git \
    libcap \
    samba \
    libcups \
    rpcbind \
    nfs-utils \
    ca-certificates \
    # THIS IS LEFT IN ONLY FOR IF WE NEED TO TRACK DOWN MISSING FILES.
    strace \
    && pacman -Scc --noconfirm

RUN useradd -m -G wheel aur-builder && \
    echo "aur-builder ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

USER aur-builder
WORKDIR /home/aur-builder
RUN git clone "${GANESHA_AUR_URL}" 
WORKDIR /home/aur-builder/nfs-ganesha

RUN _custom_flags="-DUSE_DBUS=OFF \
    -DUSE_ADMIN_TOOLS=OFF \
    -DUSE_9P=OFF \
    -DUSE_MONITORING=OFF \
    -DUSE_GSS=OFF \
    -DUSE_NFSIDMAP=OFF \
    -DENABLE_ERROR_INJECTION=OFF \
    -DUSE_FSAL_VFS=ON \
    -DUSE_FSAL_PROXY_V4=OFF \
    -DUSE_FSAL_PROXY_V3=OFF \
    -DUSE_FSAL_GPFS=OFF \
    -DUSE_FSAL_XFS=OFF \
    -DUSE_FSAL_NULL=OFF \
    -DUSE_FSAL_MEM=OFF \
    -DUSE_FSAL_SAUNAFS=OFF" && \
    for _flag in $_custom_flags; do \
    sed -i "/local _flags=(/a \    ${_flag}" PKGBUILD; \
    done && \
    makepkg -si --noconfirm --skipchecksums --skippgpcheck
# hadolint ignore=DL3002
USER root
RUN gpasswd -d aur-builder wheel && \
    userdel -r aur-builder 2>/dev/null || true

# ==============================================================================
# STAGE 2: Custom Distroless Root Filesystem Setup
# ==============================================================================
FROM system-builder AS distroless-setup
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG CACHE_STAGE2=1 

RUN mkdir -p /distroless \
    /distroless/etc \
    /distroless/etc/ca-certificates \
    /distroless/etc/ca-certificates/extracted \
    /distroless/etc/ganesha \
    /distroless/etc/samba \
    /distroless/etc/samba/private \
    /distroless/etc/ssl \
    /distroless/etc/ssl/certs \
    /distroless/run/ \
    /distroless/run/dbus \
    /distroless/run/ganesha \
    /distroless/run/rpcbind \
    /distroless/run/samba \
    /distroless/run/samba/ncalrpc \
    /distroless/run/samba/ncalrpc/np \
    /distroless/run/sendsigs.omit.d \
    /distroless/srv \
    /distroless/tmp \
    /distroless/usr \
    /distroless/usr/bin \
    /distroless/var/cache \
    /distroless/var/cache/samba \
    /distroless/usr/lib \
    /distroless/usr/lib/gorogs \
    /distroless/usr/lib/ganesha \
    /distroless/usr/lib/libnfsidmap \
    /distroless/usr/lib/samba \
    /distroless/var/lib/nfs \
    /distroless/var/lib/nfs/sm \
    /distroless/var/lib/nfs/sm.bak \
    /distroless/var/lib/nfs/statd \
    /distroless/var/lib/nfs/statd/sm \
    /distroless/var/lib/nfs/statd/sm.bak \
    /distroless/var/lib/nfs/ganesha \
    /distroless/var/lib/samba \
    /distroless/var/lib/samba/private \
    /distroless/usr/lib/samba/vfs \
    /distroless/var/lock/samba \
    /distroless/var/log/samba \
    /distroless/var/run \
    /distroless/var/run/ganesha \
    touch /distroless/var/lib/nfs/rmtab && \
    chmod 1777 /distroless/tmp && \
    chmod 0755 /distroless/srv && \
    chmod 0755 /distroless/var/lock/samba && \
    chmod 0755 /distroless/var/cache/samba && \
    chmod 0755 /distroless/var/log/samba && \
    chmod 0755 /distroless/run/samba && \
    chmod 0700 /distroless/run/samba/ncalrpc && \
    chmod 0700 /distroless/run/samba/ncalrpc/np && \
    chmod 0775 /distroless/run/rpcbind && \
    chmod 0755 /distroless/run/sendsigs.omit.d && \
    chmod 0755 /distroless/var/run/ganesha && \
    chmod 0644 /distroless/var/lib/nfs/rmtab && \
    chmod 0755 /distroless/var/lib/nfs/sm && \
    chmod 0755 /distroless/var/lib/nfs/sm.bak && \
    ln -s usr/bin /distroless/bin && \
    ln -s usr/bin /distroless/sbin && \
    ln -s bin /distroless/usr/sbin && \
    ln -s usr/lib /distroless/lib && \
    ln -s usr/lib /distroless/lib64 && \
    grep -E '^root:|^nobody:|^rpc:' /etc/passwd > /distroless/etc/passwd && \
    grep -E '^root:|^nobody:|^rpc:|^wheel:' /etc/group > /distroless/etc/group && \
    cp -a /etc/protocols /etc/services /etc/netconfig /distroless/etc/ && \
    if [ -f /etc/idmapd.conf ]; then cp -a /etc/idmapd.conf /distroless/etc/; fi && \
    echo "hosts: files dns" > /distroless/etc/nsswitch.conf && \
    cp -a /etc/ssl/certs/ca-certificates.crt /distroless/etc/ssl/certs/ && \
    cp -a /etc/ca-certificates/extracted/* /distroless/etc/ca-certificates/extracted/ && \
    ln -sf /proc/self/mounts /distroless/etc/mtab && \
    # Programs
    cp /usr/bin/ganesha.nfsd /distroless/usr/bin/ && \
    cp /usr/bin/smbd /distroless/usr/bin/ && \
    cp /usr/bin/net /distroless/usr/bin/ && \
    cp /usr/bin/tdbtool /distroless/usr/bin/ && \
    cp /usr/bin/dbwrap_tool /distroless/usr/bin/ && \
    cp /usr/bin/smbpasswd /distroless/usr/bin/ && \
    cp /usr/lib/samba/samba/samba-dcerpcd /distroless/usr/bin/ && \
    cp /usr/bin/nmbd /distroless/usr/bin/ && \
    cp /usr/bin/rpcbind /distroless/usr/bin/ && \
    cp /usr/bin/rpc.statd /distroless/usr/bin/ && \
    cp /usr/bin/sm-notify    /distroless/usr/bin/ && \
    cp -a /usr/lib/ganesha/* /distroless/usr/lib/ganesha/ && \
    cp -r /usr/lib/samba/* /distroless/usr/lib/samba/ && \
    cp -vnP /usr/lib/libnss_files* /distroless/usr/lib/ && \
    cp -vnP /usr/lib/libnss_dns* /distroless/usr/lib/ && \
    cp -vnP /usr/lib/libnfsidmap.so* /distroless/usr/lib/ && \
    cp -vnP /usr/lib/libcups.so* /distroless/usr/lib/ && \
    cp -a /usr/lib/libnfsidmap/* /distroless/usr/lib/libnfsidmap/ && \
    cp -vnP /usr/lib/libdl.so* /distroless/usr/lib/ && \
    cp -a /usr/lib/libnfsidmap/* /distroless/usr/lib/libnfsidmap/ && \
    #Free Space Zero Script Samba
    printf '#!/sh\necho "100000 0"\n' > /distroless/usr/bin/smb-fake-dfree && \
    chmod +x /distroless/usr/bin/smb-fake-dfree && \
    #Free Space Zero Trick for Nfs
    printf '#define _GNU_SOURCE\n#include <sys/vfs.h>\n#include <dlfcn.h>\nint statfs(const char *path, struct statfs *buf) {\n    int (*orig_statfs)(const char*, struct statfs*) = dlsym(RTLD_NEXT, "statfs");\n    int ret = orig_statfs(path, buf);\n    if (ret == 0) { buf->f_bfree = 0; buf->f_bavail = 0; }\n    return ret;\n}\nint statfs64(const char *path, struct statfs64 *buf) {\n    int (*orig_statfs64)(const char*, struct statfs64*) = dlsym(RTLD_NEXT, "statfs64");\n    int ret = orig_statfs64(path, buf);\n    if (ret == 0) { buf->f_bfree = 0; buf->f_bavail = 0; }\n    return ret;\n}\n' > /usr/src/fake_statfs.c && \
    gcc -shared -fPIC -o /distroless/usr/lib/libfake_statfs.so /usr/src/fake_statfs.c -ldl && \
    # ==============================================================================
    # DEBUG FILES TO BE REMOVED ONCE EVERYTHING IS
    # ==============================================================================
    if [ "$ENABLE_DEBUG" = "true" ]; then \
    # 1. Essential filesystem checking utilities
    cp /usr/bin/ls       /distroless/usr/bin/ && \
    cp /usr/bin/cat      /distroless/usr/bin/ && \
    cp /usr/bin/mkdir    /distroless/usr/bin/ && \
    # 2. Critical networking socket tracking diagnostics (Highly Recommended)
    cp /usr/bin/ss       /distroless/usr/bin/ && \
    cp /usr/bin/rpcinfo  /distroless/usr/bin/ && \
    # 3. tool for missing librarys
    cp /usr/bin/strace /distroless/usr/bin/; \
    fi  

RUN for bin in /distroless/usr/bin/* \
    /distroless/usr/lib/samba/*.so \
    /distroless/usr/lib/samba/vfs/*.so \
    /distroless/usr/lib/samba/pdb/*.so \
    /distroless/usr/lib/ganesha/*.so \
    /distroless/usr/lib/libnfsidmap/*.so; do \
    [ -f "$bin" ] || continue; \
    ldd "$bin" 2>/dev/null | awk 'match($0, /\/[^ ]+/) {print substr($0, RSTART, RLENGTH)}' | while read -r lib; do \
    cp -vnL "$lib" "/distroless/usr/lib/$(basename "$lib")"; \
    done; \
    done

# ==============================================================================
# STAGE 3: Fast Go Binary Builder & Master Dependency Scanner
# ==============================================================================
FROM system-builder AS go-binary-builder
# hadolint ignore=DL3002
USER root
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG ENABLE_DEBUG
ARG CGO_ENABLED
ARG GOOS
ARG GOARCH

COPY --from=distroless-setup /distroless /distroless

ARG CACHE_STAGE3A=1 
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
ARG CACHE_STAGE3B=1 
COPY . .

RUN TAG_LIST="" && BUILD_ARG="" && \
    if [ "$ENABLE_DEBUG" = "true" ]; then TAG_LIST="${TAG_LIST:+$TAG_LIST,}debug"; fi && \
    if [ -n "$TAG_LIST" ]; then BUILD_ARG="-tags=${TAG_LIST}"; fi && \
    CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} \
    go build -a -v ${BUILD_ARG:+"$BUILD_ARG"} -ldflags="-s -w" -o /gorogs main.go

RUN cp /gorogs /distroless/usr/bin/gorogs && chmod +x /distroless/usr/bin/gorogs

RUN find plugins -mindepth 2 -maxdepth 2 -type d | while read -r plugin_dir; do \
    _type=$(basename "$(dirname "$plugin_dir")") && \
    _name=$(basename "$plugin_dir") && \
    _output_name="gorogs-${_type}-${_name}.so" && \
    \
    echo "Compiling system appliance: ${_output_name} -> ${plugin_dir}" && \
    CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} \
    go build -buildmode=plugin \
    -ldflags="-s -w" \
    -o "/distroless/usr/lib/gorogs/${_output_name}" \
    "${plugin_dir}"/*.go; \
    done

RUN if [ "$CGO_ENABLED" = "1" ] ; then \
    for target in /distroless/usr/bin/gorogs /distroless/usr/lib/gorogs/*.so; do \
    [ -f "$target" ] || continue; \
    ldd "$target" 2>/dev/null | awk 'match($0, /\/[^ ]+/) {print substr($0, RSTART, RLENGTH)}' | while read -r lib; do \
    cp -vnL "$lib" "/distroless/usr/lib/$(basename "$lib")"; \
    done; \
    done; \
    fi


# ==============================================================================
# STAGE 4: Final Distroless Runtime Assembly
# ==============================================================================
FROM scratch AS final

WORKDIR /
COPY --from=go-binary-builder /distroless/. /

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/bin/gorogs", "--check-health"]

EXPOSE \
    135/tcp \
    137/udp \
    138/udp \
    139/tcp \
    445/tcp \
    3702/udp \
    5357/tcp \
    111/tcp 111/udp \
    2049/tcp 2049/udp \
    20048/tcp 20048/udp \
    32803/tcp 32803/udp \
    875/tcp 875/udp

ENTRYPOINT ["/usr/bin/gorogs"]
