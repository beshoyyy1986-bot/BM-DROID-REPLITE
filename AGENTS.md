# AGENTS.md — BM-DROID

## نظرة عامة
نظام تحويل APK → EXE محمي بترخيص أونلاين (Supabase)، يعمل على Android-x86 9
مصغّر داخل QEMU portable بوضع kiosk. اللغة في الواجهات والرسائل: عربي (RTL).

## البنية والتقنيات
- `admin-panel/` — Node.js + Express + @supabase/supabase-js (service_role). UI: HTML/CSS/JS خام في `public/`، بلا framework.
- `builder/build.js` — يستدعي `go build` على نسخة `launcher/` بعد كتابة `assets/app.apk` + `assets/config.json`، الناتج في `dist/`.
- `launcher/` — Go (stdlib + golang.org/x/sys فقط). ملفات `*_windows.go`. الـ APK والـ config مدمجان عبر `go:embed`. Go portable في `tools/go/`.
- `android-image/build-image.sh` — bash (WSL/Linux): يحمل Android-x86 9 ISO، يحذف التطبيقات الافتراضية (`REMOVE_APPS`)، يضيف KioskHome كمشغّل وحيد، يعيد بناء الـ ISO.
- `android-image/kiosk-home/` — مشروع Android (Java, AGP 8.1.4) يُبنى بـ `./gradlew assembleRelease`، الناتج يُنسخ لـ `android-image/kiosk-home.apk` قبل تشغيل build-image.sh.
- `runtime/make-runtime.ps1` — PowerShell: يجهز `runtime.zip` (QEMU + adb + android.iso + data.img) ويطبع SHA256.
- `supabase/schema.sql` — جدول `licenses` + دالة `activate_license(key, hwid)` بصلاحية anon.

## قواعد مهمة
- **الترخيص**: anon key لا يقرأ جدول licenses أبدًا — كل التحقق عبر RPC فقط. الربط بجهاز واحد عبر `MachineGuid` من الريجستري.
- **المدة تبدأ من أول تفعيل** (`activated_at`) وليس من الإنشاء.
- في `launcher/`: لا تضف تبعيات خارج `golang.org/x/sys` إلا للضرورة — الهدف EXE صغير سريع التجميع.
- `builder/build.js` يكتب داخل `launcher/assets/` مباشرة — هذه ملفات مؤقتة مدرجة في .gitignore.
- قائمة الحذف في `build-image.sh` هي نقطة التحكم الوحيدة في "تصغير" النظام — عدّل `REMOVE_APPS` هناك فقط.

## الأوامر
```powershell
# لوحة الأدمن
cd admin-panel; npm install; npm start        # http://localhost:3000

# بناء يدوي للـ EXE (بدون اللوحة)
node builder/build.js job.json                # job.json: appName/packageName/licenseKey/apkPath/supabaseUrl/supabaseAnonKey/runtimeUrl/runtimeSha256

# تجميع الـ launcher منفردًا للاختبار
tools\go\bin\go.exe build -ldflags "-s -w" -o dist\test.exe .   # من داخل launcher/
```

## الاختبار
- `go vet ./...` داخل `launcher/` قبل أي تعديل.
- لا يوجد CI. اختبر تصديرًا كاملًا بـ APK تجريبي: `node builder/build.js job.json`.
