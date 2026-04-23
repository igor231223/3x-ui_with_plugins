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

**3X-UI** — продвинутая панель управления с открытым исходным кодом на основе веб-интерфейса, разработанная для управления сервером Xray-core. Предоставляет удобный интерфейс для настройки и мониторинга различных VPN и прокси-протоколов.

> [!IMPORTANT]
> Этот проект предназначен только для личного использования, пожалуйста, не используйте его в незаконных целях и в производственной среде.

Как улучшенная версия оригинального проекта X-UI, 3X-UI обеспечивает повышенную стабильность, более широкую поддержку протоколов и дополнительные функции.

## Пользовательские GeoSite / GeoIP (DAT)

В панели можно задать свои источники `.dat` по URL (тот же сценарий, что и для встроенных геофайлов). Файлы сохраняются в каталоге с бинарником Xray (`XUI_BIN_FOLDER`, по умолчанию `bin/`) как `geosite_&lt;alias&gt;.dat` и `geoip_&lt;alias&gt;.dat`.

**Маршрутизация:** в правилах используйте форму `ext:имя_файла.dat:тег`, например `ext:geosite_myalias.dat:tag` (как у региональных списков `ext:geoip_IR.dat:ir`).

**Зарезервированные псевдонимы:** только для проверки на резерв используется нормализованная форма (`strings.ToLower`, `-` → `_`). Введённые пользователем псевдонимы и имена файлов в БД не переписываются и должны соответствовать `^[a-z0-9_-]+$`. Например, `geoip-ir` и `geoip_ir` попадают под одну и ту же зарезервированную запись.

## Быстрый старт

```
bash <(curl -Ls https://raw.githubusercontent.com/igor231223/3x-ui_with_plugins/main/install.sh)
```

Полную документацию смотрите в [вики проекта](https://github.com/igor231223/3x-ui_with_plugins/wiki).

## Благодарности

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules) (Лицензия: **GPL-3.0**): _Улучшенные правила маршрутизации для v2ray/xray и v2ray/xray-clients со встроенными иранскими доменами и фокусом на безопасность и блокировку рекламы._
- [Russia v2ray rules](https://github.com/runetfreedom/russia-v2ray-rules-dat) (Лицензия: **GPL-3.0**): _Этот репозиторий содержит автоматически обновляемые правила маршрутизации V2Ray на основе данных о заблокированных доменах и адресах в России._

## Поддержка

- [igor231223](https://github.com/igor231223)
