[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) |  [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./media/3x-ui-dark.png">
    <img alt="3x-ui" src="./media/3x-ui-light.png">
  </picture>
</p>

[![Release](https://img.shields.io/github/v/release/igor231223/3x-ui_with_plugins.svg)](https://github.com/igor231223/3x-ui_with_plugins/releases)
[![Build](https://img.shields.io/github/actions/workflow/status/igor231223/3x-ui_with_plugins/release.yml.svg)](https://github.com/igor231223/3x-ui_with_plugins/actions)
[![GO Version](https://img.shields.io/github/go-mod/go-version/igor231223/3x-ui_with_plugins.svg)](#)
[![Downloads](https://img.shields.io/github/downloads/igor231223/3x-ui_with_plugins/total.svg)](https://github.com/igor231223/3x-ui_with_plugins/releases/latest)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)
[![Go Reference](https://pkg.go.dev/badge/github.com/igor231223/3x-ui_with_plugins/v2.svg)](https://pkg.go.dev/github.com/igor231223/3x-ui_with_plugins/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/igor231223/3x-ui_with_plugins/v2)](https://goreportcard.com/report/github.com/igor231223/3x-ui_with_plugins/v2)

**3X-UI** — 一个基于网页的高级开源控制面板，专为管理 Xray-core 服务器而设计。它提供了用户友好的界面，用于配置和监控各种 VPN 和代理协议。

> [!IMPORTANT]
> 本项目仅用于个人使用和通信，请勿将其用于非法目的，请勿在生产环境中使用。

作为原始 X-UI 项目的增强版本，3X-UI 提供了更好的稳定性、更广泛的协议支持和额外的功能。

## 自定义 GeoSite / GeoIP（DAT）

管理员可在面板中从 URL 添加自定义 GeoSite 与 GeoIP `.dat` 文件（与内置地理文件相同的管理流程）。文件保存在 Xray 可执行文件所在目录（`XUI_BIN_FOLDER`，默认 `bin/`），文件名为 `geosite_&lt;alias&gt;.dat` 和 `geoip_&lt;alias&gt;.dat`。

**路由：** 在规则中使用 `ext:` 形式，例如 `ext:geosite_myalias.dat:tag` 或 `ext:geoip_myalias.dat:tag`，其中 `tag` 为该 DAT 文件内的列表名（与内置区域文件如 `ext:geoip_IR.dat:ir` 相同）。

**保留别名：** 仅在为判断是否命中保留名时，会对别名做规范化比较（`strings.ToLower`，`-` → `_`）。用户输入的别名与数据库中的名称不会被改写，且须符合 `^[a-z0-9_-]+$`。例如 `geoip-ir` 与 `geoip_ir` 视为同一保留项。

## 快速开始

```
bash <(curl -Ls https://raw.githubusercontent.com/igor231223/3x-ui_with_plugins/main/install.sh)
```

完整文档请参阅 [项目Wiki](https://github.com/igor231223/3x-ui_with_plugins/wiki)。

## 致谢

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules) (许可证: **GPL-3.0**): _增强的 v2ray/xray 和 v2ray/xray-clients 路由规则，内置伊朗域名，专注于安全性和广告拦截。_
- [Russia v2ray rules](https://github.com/runetfreedom/russia-v2ray-rules-dat) (许可证: **GPL-3.0**): _此仓库包含基于俄罗斯被阻止域名和地址数据自动更新的 V2Ray 路由规则。_

## 维护者

- [igor231223](https://github.com/igor231223)
