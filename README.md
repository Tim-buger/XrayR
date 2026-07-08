# XrayR

![](https://github.com/XrayR-project/XrayR/actions/workflows/release.yml/badge.svg)
![](https://github.com/XrayR-project/XrayR/actions/workflows/docker.yml/badge.svg)

一个面向 SSPanel-UIM 的 Xray 后端，支持 V2Ray、Trojan、Shadowsocks 协议。

使用教程：[详细使用教程](https://xrayr-project.github.io/XrayR-doc/)

## 免责声明

本项目只是本人个人学习开发并维护，本人不保证任何可用性，也不对使用本软件造成的任何后果负责。

## 特点

* 永久开源且免费。
* 支持V2ray，Trojan， Shadowsocks多种协议。
* 支持Vless和XTLS等新特性。
* 支持单实例对接多个 SSPanel-UIM 节点，无需重复启动。
* 支持限制在线IP
* 支持节点端口级别、用户级别限速。
* 配置简单明了。
* 修改配置自动重启实例。
* 方便编译和升级，可以快速更新核心版本， 支持Xray-core新特性。

## 功能介绍

| 功能        | v2ray | trojan | shadowsocks |
|-----------|-------|--------|-------------|
| 获取节点信息    | √     | √      | √           |
| 获取用户信息    | √     | √      | √           |
| 用户流量统计    | √     | √      | √           |
| 服务器信息上报   | √     | √      | √           |
| 自动申请tls证书 | √     | √      | √           |
| 自动续签tls证书 | √     | √      | √           |
| 在线人数统计    | √     | √      | √           |
| 在线用户限制    | √     | √      | √           |
| 审计规则      | √     | √      | √           |
| 节点端口限速    | √     | √      | √           |
| 按照用户限速    | √     | √      | √           |
| 自定义DNS    | √     | √      | √           |

## 支持面板

当前源码仅保留 `sspanel-uim` 适配器，支持 V2Ray、Trojan 和 Shadowsocks
（包括单端口多用户与 V2Ray-Plugin）节点。

## 软件安装

### 一键安装

```
bash <(curl -fLsS https://raw.githubusercontent.com/Tim-buger/XrayR/master/install.sh)
```

### 使用Docker部署软件

[Docker部署教程](https://xrayr-project.github.io/XrayR-doc/xrayr-xia-zai-he-an-zhuang/install/docker)

本仓库已提供从源码构建的 `docker-compose.yml`。SSPanel-UIM 对接、运行流程、
配置项和部署命令见 [运行流程与 Docker 部署](docs/运行流程与Docker部署.md)。

### 手动安装

[手动安装教程](https://xrayr-project.github.io/XrayR-doc/xrayr-xia-zai-he-an-zhuang/install/manual)

## 配置文件及详细使用教程

[详细使用教程](https://xrayr-project.github.io/XrayR-doc/)

## Thanks

* [Project X](https://github.com/XTLS/)
* [V2Fly](https://github.com/v2fly)
* [VNet-V2ray](https://github.com/ProxyPanel/VNet-V2ray)
* [Air-Universe](https://github.com/crossfw/Air-Universe)

## Licence

[Mozilla Public License Version 2.0](https://github.com/Tim-buger/XrayR/blob/master/LICENSE)