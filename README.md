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

## ⚙️ Environment Variables Reference

Here is a comprehensive breakdown of all variables supported by the orchestration engine. These can be configured in your `docker-compose.yml` file to tune storage behavior, network interfaces, and backend daemon operations.

| Variable Name | Type | Default Value | Allowed Options / Format | Description |
| :--- | :--- | :--- | :--- | :--- |
| **`DEBUG_LOG`** | List String | *None* | Comma-separated components (e.g., `avahi,samba`) | Enables verbose debugging output for specified subsystems or components. |
| **`gorogs.disabled`** | List String | *None* | Comma-separated components (e.g., `nfs,wsdd`) | Disables global orchestrator sub-features or plugins from booting entirely. |
| **`gorogs.enabled`** | List String | *None* | Comma-separated components | Forces specified orchestrator plugins or features to run overriding structural configurations. |
| **`gorogs.shareroot`** | String | `/srv` | Valid container path directory string | Defines the localized internal directory endpoint scanned at boot to provision storage mount points. |
| **`gorogs.Workgroup`** | String | `WORKGROUP` | Standard NetBIOS alphanumeric string | Declares the global SMB cluster domain group context processed by file sharing and network beacons. |
| **`gorogs.networkinterfaces`** | List String | `eth0` | Comma-separated tracking interfaces (e.g., `eth0,eth1`) | Binds server sockets and multicast broadcast interfaces directly to specified active adapters. |
| **`gorogs.avahi.disabled`** | List String | *None* | Comma-separated parameters | Excludes specific network service maps (e.g., `nfs`, `samba`) inside the Avahi mDNS loop. |
| **`gorogs.nfs.zerofreespace`** | Boolean | `true` | `true`, `false` | Injects a virtual wrapper over NFS Ganesha to explicitly report standard remaining export spaces as empty. |
| **`gorogs.samba.mode`** | String | `Registry` | `Registry`, `File`, `Mixed` | Configures Samba's active schema layout model (`Registry`-based config, static flat `.conf` files, or both combined). |
| **`gorogs.samba.basediroverlay`** | Boolean | `true` | `true`, `false` | Controls folder virtualization rules to ensure dynamic storage mounts map accurately onto the scratch overlay. |
| **`gorogs.samba.batchinjection`** | Boolean | `true` | `true`, `false` | Optimizes bulk runtime configuration processing, enabling accelerated share population during large array indexing cycles. |
| **`gorogs.samba.zerofreespace`** | Boolean | `true` | `true`, `false` | Intercepts disk calculation frames to force Samba shares to explicitly report zero bytes of remaining write space. |
| **`gorogs.samba.vetofiles`** | String | `/*.~tmp/` | Standard Samba veto string formatting | Blocks specific files or extension suffixes from being listed or read by clients (e.g., locking out broken caching metadata loops). |
| **`gorogs.samba.defaultcomment`** | String | *None* | Any descriptive string string descriptor | Sets the baseline fallback user-facing description label applied dynamically to each shared directory resource. |
| **`gorogs.samba.servercomment`** | String | `"GO Read Only Guest Share"` | Any clean alphanumeric text text string | Declares the global machine identifier description banner shown directly inside network browsers. |

## 📂 Automated Share Provisioning (`/srv` Scanning)

The container implements a hands-off file provisioning engine designed to automate multi-directory deployments. Rather than requiring users to manually configure Samba sections or NFS paths for every directory, `gorogs` treats the mapped `/srv` bind volume as an absolute storage matrix namespace root.

### How It Works Under the Hood

1. **Boot Scan:** At container startup, the Go orchestrator uses native kernel system utilities to perform an isolated look-ahead sweep of the mapped `/srv` filesystem tree boundary.
2. **Directory Detection:** The engine identifies every top-level child folder residing within that boundary.
3. **Dynamic Configuration Injection:**
   * **Samba:** Each directory name is instantly tokenized and registered into Samba's live configuration backend as an independent, unauthenticated public guest allocation share footprint.
   * **NFS-Ganesha:** The path maps are appended into the internal memory block export tree of the NFS daemon dynamically.

### Concrete Mapping Example

If your host storage architecture contains separate media structures organized like this:

```text
/mnt/storage/
├── movies/
├── tv-shows/
└── documentaries/
```

When you mount the root layer `/mnt/storage` directly onto the container's volume target:

```yaml
volumes:
  - /mnt/storage:/srv:ro
```

The orchestrator will scan the paths on boot and automatically declare three distinct, isolated network resource shares across the LAN:

| Network Protocol | Target Mounting Path / Resource Address | Access Permissions |
| :--- | :--- | :--- |
| **Samba / SMB** | `smb://[shareName]/movies` `smb://[shareName]/tv-shows` `smb://[shareName]/documentaries` | **Read-Only / Guest** |
| **Network File System (NFS)** | `nfs://[shareName]/movies` `nfs://[shareName]/tv-shows` `nfs://[shareName]/documentaries` | **Read-Only / Guest** |

This architecture scales seamlessly. You can drop new sub-folders into your storage array pool at any time; the container instance will make the new shares instantly discoverable without modifying a single line of code.

It is possible to point this to the root of your movies/tvshows folder and it will treat each movie/tvshow as a share so you can mount the full "server" as if it's a drive of folders.

## 📊 Health Monitor Levels (`HEALTH_MODE`)

The internal health check subsystem mounts an isolated server on `127.0.0.1:8080/healthz`. Every 30 seconds, Docker triggers an internal client probe (`gorogs --check-health`) that queries this server. The server tracks component health states via a type-safe enum strategy matrix:

* **`default`**: Scans only storage shares (`Samba`, `NFS`) that handle active file operations. It ignores discovery beacons, so a minor advertisement glitch won't crash the container.
* **`full`**: Absolutely strict validation level. Evaluates every single active daemon and network beacon. If anything returns an error, the container becomes `unhealthy`.
* **`critical`**: Evaluates all daemons and beacons that explicitly declare themselves as critical inside the Go code structures
* **`shares`**: Validates all active storage share loops while completely bypassing the network beacon mapping pools.
* **`nfs`**: Runs a micro-targeted verification loop strictly tracking the `rpcbind` and `ganesha.nfsd` threads.
* **`samba`**: Runs a micro-targeted verification loop strictly tracking the background `smbd` daemon state.
* **`disabled`**: Bypasses all monitoring tests completely and immediately returns an automatic `200 OK` response to the Docker engine.
