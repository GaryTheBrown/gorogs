# gorogs (Go Read-Only Guest Share)

`gorogs` is a Go-orchestrated, scratch-based storage appliance providing unauthenticated read-only guest shares via Samba and NFS-Ganesha.

## Core Features

* **PID 1 Orchestrator:** Manages configuration and supervises daemons (`smbd`, `ganesha.nfsd`, `rpcbind`, `avahi-daemon`, `wsdd2`).
* **Zero-Config Discovery:** Built-in network discovery support for Windows, Apple Finder/Bonjour, and Kodi.
* **Pure C WS-Discovery:** Uses a lightweight `wsdd2` binary.
* **Automated Provisioning:** Scans `/srv` directories at boot to generate shares automatically.

## Requirements & Configuration

* **Network Mode:** Requires Macvlan or Host networking to allow multicast discovery traffic.
* **Key Variables:** Configure options such as `AVAHI_HOSTNAME`, `WORKGROUP`, `SERVERICON`, and `DISABLED`.
* **Health Monitoring:** Configurable via `HEALTH_MODE` endpoints at `127.0.0.1:8080/healthz`.

## Docker Compose Required settings

To ensure the hostname is set correctly set `hostname: [Share Name]` in the service
Because of features needed for the shares `privileged: true` needs to be added to the service too

## ⚙️ Environment Variables

TODO ADD IN THIS SECTION WITH ALL THE VARIABLES

---

## 📊 Health Monitor Levels (`HEALTH_MODE`)

The internal health check subsystem mounts an isolated server on `127.0.0.1:8080/healthz`. Every 30 seconds, Docker triggers an internal client probe (`gorogs --check-health`) that queries this server. The server tracks component health states via a type-safe enum strategy matrix:

* **`default`**: Scans only storage shares (`Samba`, `NFS`) that handle active file operations. It ignores discovery beacons, so a minor advertisement glitch won't crash the container.
* **`full`**: Absolutely strict validation level. Evaluates every single active daemon and network beacon. If anything returns an error, the container becomes `unhealthy`.
* **`critical`**: Evaluates all daemons and beacons that explicitly declare themselves as critical inside the Go code structures
* **`shares`**: Validates all active storage share loops while completely bypassing the network beacon mapping pools.
* **`nfs`**: Runs a micro-targeted verification loop strictly tracking the `rpcbind` and `ganesha.nfsd` threads.
* **`samba`**: Runs a micro-targeted verification loop strictly tracking the background `smbd` daemon state.
* **`disabled`**: Bypasses all monitoring tests completely and immediately returns an automatic `200 OK` response to the Docker engine.

---
