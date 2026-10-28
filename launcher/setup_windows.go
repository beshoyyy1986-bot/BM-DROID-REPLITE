package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func installDirOf(dir string) string { return dir }

func isInstalled(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "runtime", "android.iso"))
	return err == nil
}

func setup(cfg *Config, dir string) error {
	// فحص المتطلبات قبل تنزيل أي شيء
	if err := checkRequirements(dir); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// 1) تنزيل حزمة الرن-تايم (QEMU + صورة أندرويد المصغرة)
	zipPath := filepath.Join(dir, "runtime.zip")
	if err := downloadWithProgress(cfg.RuntimeURL, zipPath); err != nil {
		return fmt.Errorf("فشل تنزيل ملفات التشغيل: %w", err)
	}

	// 2) التحقق من السلامة
	if cfg.RuntimeSHA256 != "" {
		fmt.Println("[*] التحقق من سلامة الملفات...")
		sum, err := sha256File(zipPath)
		if err != nil {
			return err
		}
		if !strings.EqualFold(sum, cfg.RuntimeSHA256) {
			os.Remove(zipPath)
			return fmt.Errorf("الملف المنزّل غير سليم (SHA256 غير متطابق)")
		}
	}

	// 3) فك الضغط
	fmt.Println("[*] استخراج الملفات...")
	if err := unzip(zipPath, filepath.Join(dir, "runtime")); err != nil {
		return fmt.Errorf("فشل فك الضغط: %w", err)
	}
	os.Remove(zipPath)

	// 4) استخراج الـ APK المدمج
	apkPath := filepath.Join(dir, "app.apk")
	if err := os.WriteFile(apkPath, apkData, 0o644); err != nil {
		return err
	}

	// 5) ملف إعدادات الـ kiosk (اسم الحزمة المستهدفة)
	kioskDir := filepath.Join(dir, "kiosk")
	os.MkdirAll(kioskDir, 0o755)
	os.WriteFile(filepath.Join(kioskDir, "target.txt"), []byte(cfg.PackageName), 0o644)

	// 6) نسخ الـ EXE نفسه داخل مجلد التثبيت + اختصار على سطح المكتب
	self, _ := os.Executable()
	selfCopy := filepath.Join(dir, filepath.Base(self))
	if !strings.EqualFold(self, selfCopy) {
		copyFile(self, selfCopy)
	}
	if err := createShortcut(selfCopy, cfg.AppName); err != nil {
		fmt.Println("[!] تعذر إنشاء اختصار سطح المكتب:", err)
	}

	// 7) سكربت إزالة التثبيت
	os.WriteFile(filepath.Join(dir, "إلغاء التثبيت.bat"),
		[]byte("@echo off\r\n\""+selfCopy+"\" --uninstall\r\npause\r\n"), 0o644)

	return nil
}

func uninstall(cfg *Config) {
	dir := filepath.Join(os.Getenv("ProgramData"), sanitizeName(cfg.AppName))
	fmt.Println("[*] إيقاف أي محاكي يعمل...")
	exec.Command("taskkill", "/F", "/IM", "qemu-system-x86_64.exe").Run()
	exec.Command("taskkill", "/F", "/IM", "adb.exe").Run()
	time.Sleep(2 * time.Second)

	// حذف الاختصار
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop", cfg.AppName+".lnk")
	os.Remove(desktop)

	fmt.Println("[*] حذف ملفات التثبيت:", dir)
	self, _ := os.Executable()
	selfCopy := filepath.Join(dir, filepath.Base(self))
	// لا نستطيع حذف الملف الجاري تشغيله — نحذف كل شيء عدا الملف الحالي ثم نجدول حذفه
	if strings.EqualFold(self, selfCopy) {
		tmp := filepath.Join(os.TempDir(), "bmdroid-uninstall-"+sanitizeName(cfg.AppName)+".bat")
		bat := "@echo off\r\ntimeout /t 2 /nobreak >nul\r\nrmdir /s /q \"" + dir + "\"\r\ndel \"%~f0\"\r\n"
		os.WriteFile(tmp, []byte(bat), 0o644)
		exec.Command("cmd", "/C", "start", "", "/min", tmp).Start()
	} else {
		os.RemoveAll(dir)
	}
	fmt.Println("[✓] تمت إزالة التثبيت")
	fmt.Println("اضغط Enter للخروج...")
	fmt.Scanln()
}

func downloadWithProgress(url, dest string) error {
	fmt.Println("[*] تنزيل ملفات التشغيل (أول مرة فقط)...")
	fmt.Println("    ", url)

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	client := &http.Client{Timeout: 0} // بلا مهلة — ملف كبير
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	total := resp.ContentLength
	var done int64
	buf := make([]byte, 256*1024)
	lastPrint := time.Now()
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			done += int64(n)
			if time.Since(lastPrint) > 500*time.Millisecond {
				lastPrint = time.Now()
				if total > 0 {
					fmt.Printf("\r    %d / %d MB (%.0f%%)   ", done>>20, total>>20, float64(done)/float64(total)*100)
				} else {
					fmt.Printf("\r    %d MB   ", done>>20)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	fmt.Println()
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)
		// حماية من Zip Slip
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("مسار غير آمن داخل الأرشيف: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0o755)
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0o755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(w, rc)
		w.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// createShortcut ينشئ اختصار .lnk على سطح المكتب عبر PowerShell.
func createShortcut(target, name string) error {
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	lnk := filepath.Join(desktop, name+".lnk")
	ps := fmt.Sprintf(`$ws = New-Object -ComObject WScript.Shell; `+
		`$s = $ws.CreateShortcut('%s'); `+
		`$s.TargetPath = '%s'; `+
		`$s.WorkingDirectory = '%s'; `+
		`$s.Description = '%s'; `+
		`$s.Save()`,
		strings.ReplaceAll(lnk, "'", "''"),
		strings.ReplaceAll(target, "'", "''"),
		strings.ReplaceAll(filepath.Dir(target), "'", "''"),
		strings.ReplaceAll(name, "'", "''"))
	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Run()
}
