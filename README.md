# gorogs (Go Read-Only Guest Share)

`gorogs` is a high-performance, statically orchestrated storage network appliance designed to export directories as **Read-Only, un-authenticated Guest Shares** concurrently over **Samba (SMB)** and **NFS-Ganesha (v6.5)**. 

The entire appliance life-cycle—including pre-flight directory sanitisation, multi-namespace network beacon advertisements, system process supervision, and dynamic signal handling—is managed natively by a lightweight **Go Orchestrator compiled as PID 1**, completely replacing error-prone legacy shell script loops.

---

## 🗺️ Architectural Topology

```text
               +-------------------------------------------------+

               |             LAN PHYSICAL WIRES                  |
               +-------------------------------------------------+
                                      |
                             [ IPvlan L2 Interface ]
                                      |
  +-----------------------------------|-----------------------------------+

  | gorogs Container Namespace        v (192.168.1.11)                    |
  |                                                                       |
  |  +-----------------------------------------------------------------+  |
  |  |                 MASTER GO ORCHESTRATOR (PID 1)                  |  |
  |  +-----------------------------------------------------------------+  |
  |         |                       |                        |            |
  |         v [Exec Forks]          v [Internal HTTP]        v [Sockets]  |
  |  +--------------+       +---------------+       +---------------+     |
  |  |  DAEMONS     |       | HEALTH SERVER |       | BEACONS       |     |
  |  | - smbd       |------>|  (Port 8080)  |<------| - MdnsBeacon  |     |
  |  | - rpcbind    |       +---------------+       | - WsddBeacon  |     |
  |  | - ganesha    |                               +---------------+     |
  |  +--------------+                                                     |
  +-----------------------------------------------------------------------+
```

---

## ⚡ Core Features

* **PID 1 Supervised Lifecycles**: The Go orchestrator manages configuration rewrites natively and supervises child daemon threads asynchronously. 
* **Zero Host Configuration Bloat**: No permanent static configurations are baked into the image. Everything (`smb.conf`, `ganesha.conf`, etc.) is dynamically compiled in memory or `/dev/shm` on boot.
* **Header-Stripping Log Stream Scraper**: Trailing C-binary metadata strings are processed asynchronously using regular expressions to route clean, readable logs directly to standard parent output streams.
* **Granular Traffic-Light Logging**: Integrates an isolated terminal coloring core mapping normal executions (`INFO` - Green), runtime problems (`ERROR` - Yellow), and application failure crashes (`FATAL` - Red) cleanly.

---

## 📡 Advanced Network & Multicast Requirements

Operating this appliance within a virtual container namespace demands a robust understanding of the underlying Linux network card attachments. Because **WS-Discovery (WSDD)** and **mDNS/Zeroconf** rely strictly on local link layer multicasting, standard Docker bridge networks will drop client discovery probes.

### 1. Network Interface Configuration
To allow client media centers (such as Kodi or Dolphin) to discover the share natively, you **must** deploy the container using an **IPvlan L2** network topology or **Host Networking Mode**. 

IPvlan L2 allows the container to bind its own dedicated MAC address and unique LAN IP address (`192.168.1.11`) directly onto your host's physical network interface card (`eth0`), bypassing bridge-layer NAT translation maps.

### 2. Linux Kernel IGMP & Multicast Routing Constraints
When the container initializes, the **`WsddBeacon`** binds an unfiltered UDP socket to `0.0.0.0:3702` and transmits an IGMP membership registration package to join the official Windows discovery multicast group: **`239.255.255.250`**.

For the host kernel to mirror this traffic down to the container's virtual network card, your infrastructure must satisfy these environmental parameters:
* **IGMP Snooping**: Your managed LAN network switches must have IGMP Snooping verified as active, or an IGMP Querier present, to prevent the switch from dropping the container's multi-recipient subscription frames.
* **Privileged Execution Context**: The container requires the **`privileged: true`** execution flag. This bypasses the default Docker network isolation boundaries and allows the Go sockets layer to interact directly with the physical network interface cards.

---

## ⚙️ Environment Variables

The appliance behavior is fully customizable at boot time using targeted environment variables.

| Variable Name | Acceptable Values / Examples | Default Behavior (If Omitted) | Description |
| :--- | :--- | :--- | :--- |
| `HEALTH_MODE` | `default`, `full`, `critical`, `shares`, `nfs`, `samba`, `disabled` | `default` | Adjusts the active internal HTTP health scanning strategy. |
| `DEBUG_LOG` | `all`, `nfs`, `samba`, `rpcbind`, `wsdd`, `mdns`, `health` | *Muted* | A comma-separated list of subsystems authorized to output verbose Cyan `[DEBUG]` telemetry. |
| `DISABLE_NFS` | *Any value (e.g., `true`)* | Enabled | Completely deactivates the NFS-Ganesha storage daemon and its dependent services. |
| `DISABLE_SAMBA` | *Any value (e.g., `true`)* | Enabled | Completely deactivates the Samba (`smbd`) storage daemon. |
| `DISABLE_RPCBIND` | *Any value (e.g., `true`)* | Enabled | **Dynamic Loopback Lock**: Restricts `rpcbind` strictly to `127.0.0.1` inside the container to prevent outside network exposure while keeping NFS functional. |
| `DISABLE_WSDD` | *Any value (e.g., `true`)* | Enabled | Deactivates the WS-Discovery background loopback beacon completely. |
| `DISABLE_ZEROCONF` | *Any value (e.g., `true`)* | Enabled | Deactivates the unified grandcat mDNS service advertisement socket. |

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

```yaml
version: "3.8"

services:
  gorogs:
    image: local/gorogs:latest
    container_name: gorogs
    privileged: true
    restart: unless-stopped
    # IPvlan L2 setup required to bridge multicast wires cleanly
    network_mode: host 
    environment:
      # Tailored to prioritize critical filesystem operations
      HEALTH_MODE: "default" 
      # Enables deep packet/configuration dumps for target systems
      DEBUG_LOG: "nfs,wsdd" 
    volumes:
      # Map your host media drive array directly to the strict internal share root
      - /mnt/storage/media:/srv:ro 
```

---

## 🧼 Graceful Termination Handling

When the container receives a shutdown signal (`SIGTERM` or `SIGINT`) from the Docker engine, the Go orchestrator captures the event cleanly via `os/signal` traps and enforces a strict, dependency-ordered teardown sequence:

1. **Beacons Disarmed First**: The `WsddBeacon` instantly transmits an autonomous **`Bye` announcement** envelope over the multicast link to inform nearby Kodi media centers that the host is disconnecting, gracefully clearing stale targets from client menus. The `MdnsBeacon` then severs its active UDP registration sockets.
2. **File Daemons Reaped Last**: The orchestrator sends a `SIGTERM` signal down to `smbd` and `ganesha.nfsd`, flushing active file-handle caches to disk and tearing down the storage pipelines cleanly to prevent memory leaks or file system corruptions.
