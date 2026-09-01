# gorogs (Go Read-Only Guest Share)

`gorogs` is a high-performance, statically orchestrated storage network appliance designed to export directories as **Read-Only, un-authenticated Guest Shares** concurrently over **Samba ** and **NFS-Ganesha**.

The entire appliance life-cycle—including pre-flight directory sanitisation, multi-namespace network beacon advertisements, system process supervision, and dynamic signal handling—is managed natively by a lightweight **Go Orchestrator compiled as PID 1**, completely replacing error-prone legacy shell script loops.

---

## ⚡ Core Features
* **PID 1 Supervised Lifecycles**: The Go orchestrator manages configuration rewrites natively and supervises child daemon threads asynchronously.
---
## 📡 Advanced Network & Multicast Requirements
**YOU MUST** deploy the container using either **Macvlan** network topology or **Host Networking Mode**.
---
## ⚙️ Environment Variables
TODO ADD IN THIS SECTION WITH ALL THE VARIABLES
---
## 📊 Health Monitor Levels (`HEALTH_MODE`)
The internal health check subsystem mounts an isolated server on `127.0.0.1:8080/healthz`. Every 30 seconds, Docker triggers an internal client probe (`gorogs --check-health`) that queries this server. The server tracks component health states via a type-safe enum strategy matrix:

* **`default`**: Scans only storage shares (`Samba`, `NFS`) that handle active file operations. It ignores discovery beacons, so a minor advertisement glitch won't crash the container.
* **`full`**: Absolutely strict validation level. Evaluates every single active daemon and network beacon. If anything returns an error, the container becomes `unhealthy`.
* **`critical`**: Evaluates all daemons and beacons that explicitly declare themselves as critical inside the Go code structures (`IsCritical() == true`).
* **`shares`**: Validates all active storage share loops while completely bypassing the network beacon mapping pools.
* **`nfs`**: Runs a micro-targeted verification loop strictly tracking the `rpcbind` and `ganesha.nfsd` threads.
* **`samba`**: Runs a micro-targeted verification loop strictly tracking the background `smbd` daemon state.
* **`disabled`**: Bypasses all monitoring tests completely and immediately returns an automatic `200 OK` response to the Docker engine.
---
## 📦 Container Layout & Storage Paths
* **Internal Share Root**: The container strictly mounts and processes media exports out of **`/srv`**. This path is defined as an immutable constant (`config.ShareRoot`) inside the compiled code.
* **Samba Configuration Hooks**: On boot, the `SambaShare` component scans `/srv`. It ignores protected system labels (`nfs`, `ganesha`) and runs a strict regular expression to check folder names for illegal characters. Valid folders are dynamically mapped as un-authenticated guest exports into **`/dev/shm/smb-shares.conf`**, which is automatically appended to the parent `/etc/samba/smb.conf`.
---
## 🐳 Docker Compose Deployment Blueprint
```
  [name]:
    build: ./gorogs
    image: local/gorogs:latest
    container_name: [name]
    stop_grace_period: 30s # this is only if your system is slow to allow it more time to do what it needs to cleanly shutdown
    stop_signal: SIGINT
    hostname: [name]
    restart: unless-stopped
    privileged: true
    environment:
      TZ: [your timezone]
    volumes:
      - [Share]:/srv:ro # ro is just another layer of read only protection
    networks:
      lan_macvlan:
        ipv4_address: [IP Address]
        mac_address: "[Mac Address]"
```
---
