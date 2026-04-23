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

**3X-UI** — panel de control avanzado basado en web de código abierto diseñado para gestionar el servidor Xray-core. Ofrece una interfaz fácil de usar para configurar y monitorear varios protocolos VPN y proxy.

> [!IMPORTANT]
> Este proyecto es solo para uso personal y comunicación, por favor no lo use para fines ilegales, por favor no lo use en un entorno de producción.

Como una versión mejorada del proyecto X-UI original, 3X-UI proporciona mayor estabilidad, soporte más amplio de protocolos y características adicionales.

## Fuentes DAT personalizadas GeoSite / GeoIP

Los administradores pueden añadir archivos `.dat` de GeoSite y GeoIP desde URLs en el panel (mismo flujo que los geoficheros integrados). Los archivos se guardan junto al binario de Xray (`XUI_BIN_FOLDER`, por defecto `bin/`) con nombres fijos: `geosite_&lt;alias&gt;.dat` y `geoip_&lt;alias&gt;.dat`.

**Enrutamiento:** use la forma `ext:`, por ejemplo `ext:geosite_myalias.dat:tag` o `ext:geoip_myalias.dat:tag`, donde `tag` es un nombre de lista dentro del DAT (igual que en archivos regionales como `ext:geoip_IR.dat:ir`).

**Alias reservados:** solo para comprobar si un nombre está reservado se compara una forma normalizada (`strings.ToLower`, `-` → `_`). Los alias introducidos y los nombres en la base de datos no se reescriben; deben cumplir `^[a-z0-9_-]+$`. Por ejemplo, `geoip-ir` y `geoip_ir` chocan con la misma entrada reservada.

## Inicio Rápido

```
bash <(curl -Ls https://raw.githubusercontent.com/igor231223/3x-ui_with_plugins/main/install.sh)
```

Para documentación completa, visita la [Wiki del proyecto](https://github.com/igor231223/3x-ui_with_plugins/wiki).

## Reconocimientos

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules) (Licencia: **GPL-3.0**): _Reglas de enrutamiento mejoradas para v2ray/xray y v2ray/xray-clients con dominios iraníes incorporados y un enfoque en seguridad y bloqueo de anuncios._
- [Russia v2ray rules](https://github.com/runetfreedom/russia-v2ray-rules-dat) (Licencia: **GPL-3.0**): _Este repositorio contiene reglas de enrutamiento V2Ray actualizadas automáticamente basadas en datos de dominios y direcciones bloqueadas en Rusia._

## Mantenedor

- [igor231223](https://github.com/igor231223)
