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

**3X-UI** — advanced, open-source web-based control panel designed for managing Xray-core server. It offers a user-friendly interface for configuring and monitoring various VPN and proxy protocols.

> [!IMPORTANT]
> This project is only for personal usage, please do not use it for illegal purposes, and please do not use it in a production environment.

As an enhanced fork of the original X-UI project, 3X-UI provides improved stability, broader protocol support, and additional features.

## Custom GeoSite / GeoIP DAT sources

Administrators can add custom GeoSite and GeoIP `.dat` files from URLs in the panel (same workflow as updating built-in geofiles). Files are stored under the same directory as the Xray binary (`XUI_BIN_FOLDER`, default `bin/`) with deterministic names: `geosite_&lt;alias&gt;.dat` and `geoip_&lt;alias&gt;.dat`.

**Routing:** Xray resolves extra lists using the `ext:` form, for example `ext:geosite_myalias.dat:tag` or `ext:geoip_myalias.dat:tag`, where `tag` is a list name inside that DAT file (same pattern as built-in regional files such as `ext:geoip_IR.dat:ir`).

**Reserved aliases:** Only for deciding whether a name is reserved, the panel compares a normalized form of the alias (`strings.ToLower`, `-` → `_`). User-entered aliases and generated file names are not rewritten in the database; they must still match `^[a-z0-9_-]+$`. For example, `geoip-ir` and `geoip_ir` collide with the same reserved entry.

## Quick Start

```bash
bash <(curl -Ls https://raw.githubusercontent.com/igor231223/3x-ui_with_plugins/main/install.sh)
```

For full documentation, please visit the [project Wiki](https://github.com/igor231223/3x-ui_with_plugins/wiki).

## Acknowledgment

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules) (License: **GPL-3.0**): _Enhanced v2ray/xray and v2ray/xray-clients routing rules with built-in Iranian domains and a focus on security and adblocking._
- [Russia v2ray rules](https://github.com/runetfreedom/russia-v2ray-rules-dat) (License: **GPL-3.0**): _This repository contains automatically updated V2Ray routing rules based on data on blocked domains and addresses in Russia._

## Maintainer

- [igor231223](https://github.com/igor231223)
