# XrayR 运行流程与 Docker Compose 部署

## 一、程序在做什么

XrayR 不是面板，也不保存用户数据库。它是 SSPanel-UIM 和 xray-core 之间的后端：

1. 从 `config.yml` 读取面板地址、密钥、节点 ID 和本地控制参数。
2. 请求 SSPanel-UIM 的 `/mod_mu` API，取得节点配置、用户列表和审计规则。
3. 把面板数据转换为 xray-core 的入站、出站和用户配置。
4. xray-core 接收客户端连接，并统计每个用户的流量与在线 IP。
5. 按 `UpdatePeriodic` 周期重新同步节点/用户，并向面板上报流量、在线 IP、系统状态和审计命中记录。
6. `config.yml` 发生变化时，程序关闭旧实例并在进程内重新启动。

当前源码只保留 SSPanel-UIM 适配器。其它面板适配器及其专属依赖已经删除，
配置中的 `PanelType` 必须是 `SSpanel`。

主要调用链：

```text
main.go
  -> cmd.Execute()
  -> cmd/root.go: run()
  -> panel.New() / Panel.Start()
  -> api/sspanel.New()
  -> controller.New() / Controller.Start()
  -> InboundBuilder + OutboundBuilder + 用户构建
  -> nodeInfoMonitor + userInfoMonitor 周期任务
```

## 二、SSPanel-UIM 接口

启动和周期同步会使用这些接口：

| 用途 | 请求 |
| --- | --- |
| 节点配置 | `GET /mod_mu/nodes/{NodeID}/info` |
| 用户列表 | `GET /mod_mu/users?node_id={NodeID}` |
| 上报流量 | `POST /mod_mu/users/traffic?node_id={NodeID}` |
| 上报在线 IP | `POST /mod_mu/users/aliveip?node_id={NodeID}` |
| 获取审计规则 | `GET /mod_mu/func/detect_rules` |
| 上报审计命中 | `POST /mod_mu/users/detectlog?node_id={NodeID}` |

每次请求都会带 `key` 和 `muKey`，值均来自 `ApiKey`。面板返回 ETag 时，后续同步会带 `If-None-Match`，收到 HTTP 304 就沿用旧配置。

## 三、使用服务器现有 `/etc/XrayR` 部署

要求：Linux 服务器、Docker Engine、Docker Compose v2，以及可以访问面板 API。

本仓库的 Compose 会把宿主机 `/etc/XrayR` 挂载到容器内同一路径。因此原一键
安装脚本留下的 `config.yml`、geo 数据、规则文件和证书可以继续使用，不需要复制
到源码目录。

先确认旧配置只使用 SSPanel：

```bash
cd /opt/XrayR
grep -n "PanelType\\|ApiHost\\|NodeID\\|NodeType" /etc/XrayR/config.yml
sudo test -r /etc/XrayR/config.yml
sudo cp -a /etc/XrayR "/etc/XrayR.backup-$(date +%Y%m%d-%H%M%S)"
```

`PanelType` 必须是 `SSpanel`。配置中的文件路径应使用 `/etc/XrayR/...`。如果证书
实际放在 `/root`、`/usr/local` 等其它目录，需要移入 `/etc/XrayR/cert` 并修改
`CertFile`、`KeyFile`，或者额外增加对应的卷挂载。

先构建镜像，此时不要停止旧服务：

```bash
cd /opt/XrayR
docker compose build
docker run --rm xrayr-local:latest version
```

确认构建成功后，查找并停止一键脚本安装的旧服务：

```bash
systemctl list-unit-files | grep -i xrayr
systemctl status XrayR --no-pager
sudo systemctl disable --now XrayR
```

如果服务名是小写，则把命令中的 `XrayR` 改成 `xrayr`。然后启动容器：

```bash
cd /opt/XrayR
docker compose up -d
docker compose ps
docker compose logs -f --tail=200
```

看到成功拉取节点、添加用户并启动周期任务后，再测试客户端连接。检查宿主机实际
监听的端口：

```bash
ss -lntup
```

Compose 使用 `network_mode: host`，因为监听端口由面板动态下发。该模式仅建议在 Linux 服务器使用，也意味着容器监听的端口就是宿主机端口，防火墙必须放行节点的 TCP/UDP 端口。

## 四、证书模式

- `CertMode: none`：不使用 TLS，或使用 REALITY。
- `CertMode: file`：把证书和私钥放入 `/etc/XrayR/cert/`，配置路径 `/etc/XrayR/cert/...`。
- `CertMode: http`：自动申请，域名的 80 端口必须可从公网访问。
- `CertMode: tls`：自动申请，域名的 443 端口必须可从公网访问。
- `CertMode: dns`：通过 DNS 服务商 API 申请，需要正确填写 `Provider` 和 `DNSEnv`。

整个 `/etc/XrayR` 通过卷保存在宿主机，重建容器不会丢失证书。

## 五、更新与回滚

更新源码后：

```bash
docker compose build --pull
docker compose up -d
docker compose logs -f --tail=200
```

上线前建议备份当前源码目录和 `/etc/XrayR`。原仓库不可用时，不要依赖远程
`latest` 镜像；应保留自己的源码和已经验证的本地镜像。

需要回滚到一键脚本安装的旧程序时：

```bash
cd /opt/XrayR
docker compose down
sudo systemctl enable --now XrayR
```

本部署流程不会修改 `/etc/XrayR/config.yml`，一般不需要恢复配置。只有手动修改过
配置时，才从 `/etc/XrayR.backup-日期时间` 恢复。

## 六、常见问题

`Config file error`：确认 `/etc/XrayR/config.yml` 存在、容器可读且 YAML 缩进正确。

`request ... failed`：在宿主机测试 `curl -v "https://面板地址/mod_mu/nodes/节点ID/info?key=密钥"`，并检查 DNS、HTTPS 证书和面板防火墙。

`server port must > 0`：面板没有返回有效端口，通常是 NodeID、NodeType 或 SSPanel 节点配置错误。

容器运行但客户端不通：检查 `docker compose logs`、服务器安全组、防火墙、节点 TCP/UDP 端口、域名解析和证书路径。

依赖下载失败：构建时切换 Go 代理，例如：

```bash
GOPROXY=https://goproxy.cn,direct docker compose build
```

如果该代理不可用，改为 `https://proxy.golang.org,direct` 或你自己的可信代理。
