package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// runEmulator يشغّل QEMU بصورة أندرويد المصغرة في وضع kiosk،
// ويثبّت الـ APK في أول إقلاع، ويعيد فحص الترخيص دوريًا.
func runEmulator(cfg *Config, dir, hwid string) {
	rt := filepath.Join(dir, "runtime")
	qemu := filepath.Join(rt, "qemu", "qemu-system-x86_64.exe")
	iso := filepath.Join(rt, "android.iso")
	dataImg := filepath.Join(rt, "data.img")
	adb := filepath.Join(rt, "adb", "adb.exe")

	accel := "whpx"
	if !hasWHPX() {
		accel = "tcg"
		fmt.Println("[!] يعمل بدون تسريع عتادي — الأداء سيكون بطيئًا")
	}

	args := []string{
		"-accel", accel,
		"-smp", "2",
		"-m", "2048",
		"-drive", "index=0,if=ide,id=cdrom,media=cdrom,file=" + iso,
		"-drive", "index=1,if=ide,id=data,file=" + dataImg + ",format=raw",
		"-netdev", "user,id=net0,hostfwd=tcp:127.0.0.1:5555-:5555",
		"-device", "e1000,netdev=net0",
		"-vga", "std",
		"-display", "sdl",
		"-usb", "-device", "usb-tablet",
		"-audiodev", "dsound,id=snd0", "-device", "AC97,audiodev=snd0",
		"-name", cfg.AppName,
	}

	fmt.Println("[*] تشغيل النظام...")
	cmd := exec.Command(qemu, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		fatal("تعذر تشغيل المحاكي: %v", err)
	}

	// إقلاع أول مرة: تثبيت الـ APK وضبط هدف الـ kiosk
	go provision(adb, cfg, dir)

	// إعادة فحص الترخيص كل 30 دقيقة — لو انتهى أو اتلغى، يقفل فورًا
	stopWatch := make(chan struct{})
	go func() {
		t := time.NewTicker(30 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-stopWatch:
				return
			case <-t.C:
				lic, err := activateLicense(cfg, hwid)
				if err != nil {
					continue // مشكلة شبكة مؤقتة — لا نعاقب المستخدم
				}
				if !lic.OK {
					fmt.Println("\n[!] الترخيص لم يعد صالحًا:", licenseErrorText(lic.Error))
					killQemu()
				}
			}
		}
	}()

	cmd.Wait()
	close(stopWatch)
	fmt.Println("[*] تم إغلاق النظام")
}

// provision ينتظر الإقلاع ثم يثبّت التطبيق ويكتب إعدادات الـ kiosk (أول مرة فقط).
func provision(adb string, cfg *Config, dir string) {
	marker := filepath.Join(dir, ".provisioned")
	waitBoot(adb)

	if _, err := os.Stat(marker); err == nil {
		return // تم التجهيز من قبل
	}

	adbRun(adb, "root")
	time.Sleep(3 * time.Second)
	adbRun(adb, "connect", "127.0.0.1:5555")
	waitBoot(adb)

	fmt.Println("[*] تثبيت التطبيق داخل النظام (أول مرة فقط)...")
	if out, err := adbRun(adb, "install", "-r", "-g", filepath.Join(dir, "app.apk")); err != nil {
		fmt.Println("[!] فشل تثبيت التطبيق:", strings.TrimSpace(out))
	} else {
		fmt.Println("[✓] تم تثبيت التطبيق")
	}

	adbRun(adb, "shell", "mkdir", "-p", "/sdcard/kiosk")
	adbRun(adb, "push", filepath.Join(dir, "kiosk", "target.txt"), "/sdcard/kiosk/target.txt")
	// امنح صلاحيات الكشك: إبقاء الشاشة مستيقظة
	adbRun(adb, "shell", "settings", "put", "global", "stay_on_while_plugged_in", "7")

	os.WriteFile(marker, []byte(time.Now().String()), 0o644)
	fmt.Println("[✓] النظام جاهز — سيظهر تطبيقك خلال لحظات")
}

func waitBoot(adb string) {
	fmt.Println("[*] في انتظار إقلاع النظام...")
	for i := 0; i < 120; i++ {
		adbRun(adb, "connect", "127.0.0.1:5555")
		out, _ := adbRun(adb, "shell", "getprop", "sys.boot_completed")
		if strings.TrimSpace(out) == "1" {
			return
		}
		time.Sleep(3 * time.Second)
	}
}

func adbRun(adb string, args ...string) (string, error) {
	cmd := exec.Command(adb, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func killQemu() {
	exec.Command("taskkill", "/F", "/IM", "qemu-system-x86_64.exe").Run()
}
