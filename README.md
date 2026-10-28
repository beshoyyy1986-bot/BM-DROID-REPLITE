<<<<<<< HEAD
# BM-DROID — مصدّر تطبيقات أندرويد كـ EXE محمي

حوّل أي APK إلى ملف **EXE واحد للويندوز** محمي بترخيص أونلاين، يعمل على نظام
**Android-x86 9 مصغّر** (بدون هاتف، ساعة، منبه، كاميرا أو أي تطبيقات افتراضية)
داخل **QEMU portable** بوضع Kiosk — المستخدم لا يرى سوى تطبيقك.

## المكونات

| المجلد | الوظيفة |
|---|---|
| `admin-panel/` | لوحة الأدمن (Express + Supabase): رفع APK، إصدار تراخيص، تصدير |
| `builder/` | مولّد الـ EXE — يدمج الـ APK + إعدادات الترخيص داخل الملف |
| `launcher/` | كود Go للـ EXE: تحقق ترخيص + setup + تشغيل kiosk |
| `android-image/` | سكربت تصغير صورة Android-x86 9 + تطبيق Kiosk Home |
| `runtime/` | سكربت تجهيز حزمة QEMU portable (`runtime.zip`) |
| `supabase/` | سكيمة التراخيص + دالة التفعيل الآمنة |

## التجهيز (مرة واحدة)

### 1) Supabase
1. أنشئ مشروعًا على [supabase.com](https://supabase.com)
2. نفّذ `supabase/schema.sql` في SQL Editor

### 2) صورة أندرويد المصغرة (WSL أو Linux)
```bash
cd android-image/kiosk-home && ./gradlew assembleRelease   # أو ابنِها من Android Studio
cp app/build/outputs/apk/release/app-release.apk ../kiosk-home.apk
cd .. && ./build-image.sh    # الناتج: out/android.iso + out/data.img
```
التطبيقات المحذوفة معرفة في مصفوفة `REMOVE_APPS` داخل السكربت — عدّلها كما تشاء.

### 3) حزمة الرن-تايم (ويندوز)
```powershell
.\runtime\make-runtime.ps1
# ارفع runtime/out/runtime.zip على Supabase Storage (أو أي استضافة)
# وانسخ الرابط المباشر + الـ SHA256 المطبوع
```

### 4) لوحة الأدمن
```powershell
cd admin-panel
copy .env.example .env   # املأ: دخول الأدمن + مفاتيح Supabase + RUNTIME_URL/SHA256
npm install
npm start                # http://localhost:3000
```

## الاستخدام اليومي

1. افتح اللوحة → **تصدير جديد**
2. ارفع الـ APK، اكتب اسم التطبيق والحزمة، حدد مدة الاشتراك
3. اضغط **تصدير EXE** → حمّل الملف وسلّمه للعميل
4. من تبويب **التراخيص**: إلغاء / إعادة تفعيل / فك ربط الجهاز / حذف

## ما يحدث عند العميل

1. يشغّل الـ EXE → يتحقق من الترخيص أونلاين (Supabase) ويربطه بجهازه
2. يفحص المتطلبات (4GB RAM، مساحة 6GB، Hypervisor Platform)
3. ينزّل حزمة التشغيل (أول مرة فقط) ويتحقق من SHA256
4. يثبّت الـ APK داخل النظام تلقائيًا ويضبط الـ kiosk
5. كل تشغيل لاحق: تحقق ترخيص → تطبيقك يفتح مباشرة في نافذة بحجم هاتف
6. فحص دوري كل 30 دقيقة — لو الترخيص انتهى أو اتلغى يتقفل فورًا

إلغاء التثبيت: من مجلد البرنامج في `ProgramData` أو تشغيل الـ EXE مع `--uninstall`.
=======
# BM-DROID-NEW
>>>>>>> 2962d7fcef764e96b45e27433b221839113f1bd6
