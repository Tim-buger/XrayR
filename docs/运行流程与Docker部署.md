# XrayR 运行流程与 Docker Compose 部署

## 一、程序在做什么

XrayR 不是面板，也不保存用户数据库。它是 SSPanel-UIM 和 xray-core 之间的后端：

1. 从 `config.yml` 读取面板地址、密钥、节点 ID 和本地控制参数。
2. 请求 SSPanel-UIM 的 `/mod_mu` API，取得节点配置、用户列表和审计规则。
3. 把面板数据转换为 xray-core 的入站、出站和用户配置。
4. xray-core 接收客户端连接，并统计每个用户的流量与在线 IP。
5. 按 `UpdatePeriodic` 周期重新同步节点/用户，并向面板上报流量、在线 IP、系统状态和审计命中记录。
6. `config.yml` 发生变化时，程序关闭旧实例并在进程内重新启动。

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

## 三、首次部署

要求：Linux 服务器、Docker Engine、Docker Compose v2，以及可以访问面板 API。

```bash
cd /opt/XrayR
cp release/config/config.sspanel.yml.example release/config/config.yml
nano release/config/config.yml
docker compose build
docker compose up -d
docker compose logs -f --tail=200
```

至少修改：

- `ApiHost`：SSPanel-UIM 地址，例如 `https://panel.example.com`，末尾不要加 `/`。
- `ApiKey`：面板后端使用的 `muKey`。
- `NodeID`：面板后台节点 ID。
- `NodeType`：必须与节点类型匹配。
- TLS/REALITY：按面板节点配置选择，不要同时随意启用。

Compose 使用 `network_mode: host`，因为监听端口由面板动态下发。该模式仅建议在 Linux 服务器使用，也意味着容器监听的端口就是宿主机端口，防火墙必须放行节点的 TCP/UDP 端口。

## 四、证书模式

- `CertMode: none`：不使用 TLS，或使用 REALITY。
- `CertMode: file`：把证书和私钥放入 `release/config/cert/`，配置容器路径 `/etc/XrayR/cert/...`。
- `CertMode: http`：自动申请，域名的 80 端口必须可从公网访问。
- `CertMode: tls`：自动申请，域名的 443 端口必须可从公网访问。
- `CertMode: dns`：通过 DNS 服务商 API 申请，需要正确填写 `Provider` 和 `DNSEnv`。

证书目录通过卷保存在宿主机，重建容器不会丢失。

## 五、更新与回滚

更新源码后：

```bash
docker compose build --pull
docker compose up -d
docker compose logs -f --tail=200
```

上线前建议把当前源码目录和 `release/config/` 一起备份。原仓库不可用时，不要依赖 `latest` 镜像；应给自己的源码打 Git 标签，并为验证过的镜像增加固定标签。

## 六、常见问题

`Config file error`：确认 `release/config/config.yml` 存在且 YAML 缩进正确。

`request ... failed`：在宿主机测试 `curl -v "https://面板地址/mod_mu/nodes/节点ID/info?key=密钥"`，并检查 DNS、HTTPS 证书和面板防火墙。

`server port must > 0`：面板没有返回有效端口，通常是 NodeID、NodeType 或 SSPanel 节点配置错误。

容器运行但客户端不通：检查 `docker compose logs`、服务器安全组、防火墙、节点 TCP/UDP 端口、域名解析和证书路径。

依赖下载失败：构建时切换 Go 代理，例如：

```bash
GOPROXY=https://goproxy.cn,direct docker compose build
```

如果该代理不可用，改为 `https://proxy.golang.org,direct` 或你自己的可信代理。
