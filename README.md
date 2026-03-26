# Frontline VPS Monitor

一个面向 3 台 VPS 的分布式监控探针服务：

- 3 节点互探
- `hashicorp/raft` 自动选主
- SQLite 本地物化视图
- Cloudflare Free 固定域名接入
- 同仓前后端分离仪表盘

## 主要能力

- 每台节点都采集本机 CPU、内存、磁盘、load、uptime
- 自动探测并展示硬件配置（CPU 型号/核心数、总内存、总磁盘、OS、内核版本）
- 每台节点都探测另外两台的 SSH、443、额外 TCP 端口和 HTTP 健康检查
- leader 基于心跳和互探结果计算 `healthy / degraded / critical / unknown`
- active incident、事件流、入口 DNS 状态会复制到所有节点
- 页面可以从任意健康节点访问

## 项目结构

- `cmd/vps-monitor`: 启动入口
- `frontend`: 独立前端入口（同仓 SPA）
- `internal/config`: 配置加载和校验
- `internal/cluster`: Raft、FSM 和 leader 提交
- `internal/monitor`: 采集器与互探器
- `internal/engine`: 状态判定、incident、告警和 DNS 控制
- `internal/web`: API、静态前端分发和内部写入接口

## 本地开发

1. 复制 `monitor.example.yaml` 为 `monitor.yaml`
2. 按你的节点信息修改配置
3. 安装依赖并构建：

```bash
go mod tidy
go build ./cmd/vps-monitor
```

4. 启动：

```bash
./vps-monitor -config monitor.yaml
```

## 本机快速测试

Windows 或单机 smoke test 可以直接用仓库里的本地配置：

```bash
go build -o vps-monitor.exe ./cmd/vps-monitor
vps-monitor.exe -config monitor.local.yaml
```

然后打开：

- `http://127.0.0.1:8443/`
- `http://127.0.0.1:8443/healthz`
- `http://127.0.0.1:8443/api/v1/cluster`

`monitor.local.yaml` 关闭了 Cloudflare、告警、systemd 和 Docker 检查，并启用单节点本地模式，适合先验证前端页面、API、Raft 自举和本机采集是否正常。

## 前端结构

- 浏览器访问 `/`、`/events`、`/nodes/{nodeID}` 时，后端统一返回 `frontend/index.html`
- 前端自己通过 `/api/v1/*` 拉取数据并渲染页面
- `GET /api/v1/meta` 提供前端所需的轻量运行时元信息，例如测试告警渠道与 token 要求

## 测试告警

事件页现在带了一个“发送测试告警”面板。

- 本机访问时，如果没有设置 `MONITOR_TEST_ALERT_TOKEN`，可以直接触发测试
- 如果要远程触发，先设置环境变量 `MONITOR_TEST_ALERT_TOKEN`
- 设置后，事件页会要求输入 token，后端也接受 `Authorization: Bearer <token>`
- 测试消息不会创建真实 incident，只会直接调用已启用的通知渠道

## 生产部署

### 1. 准备配置

以 [monitor.example.yaml](monitor.example.yaml) 为模板，为每台 VPS 准备一份 `/etc/vps-monitor/monitor.yaml`。

必须按真实环境修改：

- `cluster.node_id`
- `cluster.raft_addr`
- `network.public_ipv4`
- `cluster.peers`
- `checks.services`
- `checks.docker_checks`
- `cloudflare.*`
- `alerts.*`

说明：

- `cluster.peers` 需要在三台机器上保持一致，并且必须包含三台节点的 `node_id / api_addr / raft_addr / public_ipv4`
- `cluster.raft_bind_addr` 是可选字段；未设置时默认回退到 `cluster.raft_addr`。如果节点在 NAT 后面，或者想让 Raft 监听 `0.0.0.0:7000` 但对外广播另一地址，就把 bind 写在这里
- `cluster.peers[].display_name` 是可选字段，只影响页面和 API 展示名称，不影响 `node_id`、Raft 身份或路由
- `cluster.peers[].ingress_candidate` 是可选字段，默认 `true`。设成 `false` 后，该节点仍然参与 Raft、可以当 leader、也继续做监控，只是不再参与 ingress 入口节点选择
- 如果某台机器的 `443` 已经被 Xray/VLESS+Reality 等服务占用，可以把它的 `ingress_candidate` 设为 `false`，这样它还能留在集群里，但不会被选成 `monitor.example.com` 的 HTTPS 入口
- 管理后台可以在运行时覆盖节点显示名称；覆盖值优先于 `monitor.yaml` 里的 `display_name`
- 如果某台机器没有 `nginx` 或 `docker`，把它从 `checks.services` / `checks.docker_checks` 里删掉，否则会长期显示 `degraded`
- 如果暂时不接 Cloudflare，把 `cloudflare.enabled` 设为 `false`

### 2. 构建 Linux 二进制

在开发机上构建：

```bash
GOOS=linux GOARCH=amd64 go build -o vps-monitor ./cmd/vps-monitor
```

如果目标 VPS 是 ARM64：

```bash
GOOS=linux GOARCH=arm64 go build -o vps-monitor ./cmd/vps-monitor
```

Windows `cmd` 下可写成：

```cmd
set GOOS=linux
set GOARCH=amd64
go build -o vps-monitor ./cmd/vps-monitor
```

### 3. 上传文件到每台 VPS

每台机器至少需要这些文件：

- `/opt/vps-monitor/vps-monitor`
- `/etc/vps-monitor/monitor.yaml`
- `/etc/systemd/system/vps-monitor.service`

先创建目录：

```bash
mkdir -p /opt/vps-monitor
mkdir -p /etc/vps-monitor
mkdir -p /var/lib/vps-monitor
```

然后给二进制执行权限：

```bash
chmod +x /opt/vps-monitor/vps-monitor
```

### 4. 环境变量

如果启用了 Cloudflare、Telegram、SMTP 或测试告警 token，把它们写到：

```bash
/etc/vps-monitor/env
```

例如：

```bash
CLOUDFLARE_API_TOKEN=xxxx
TELEGRAM_BOT_TOKEN=xxxx
SMTP_PASSWORD=xxxx
MONITOR_TEST_ALERT_TOKEN=xxxx
MONITOR_INTERNAL_TOKEN=xxxx
```

建议：

- `chmod 600 /etc/vps-monitor/env`
- 不要把这些值直接写进 `monitor.yaml`

### 5. TLS 与端口

如果你让程序自己直接提供 HTTPS：

- 把证书放到 `/etc/ssl/cloudflare-origin.pem` 和 `/etc/ssl/cloudflare-origin.key`
- 在配置里设置：

```yaml
network:
  listen_addr: :443
  public_https_port: 443
  tls_cert_file: /etc/ssl/cloudflare-origin.pem
  tls_key_file: /etc/ssl/cloudflare-origin.key
```

如果你准备用 Nginx / Caddy 反代：

- 程序保持监听 `:8443`
- 反代对外监听 `443`
- `public_https_port` 仍应写对外真实访问端口

### 6. systemd 安装

把仓库里的 [vps-monitor.service](deploy/vps-monitor.service) 放到：

```bash
/etc/systemd/system/vps-monitor.service
```

然后执行：

```bash
systemctl daemon-reload
systemctl enable vps-monitor
```

### 7. 启动顺序

Raft 默认使用 `cluster.peers` 里的第一台节点做初始 bootstrap，所以建议按顺序启动：

1. `node-a`
2. `node-b`
3. `node-c`

每台启动命令：

```bash
systemctl start vps-monitor
```

### 8. 防火墙

至少放通这些端口：

- `7000/tcp`：Raft 节点通信
- `8443/tcp` 或 `443/tcp`：页面和 API
- `22/tcp`：SSH 探测
- `checks.tcp_ports` 里配置的业务端口

### 9. 验证

先看服务状态：

```bash
systemctl status vps-monitor
journalctl -u vps-monitor -n 100 --no-pager
```

再访问：

- `/healthz`
- `/api/v1/cluster`
- `/events`

如果启用了告警渠道，可以直接在事件页用“发送测试告警”面板验证 Telegram / SMTP / webhook。

### 10. Cloudflare 固定域名

如果启用 Cloudflare Free 固定域名：

- 三台机器都部署同一张 Cloudflare Origin CA 证书
- Cloudflare 上创建 `monitor.example.com`
- 开启橙云代理
- 在配置中填好 `zone_id`、`dns_record_id`、`api_token_env`
- leader 会在入口节点变化时通过 DNS API 更新回源 IP

## 说明

- Raft bootstrap 逻辑默认使用 `cluster.peers` 数组里的第一台作为初始 bootstrap 节点
- 观察数据通过 leader 复制，因此初次启动的前几轮页面会看到 `awaiting cluster data`
- `checks.services` 使用 `systemctl is-active`
- `checks.docker_checks` 使用 `docker inspect --format {{.State.Status}}`
- 对外写接口没有开放；公网只提供只读页面和只读 API

## 安全

### 内部端点保护

当服务运行在反向代理（如 Nginx）后面时，必须配置内部通信 token，否则 `/internal/v1/*` 端点的 IP 校验会被绕过。

在每台节点的环境变量中设置相同的 token：

```bash
MONITOR_INTERNAL_TOKEN=<你的随机密钥>
```

生成方式：`openssl rand -base64 32`

也可以在配置中自定义环境变量名：

```yaml
cluster:
  internal_token_env: "MY_CUSTOM_TOKEN_ENV"
```

如果不配置 token，服务回退到基于 IP 的访问控制（仅适用于无反向代理的场景）。

## 发布到 GitHub

如果要让我直接帮你发布到 GitHub，你至少需要提供这些信息：

- 目标 GitHub 仓库地址
  - 例如 `https://github.com/<user>/<repo>.git`
  - 或者明确说“帮我新建 `<repo>` 这个仓库”
- 你希望仓库放在个人账号还是组织下
- 认证方式
  - 已登录的 `gh`
  - 或者可用的 GitHub token
  - 或者你自己先在本机配好 `git`/`gh` 凭据
- 是否需要我顺手做第一次提交
  - 提交信息比如 `Initial commit`
- 仓库可见性
  - `public` 还是 `private`

如果你只是想让我推送现有目录，最省事的方式是你先准备好：

1. 一个空仓库
2. 本机已经能正常 `git push`

然后把仓库 URL 发我，我就可以继续做本地 `git init`、提交、绑定 remote、推送。
