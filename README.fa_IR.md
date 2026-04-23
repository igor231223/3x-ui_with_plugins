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

**3X-UI** — یک پنل کنترل پیشرفته مبتنی بر وب با کد باز که برای مدیریت سرور Xray-core طراحی شده است. این پنل یک رابط کاربری آسان برای پیکربندی و نظارت بر پروتکل‌های مختلف VPN و پراکسی ارائه می‌دهد.

> [!IMPORTANT]
> این پروژه فقط برای استفاده شخصی و ارتباطات است، لطفاً از آن برای اهداف غیرقانونی استفاده نکنید، لطفاً از آن در محیط تولید استفاده نکنید.

به عنوان یک نسخه بهبود یافته از پروژه اصلی X-UI، 3X-UI پایداری بهتر، پشتیبانی گسترده‌تر از پروتکل‌ها و ویژگی‌های اضافی را ارائه می‌دهد.

## منابع DAT سفارشی GeoSite / GeoIP

سرپرستان می‌توانند از طریق پنل فایل‌های `.dat` GeoSite و GeoIP را از URL اضافه کنند (همان الگوی به‌روزرسانی ژئوفایل‌های داخلی). فایل‌ها در کنار باینری Xray (`XUI_BIN_FOLDER`، پیش‌فرض `bin/`) با نام‌های ثابت `geosite_&lt;alias&gt;.dat` و `geoip_&lt;alias&gt;.dat` ذخیره می‌شوند.

**مسیریابی:** از شکل `ext:` استفاده کنید، مثلاً `ext:geosite_myalias.dat:tag` یا `ext:geoip_myalias.dat:tag`؛ `tag` نام لیست داخل همان DAT است (مانند `ext:geoip_IR.dat:ir`).

**نام‌های رزرو:** فقط برای تشخیص رزرو بودن، نسخه نرمال‌شده (`strings.ToLower`، `-` → `_`) مقایسه می‌شود. نام‌های واردشده و رکورد پایگاه داده بازنویسی نمی‌شوند و باید با `^[a-z0-9_-]+$` سازگار باشند؛ مثلاً `geoip-ir` و `geoip_ir` به یک رزرو یکسان می‌خورند.

## شروع سریع

```
bash <(curl -Ls https://raw.githubusercontent.com/igor231223/3x-ui_with_plugins/main/install.sh)
```

برای مستندات کامل، لطفاً به [ویکی پروژه](https://github.com/igor231223/3x-ui_with_plugins/wiki) مراجعه کنید.

## قدردانی

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules) (مجوز: **GPL-3.0**): _قوانین مسیریابی بهبود یافته v2ray/xray و v2ray/xray-clients با دامنه‌های ایرانی داخلی و تمرکز بر امنیت و مسدود کردن تبلیغات._
- [Russia v2ray rules](https://github.com/runetfreedom/russia-v2ray-rules-dat) (مجوز: **GPL-3.0**): _این مخزن شامل قوانین مسیریابی V2Ray به‌روزرسانی شده خودکار بر اساس داده‌های دامنه‌ها و آدرس‌های مسدود شده در روسیه است._

## نگهدارنده

- [igor231223](https://github.com/igor231223)
