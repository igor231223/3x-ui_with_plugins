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

**3X-UI** — لوحة تحكم متقدمة مفتوحة المصدر تعتمد على الويب مصممة لإدارة خادم Xray-core. توفر واجهة سهلة الاستخدام لتكوين ومراقبة بروتوكولات VPN والوكيل المختلفة.

> [!IMPORTANT]
> هذا المشروع مخصص للاستخدام الشخصي والاتصال فقط، يرجى عدم استخدامه لأغراض غير قانونية، يرجى عدم استخدامه في بيئة الإنتاج.

كمشروع محسن من مشروع X-UI الأصلي، يوفر 3X-UI استقرارًا محسنًا ودعمًا أوسع للبروتوكولات وميزات إضافية.

## مصادر DAT مخصصة GeoSite / GeoIP

يمكن للمسؤولين إضافة ملفات `.dat` لـ GeoSite وGeoIP من عناوين URL في اللوحة (نفس أسلوب تحديث ملفات الجيو المدمجة). تُحفظ الملفات بجانب ثنائي Xray (`XUI_BIN_FOLDER`، الافتراضي `bin/`) بأسماء ثابتة: `geosite_&lt;alias&gt;.dat` و`geoip_&lt;alias&gt;.dat`.

**التوجيه:** استخدم الصيغة `ext:`، مثل `ext:geosite_myalias.dat:tag` أو `ext:geoip_myalias.dat:tag`، حيث `tag` اسم قائمة داخل ملف DAT (كما في `ext:geoip_IR.dat:ir`).

**الأسماء المحجوزة:** يُقارَن شكل مُطبَّع فقط لمعرفة التحفظ (`strings.ToLower`، `-` → `_`). لا تُعاد كتابة الأسماء التي يدخلها المستخدم أو سجلات قاعدة البيانات؛ يجب أن تطابق `^[a-z0-9_-]+$`. مثلاً `geoip-ir` و`geoip_ir` يصطدمان بنفس الحجز.

## البدء السريع

```
bash <(curl -Ls https://raw.githubusercontent.com/igor231223/3x-ui_with_plugins/main/install.sh)
```

للحصول على الوثائق الكاملة، يرجى زيارة [ويكي المشروع](https://github.com/igor231223/3x-ui_with_plugins/wiki).

## الاعتراف

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules) (الترخيص: **GPL-3.0**): _قواعد توجيه v2ray/xray و v2ray/xray-clients المحسنة مع النطاقات الإيرانية المدمجة وتركيز على الأمان وحظر الإعلانات._
- [Russia v2ray rules](https://github.com/runetfreedom/russia-v2ray-rules-dat) (الترخيص: **GPL-3.0**): _يحتوي هذا المستودع على قواعد توجيه V2Ray محدثة تلقائيًا بناءً على بيانات النطاقات والعناوين المحظورة في روسيا._

## الصيانة

- [igor231223](https://github.com/igor231223)
