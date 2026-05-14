<div align="center">
  <img src="webui/public/vite.svg" width="48" alt="Resin Logo" />
  <h1>Resin</h1>
  <p><strong>将大量的代理订阅转化为一个稳定、智能、可观测且支持会话保持的统一代理入口。</strong></p>
</div>

<p align="center">
  <a href="https://github.com/jiujiu532/Resin/releases"><img src="https://img.shields.io/github/v/release/jiujiu532/Resin?style=flat-square&label=release&sort=semver" alt="Release" /></a>
  <a href="https://github.com/jiujiu532/Resin/actions/workflows/release.yml"><img src="https://img.shields.io/github/actions/workflow/status/jiujiu532/Resin/release.yml?style=flat-square&label=build" alt="Build" /></a>
  <a href="https://github.com/jiujiu532/Resin/pkgs/container/resin"><img src="https://img.shields.io/badge/ghcr-ghcr.io%2Fjiujiu532%2Fresin-2496ED?style=flat-square&logo=docker&logoColor=white" alt="GHCR Image" /></a>
  <a href="https://github.com/jiujiu532/Resin/blob/main/LICENSE"><img src="https://img.shields.io/github/license/jiujiu532/Resin?style=flat-square" alt="License" /></a>
</p>

---

**Resin** 是一个专为接管海量节点设计的**高性能智能代理池网关**，在原版基础上新增了**平台级节点黑名单**功能，可以精确屏蔽特定节点参与路由。

## ✨ 相比原版新增功能

**平台级节点黑名单**：在指定平台内拉黑某个节点，该节点不再参与该平台的路由，其他平台不受影响。

- 节点池页面 → 点击节点 → 运维操作 → 选择平台 → 一键拉黑
- 平台详情页 → 运维 Tab → 节点黑名单 → 查看/移除已拉黑节点

---

## 💡 核心特性

- **海量接管**：轻松管理十万级规模的代理节点，原生高并发。
- **智能调度与熔断**：全自动被动+主动健康探测、出口 IP 探测、延迟分析，P2C 算法智能选优。
- **粘性代理**：同一业务账号绑定同一出口 IP，节点故障时自动切换同 IP 备用节点。
- **多种接入方式**：HTTP 正向代理、SOCKS5 正向代理、URL 反向代理。
- **可观测性**：Web 管理后台 + 结构化请求日志，支持按平台、账号、目标站点查询审计。
- **热更新**：配置变更不重启，订阅刷新不断连。
- **状态持久化**：重启后恢复节点健康数据、延迟统计与租约绑定。
- **平台级节点黑名单**（本版新增）：精确屏蔽指定节点，不影响其他平台。

---

## 🚀 部署

### 方式一：Docker Compose（推荐）

**1. 创建 `docker-compose.yml`**

```yaml
services:
  resin:
    image: ghcr.io/jiujiu532/resin:latest
    container_name: resin
    restart: unless-stopped
    environment:
      RESIN_AUTH_VERSION: "V1"
      RESIN_ADMIN_TOKEN: "your-admin-password"   # 管理后台登录密码，自行修改
      RESIN_PROXY_TOKEN: "your-proxy-token"       # 代理认证密码，自行修改
      RESIN_LISTEN_ADDRESS: "0.0.0.0"
      RESIN_PORT: "2260"
    ports:
      - "2260:2260"
    volumes:
      - ./data/cache:/var/cache/resin
      - ./data/state:/var/lib/resin
      - ./data/log:/var/log/resin
```

**2. 启动**

```bash
docker compose up -d
```

**3. 访问管理后台**

浏览器打开 `http://服务器IP:2260`，用 `RESIN_ADMIN_TOKEN` 登录，在「订阅管理」添加节点订阅即可。

---

### 方式二：Docker 单行命令

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

### 方式三：二进制文件直接运行

前往 [Releases](https://github.com/jiujiu532/Resin/releases) 下载对应系统架构的压缩包，解压得到单个 `resin` 二进制文件。

**Linux / macOS：**

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

**Windows（PowerShell）：**

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

### 方式四：源码编译

需要 Go 1.24+ 和 Node.js 22+。

```bash
git clone https://github.com/jiujiu532/Resin.git
cd Resin

# 编译前端
cd webui && npm ci && npm run build && cd ..

# 编译后端
go build \
  -tags "with_quic with_wireguard with_grpc with_utls" \
  -o resin ./cmd/resin

# 运行
RESIN_AUTH_VERSION=V1 \
RESIN_ADMIN_TOKEN=your-admin-password \
RESIN_PROXY_TOKEN=your-proxy-token \
RESIN_PORT=2260 \
./resin
```

---

### 环境变量说明

| 变量 | 必填 | 说明 |
| :--- | :---: | :--- |
| `RESIN_AUTH_VERSION` | ✅ | 认证版本，新部署填 `V1` |
| `RESIN_ADMIN_TOKEN` | ✅ | 管理后台密码，不想要密码填空字符串 `""` |
| `RESIN_PROXY_TOKEN` | ✅ | 代理认证密码，不想要密码填空字符串 `""` |
| `RESIN_PORT` | | 监听端口，默认 `2260` |
| `RESIN_LISTEN_ADDRESS` | | 监听地址，默认 `0.0.0.0` |
| `RESIN_STATE_DIR` | | 配置数据目录，默认 `/var/lib/resin` |
| `RESIN_CACHE_DIR` | | 缓存目录，默认 `/var/cache/resin` |
| `RESIN_LOG_DIR` | | 日志目录，默认 `/var/log/resin` |

---

## 🟢 基础使用

### 接入代理

启动后，按客户端能力选择接入方式：

**HTTP 正向代理：**
```bash
curl -x http://127.0.0.1:2260 -U ":your-proxy-token" https://api.ipify.org
```

**SOCKS5 正向代理（需 V1 模式）：**
```bash
curl --proxy socks5h://127.0.0.1:2260 -U "Default:your-proxy-token" https://api.ipify.org
```

**反向代理：**
```bash
curl http://127.0.0.1:2260/your-proxy-token/./https/api.ipify.org
```

### 筛选节点

在管理后台「平台管理」创建 Platform，配置地区过滤（如只用日本节点填 `jp`）或节点名正则过滤，然后在代理认证中指定平台名：

```bash
# HTTP 正向代理指定平台
curl -x http://127.0.0.1:2260 -U "MyPlatform:your-proxy-token" https://api.ipify.org
```

### 拉黑节点（本版新增）

某个节点解锁效果差？在节点池页面点击该节点，运维操作区选择平台后点「拉黑」，该节点立即从该平台路由中排除，无需重启。

---

## 📖 粘性代理

让同一业务账号始终走同一出口 IP。

**HTTP / SOCKS5 正向代理（V1 格式：`平台.账号:密码`）：**

```bash
curl -x http://127.0.0.1:2260 -U "Default.user_tom:your-proxy-token" https://api.ipify.org
```

**反向代理（推荐生产环境用 Header 传账号）：**

```bash
curl "http://127.0.0.1:2260/your-proxy-token/Default/https/api.example.com/v1/orders" \
  -H "X-Resin-Account: user_tom"
```

---

## 🛠️ 常见问题

**Q: 启动报错 `RESIN_PROXY_TOKEN` 未定义？**
不想设密码也必须显式定义为空：`RESIN_PROXY_TOKEN=""`。

**Q: 启动报错 `RESIN_AUTH_VERSION` 未定义？**
新部署设为 `V1`，有旧数据的参考 [迁移指南](doc/v1.0.0-migration-guide.zh-CN.md)。

**Q: SOCKS5 客户端连不上？**
确认 `RESIN_AUTH_VERSION=V1`，`LEGACY_V0` 模式不启用 SOCKS5。

**Q: WebSocket 反向代理怎么写路径？**
协议字段写 `http` 或 `https`，不要写 `ws`/`wss`，Resin 自动处理升级。

**Q: 节点名正则过滤支持排除语法吗？**
节点名正则字段**不支持** `!` 排除语法（只有地区过滤支持）。排除特定节点请使用平台级节点黑名单功能。

---

## ⚠️ 免责声明

本项目基于 [MIT License](LICENSE) 开源，仅供技术研究与工程实践。使用者须自行确保合法合规，不得用于未授权访问、欺诈、攻击等违法活动。
