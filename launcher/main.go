// BM-DROID Launcher — ملف تنفيذي واحد محمي بترخيص، الـ APK مدمج بداخله.
// يعمل كـ Setup في أول تشغيل (يفحص المتطلبات وينزّل الرن-تايم)،
// ثم كمشغّل kiosk في كل تشغيل لاحق.
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed assets/app.apk
var apkData []byte

//go:embed assets/config.json
var configRaw []byte

type Config struct {
	AppName         string `json:"app_name"`
	PackageName     string `json:"package_name"`
	LicenseKey      string `json:"license_key"`
	SupabaseURL     string `json:"supabase_url"`
	SupabaseAnonKey string `json:"supabase_anon_key"`
	RuntimeURL      string `json:"runtime_url"`
	RuntimeSHA256   string `json:"runtime_sha256"`
}

func fatal(format string, args ...any) {
	fmt.Printf("\n[خطأ] "+format+"\n", args...)
	fmt.Println("اضغط Enter للخروج...")
	fmt.Scanln()
	os.Exit(1)
}

func main() {
	var cfg Config
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		fatal("ملف الإعدادات تالف: %v", err)
	}

	fmt.Println("===============================================")
	fmt.Printf("  %s — BM-DROID Portable\n", cfg.AppName)
	fmt.Println("===============================================")

	if len(os.Args) > 1 && os.Args[1] == "--uninstall" {
		uninstall(&cfg)
		return
	}

	// 1) بصمة الجهاز
	hwid, err := machineGUID()
	if err != nil {
		fatal("تعذر قراءة بصمة الجهاز: %v", err)
	}

	// 2) التحقق من الترخيص أونلاين (Supabase)
	fmt.Println("[*] جارٍ التحقق من الترخيص...")
	lic, err := activateLicense(&cfg, hwid)
	if err != nil {
		fatal("لا يوجد اتصال بالإنترنت أو الخادم لا يستجيب.\nالتفاصيل: %v", err)
	}
	if !lic.OK {
		fatal("الترخيص غير صالح: %s", licenseErrorText(lic.Error))
	}
	fmt.Printf("[✓] ترخيص صالح — ينتهي في: %s\n", lic.ExpiresAt)

	// 3) مسار التثبيت
	dir := filepath.Join(os.Getenv("ProgramData"), sanitizeName(cfg.AppName))

	// 4) أول تشغيل = Setup
	if !isInstalled(dir) {
		fmt.Println("[*] أول تشغيل — بدء التثبيت...")
		if err := setup(&cfg, dir); err != nil {
			fatal("فشل التثبيت: %v", err)
		}
		fmt.Println("[✓] اكتمل التثبيت")
	}

	// 5) تشغيل النظام
	runEmulator(&cfg, dir, hwid)
}

func licenseErrorText(code string) string {
	switch code {
	case "invalid_key":
		return "مفتاح غير معروف"
	case "revoked":
		return "الترخيص تم إلغاؤه من الإدارة"
	case "expired":
		return "انتهت مدة الاشتراك"
	case "hwid_mismatch":
		return "هذا الترخيص مفعّل على جهاز آخر"
	default:
		return code
	}
}

func sanitizeName(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || (r >= 0x0600 && r <= 0x06FF) {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "BMDroidApp"
	}
	return string(out)
}
