# Frontline VPS Monitor

一个面向 3 台 VPS 的分布式监控探针服务：

- 多节点互探（默认 full-mesh，可选稀疏观察者模式）
- `hashicorp/raft` 自动选主
- 动态成员目录 + seed auto-join
- SQLite 本地物化视图
- Cloudflare Free 固定域名接入
- 同仓前后端分离仪表盘

## 主要能力

- 每台节点都采集本机 CPU、内存、磁盘、load、uptime
- 自动探测并展示硬件配置（CPU 型号/核心数、总内存、总磁盘、OS、内核版本）
- 每台节点都探测另外两台的 SSH、443、额外 TCP 端口和 HTTP 健康检查
- leader 基于心跳和互探结果计算 `healthy / degraded / critical / unknown`
- active incident、事件流、入口 DNS 状态会复制到所有节点
- 新节点可通过 `join_seeds` 自动加入，旧节点会在运行时自动看到它
- `/admin` 里按分区管理 Alert Channels、Runtime Checks、Node Names、Cluster Membership
- 告警渠道既可继续使用 `monitor.yaml` / 环境变量，也可在后台运行时覆盖并直接发送测试
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

事件页现在带了一个“发送测试告警”面板，管理员后台也新增了按渠道配置和测试的入口。

- 本机访问时，如果没有设置 `MONITOR_TEST_ALERT_TOKEN`，可以直接触发测试
- 如果要远程触发，先设置环境变量 `MONITOR_TEST_ALERT_TOKEN`
- 设置后，事件页会要求输入 token，后端也接受 `Authorization: Bearer <token>`
- 测试消息不会创建真实 incident，只会直接调用已启用的通知渠道
- 管理后台里的渠道测试会使用当前已保存的运行时配置；如果某个渠道没有被后台接管，则自动回退到 `monitor.yaml` / 环境变量

## 生产部署

### 推荐：动态模式 + 单一清单生成各节点配置

现在更推荐直接用动态成员模式部署。这样新增节点时，不需要再给所有老节点改 `cluster.peers`。

1. 复制 [cluster.inventory.example.yaml](cluster.inventory.example.yaml) 为 `cluster.inventory.yaml`
2. 保持 `cluster.mode: dynamic`
3. 只在这一个文件里维护节点列表和共享配置
4. 运行：

```bash
go run ./cmd/vps-monitor-render -inventory cluster.inventory.yaml -out build/configs
```

会生成：

- `build/configs/node-a/monitor.yaml`
- `build/configs/node-b/monitor.yaml`
- `build/configs/node-c/monitor.yaml`

这个工作流的特点：

- 新增节点时，只需要改一份 `cluster.inventory.yaml`
- 旧节点不需要再手工编辑配置内容
- 首个节点会自动生成 `cluster.bootstrap: true`
- 其余节点会自动生成 `cluster.join_seeds`
- 新增节点时，通常只需要下发新节点自己的配置并启动它，旧节点会通过成员目录自动看到它

如果你还想保留旧的静态方式，也可以把 inventory 里的 `cluster.mode` 省略或显式写成 `static`，这样渲染结果仍然会包含完整的 `cluster.peers`。

如果你要新增一个节点，推荐流程是：

1. 在 `cluster.inventory.yaml` 里追加新节点
2. 重新运行 `vps-monitor-render`
3. 把新节点对应的 `monitor.yaml` 下发到新机器
4. 启动新节点，让它通过 `join_seeds` 自动入群

### 1. 准备配置

你可以直接手工维护 [monitor.example.yaml](monitor.example.yaml)，也可以更推荐地使用上面的 `cluster.inventory.yaml` 生成每台 VPS 的 `/etc/vps-monitor/monitor.yaml`。

动态模式推荐至少按真实环境修改：

- `cluster.node_id`
- `cluster.api_addr`
- `cluster.display_name`
- `cluster.raft_addr`
- `network.public_ipv4`
- `cluster.bootstrap` 或 `cluster.join_seeds`
- `cluster.internal_token_env`
- `checks.services`
- `checks.docker_checks`
- `cloudflare.*`
- `alerts.*`

可选性能参数：

- `runtime.loop_interval`
  - 控制 collector、prober、leader engine 三个主循环的间隔
  - 不配置时默认 `15s`
  - 想进一步省资源，可以调成 `30s` 或 `60s`
- `runtime.probe_observers_per_target`
  - 控制每个目标节点由多少个观察节点执行互探
  - `0` 或不配置时保持原来的 full-mesh 互探
  - `>0` 时启用稀疏观察者模式，把探测 fan-out 从 `O(N^2)` 降到 `O(N*K)`
  - 观察节点失去新鲜状态后，会在下一轮 leader 评估中自动被剔除并重分配

动态模式说明：

- `cluster.peers` 为空时即进入动态模式
- 首个节点设置 `cluster.bootstrap: true`
- 其它节点设置 `cluster.join_seeds`
- 动态模式强制要求内部 token；运行前必须在环境变量里提供 `MONITOR_INTERNAL_TOKEN`（或你自定义的 env 名）
- `cluster.role` 可选，默认 `voter`
- `cluster.display_name`、`cluster.priority`、`cluster.ingress_candidate` 会直接进入运行时成员目录
- 管理后台的 “Cluster Membership” 面板可以查看当前角色、leader、健康概况，并执行升为 `voter`、降为 `nonvoter`、移除节点

如果你继续使用静态兼容模式，则还需要维护：

- `cluster.peers`

静态模式说明：

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

动态模式：

1. 先启动 inventory 里第一台节点（渲染结果会带 `cluster.bootstrap: true`）
2. 再启动其它节点；它们会循环请求 `join_seeds`，直到 leader 接受加入
3. 之后如果要加第四台、第五台，直接启动新节点即可，老节点会自动在 `/api/v1/cluster` 和 `/admin` 里看到它

静态模式：

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

- 动态模式下，节点无本地 Raft 状态且配置了 `join_seeds` 时，会在后台自动向 seed 节点发起 join，成功后再进入采集/探测循环
- 动态模式下，运行时节点真相来源是复制到全员的成员目录，而不是本地 `cluster.peers`
- 静态模式仍然可用；这时 bootstrap 逻辑默认使用 `cluster.peers` 数组里的第一台作为初始 bootstrap 节点
- 观察数据通过 leader 复制，因此初次启动的前几轮页面会看到 `awaiting cluster data`
- 稀疏观察者模式下，leader 会按稳定哈希为每个目标节点选择固定数量的观察节点；如果观察证据不足，节点会先进入 `degraded`，不会直接升级成 availability `critical`
- `checks.services` 使用 `systemctl is-active`
- `checks.docker_checks` 使用 `docker inspect --format {{.State.Status}}`
- 对外写接口没有开放；公网只提供只读页面和只读 API

## 安全

### 内部端点保护

动态成员模式要求必须配置内部通信 token；没有 token 时，配置加载会直接失败。

即使在静态模式下，只要服务运行在反向代理（如 Nginx）后面，也强烈建议配置内部通信 token，否则 `/internal/v1/*` 端点的 IP 校验会被绕过。

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

静态模式下如果不配置 token，服务会回退到基于 IP 的访问控制（仅适用于无反向代理的场景）。动态 auto-join 不支持这种回退。
