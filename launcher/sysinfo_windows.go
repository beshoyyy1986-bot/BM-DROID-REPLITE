package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// machineGUID يقرأ بصمة الجهاز من ريجستري ويندوز (فريدة لكل جهاز).
func machineGUID() (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Cryptography`, registry.READ|registry.WOW64_64KEY)
	if err != nil {
		return "", err
	}
	defer key.Close()

	guid, _, err := key.GetStringValue("MachineGuid")
	if err != nil {
		return "", err
	}
	return guid, nil
}

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func totalRAMBytes() uint64 {
	var ms memoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms))
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")
	r, _, _ := proc.Call(uintptr(unsafe.Pointer(&ms)))
	if r == 0 {
		return 0
	}
	return ms.TotalPhys
}

func freeDiskBytes(dir string) uint64 {
	var free, total, avail uint64
	p, _ := syscall.UTF16PtrFromString(filepath.VolumeName(dir) + `\`)
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")
	r, _, _ := proc.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&avail)),
		uintptr(unsafe.Pointer(&total)),
		uintptr(unsafe.Pointer(&free)),
	)
	if r == 0 {
		return 0
	}
	return free
}

// hasWHPX يتحقق من وجود Windows Hypervisor Platform (تسريع المحاكاة).
func hasWHPX() bool {
	sysroot := os.Getenv("SystemRoot")
	if sysroot == "" {
		sysroot = `C:\Windows`
	}
	_, err := os.Stat(filepath.Join(sysroot, "System32", "WinHvPlatform.dll"))
	return err == nil
}

// checkRequirements يفحص الحد الأدنى من المتطلبات قبل التثبيت.
func checkRequirements(dir string) error {
	if totalRAMBytes() < 4<<30 {
		return fmt.Errorf("الذاكرة أقل من 4 جيجا — المطلوب 4GB على الأقل")
	}
	if freeDiskBytes(dir) < 6<<30 {
		return fmt.Errorf("مساحة القرص المتاحة أقل من 6 جيجا")
	}
	if !hasWHPX() {
		fmt.Println("[!] تنبيه: ميزة 'Windows Hypervisor Platform' غير مفعّلة.")
		fmt.Println("    سيعمل النظام بوضع بطيء. لتفعيلها:")
		fmt.Println("    optionalfeatures.exe ← Windows Hypervisor Platform")
	}
	return nil
}
