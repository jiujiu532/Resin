<div align="center">
  <img src="webui/public/vite.svg" width="48" alt="Resin Logo" />
  <h1>Resin</h1>
  <p><strong>Turn massive proxy subscriptions into a stable, smart, observable unified proxy entrypoint with sticky sessions.</strong></p>
</div>

<p align="center">
  <a href="https://github.com/jiujiu532/Resin/releases"><img src="https://img.shields.io/github/v/release/jiujiu532/Resin?style=flat-square&label=release&sort=semver" alt="Release" /></a>
  <a href="https://github.com/jiujiu532/Resin/actions/workflows/release.yml"><img src="https://img.shields.io/github/actions/workflow/status/jiujiu532/Resin/release.yml?style=flat-square&label=build" alt="Build" /></a>
  <a href="https://github.com/jiujiu532/Resin/pkgs/container/resin"><img src="https://img.shields.io/badge/ghcr-ghcr.io%2Fjiujiu532%2Fresin-2496ED?style=flat-square&logo=docker&logoColor=white" alt="GHCR Image" /></a>
  <a href="https://github.com/jiujiu532/Resin/blob/main/LICENSE"><img src="https://img.shields.io/github/license/jiujiu532/Resin?style=flat-square" alt="License" /></a>
</p>

[简体中文](README.zh-CN.md) | English

---

**Resin** is a high-performance intelligent proxy pool gateway. This fork adds a **platform-level node blocklist** on top of the original.

## ✨ Added Features

**Platform-level node blocklist**: Block specific nodes from routing within a platform without affecting other platforms.

- Node Pool page → click a node → Ops section → select platform → Block
- Platform detail page → Ops tab → Node Blocklist → view / remove blocked nodes

---

## 💡 Core Features

- **Massive scale**: Handles 100k+ proxy nodes natively.
- **Smart scheduling**: Passive + active health checks, egress IP probing, latency analysis, P2C algorithm.
- **Sticky sessions**: Same account stays on the same egress IP; seamless failover to same-IP backup nodes.
- **Multiple access modes**: HTTP forward proxy, SOCKS5 forward proxy, URL reverse proxy.
- **Observability**: Web UI + structured request logs, queryable by platform, account, and target.
- **Hot reload**: Config changes without restart, subscription refresh without dropping connections.
- **Persistent state**: Node health, latency stats, and lease bindings survive restarts.
- **Node blocklist** (this fork): Precisely exclude nodes per platform.

---

## 🚀 Deployment

### Option 1: Docker Compose (Recommended)

**1. Create `docker-compose.yml`**

```yaml
services:
  resin:
    image: ghcr.io/jiujiu532/resin:latest
    container_name: resin
    restart: unless-stopped
    environment:
      RESIN_AUTH_VERSION: "V1"
      RESIN_ADMIN_TOKEN: "your-admin-password"
      RESIN_PROXY_TOKEN: "your-proxy-token"
      RESIN_LISTEN_ADDRESS: "0.0.0.0"
      RESIN_PORT: "2260"
    ports:
      - "2260:2260"
    volumes:
      - ./data/cache:/var/cache/resin
      - ./data/state:/var/lib/resin
      - ./data/log:/var/log/resin
```

**2. Start**

```bash
docker compose up -d
```

**3. Open the Web UI**

Navigate to `http://YOUR_SERVER_IP:2260` and log in with `RESIN_ADMIN_TOKEN`. Add your proxy subscriptions under **Subscriptions**.

---

### Option 2: Docker Run

```bash
docker run -d \
  --name resin \
  --restart unless-stopped \
  -e RESIN_AUTH_VERSION=V1 \
  -e RESIN_ADMIN_TOKEN=your-admin-password \
  -e RESIN_PROXY_TOKEN=your-proxy-token \
  -e RESIN_LISTEN_ADDRESS=0.0.0.0 \
  -e RESIN_PORT=2260 \
  -p 2260:2260 \
  -v $(pwd)/data/cache:/var/cache/resin \
  -v $(pwd)/data/state:/var/lib/resin \
  -v $(pwd)/data/log:/var/log/resin \
  ghcr.io/jiujiu532/resin:latest
```

---

### Option 3: Prebuilt Binary

Download the archive for your OS/arch from [Releases](https://github.com/jiujiu532/Resin/releases) and extract the `resin` binary.

**Linux / macOS:**

```bash
chmod +x resin

RESIN_AUTH_VERSION=V1 \
RESIN_ADMIN_TOKEN=your-admin-password \
RESIN_PROXY_TOKEN=your-proxy-token \
RESIN_LISTEN_ADDRESS=0.0.0.0 \
RESIN_PORT=2260 \
RESIN_STATE_DIR=./data/state \
RESIN_CACHE_DIR=./data/cache \
RESIN_LOG_DIR=./data/log \
./resin
```

**Windows (PowerShell):**

```powershell
$env:RESIN_AUTH_VERSION="V1"
$env:RESIN_ADMIN_TOKEN="your-admin-password"
$env:RESIN_PROXY_TOKEN="your-proxy-token"
$env:RESIN_LISTEN_ADDRESS="0.0.0.0"
$env:RESIN_PORT="2260"
$env:RESIN_STATE_DIR=".\data\state"
$env:RESIN_CACHE_DIR=".\data\cache"
$env:RESIN_LOG_DIR=".\data\log"
.\resin.exe
```

---

### Option 4: Build from Source

Requires Go 1.24+ and Node.js 22+.

```bash
git clone https://github.com/jiujiu532/Resin.git
cd Resin

# Build WebUI
cd webui && npm ci && npm run build && cd ..

# Build binary
go build \
  -tags "with_quic with_wireguard with_grpc with_utls" \
  -o resin ./cmd/resin

# Run
RESIN_AUTH_VERSION=V1 \
RESIN_ADMIN_TOKEN=your-admin-password \
RESIN_PROXY_TOKEN=your-proxy-token \
RESIN_PORT=2260 \
./resin
```

---

### Environment Variables

| Variable | Required | Description |
| :--- | :---: | :--- |
| `RESIN_AUTH_VERSION` | ✅ | Auth version. Use `V1` for new deployments. |
| `RESIN_ADMIN_TOKEN` | ✅ | Web UI admin password. Set to `""` to disable auth. |
| `RESIN_PROXY_TOKEN` | ✅ | Proxy auth token. Set to `""` to disable auth. |
| `RESIN_PORT` | | Listen port. Default: `2260` |
| `RESIN_LISTEN_ADDRESS` | | Listen address. Default: `0.0.0.0` |
| `RESIN_STATE_DIR` | | State data directory. Default: `/var/lib/resin` |
| `RESIN_CACHE_DIR` | | Cache directory. Default: `/var/cache/resin` |
| `RESIN_LOG_DIR` | | Log directory. Default: `/var/log/resin` |

---

## 🟢 Basic Usage

**HTTP forward proxy:**
```bash
curl -x http://127.0.0.1:2260 -U ":your-proxy-token" https://api.ipify.org
```

**SOCKS5 forward proxy (V1 only):**
```bash
curl --proxy socks5h://127.0.0.1:2260 -U "Default:your-proxy-token" https://api.ipify.org
```

**Reverse proxy:**
```bash
curl http://127.0.0.1:2260/your-proxy-token/./https/api.ipify.org
```

### Filter nodes by platform

Create a Platform in the Web UI with region or regex filters, then specify it in the proxy auth:

```bash
curl -x http://127.0.0.1:2260 -U "MyPlatform:your-proxy-token" https://api.ipify.org
```

### Block a node (this fork)

Open a node in the Node Pool page, pick a platform in the Ops section, and click **Block**. The node is immediately excluded from that platform's routing.

---

## 📖 Sticky Sessions

Keep the same account on the same egress IP.

**HTTP / SOCKS5 (V1 format: `Platform.Account:token`):**

```bash
curl -x http://127.0.0.1:2260 -U "Default.user_tom:your-proxy-token" https://api.ipify.org
```

**Reverse proxy with `X-Resin-Account` header (recommended for production):**

```bash
curl "http://127.0.0.1:2260/your-proxy-token/Default/https/api.example.com/v1/orders" \
  -H "X-Resin-Account: user_tom"
```

---

## 🛠️ FAQ

**Q: Startup fails with `RESIN_PROXY_TOKEN` undefined?**
Even if you don't want a password, you must define it explicitly: `RESIN_PROXY_TOKEN=""`.

**Q: Startup fails with `RESIN_AUTH_VERSION` undefined?**
Set it to `V1` for new deployments. See the [migration guide](doc/v1.0.0-migration-guide.md) if upgrading.

**Q: SOCKS5 client can't connect?**
Confirm `RESIN_AUTH_VERSION=V1`. SOCKS5 is disabled in `LEGACY_V0` mode.

**Q: How to use WebSocket with reverse proxy?**
Use `http` or `https` in the path — not `ws`/`wss`. Resin handles the upgrade automatically.

**Q: Does the node name regex filter support exclusion (`!`)?**
No. Only region filters support `!` negation. Use the platform-level node blocklist to exclude specific nodes.

---

## ⚠️ Disclaimer

Licensed under [MIT](LICENSE). For technical research and engineering use only. Users are responsible for ensuring lawful and compliant usage.
