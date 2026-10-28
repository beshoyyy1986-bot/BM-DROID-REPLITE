#!/usr/bin/env bash
# ============================================================
# بناء صورة Android-x86 9 مصغّرة (Kiosk) — شغّله في WSL أو Linux
# الناتج: android.iso + data.img  ← يتحطوا في runtime.zip
#
# المتطلبات: wget, p7zip, squashfs-tools, genisoimage, جزء من:
#   sudo apt install wget p7zip-full squashfs-tools genisoimage
# ============================================================
set -euo pipefail

ISO_URL="https://osdn.net/frs/redir.php?m=constant&f=android-x86%2F71931%2Fandroid-x86_64-9.0-r2.iso"
ISO_NAME="android-x86_64-9.0-r2.iso"
WORK="$(pwd)/work"
OUT="$(pwd)/out"

# التطبيقات الافتراضية المحذوفة — لا هاتف ولا ساعة ولا منبه ولا ألعاب
REMOVE_APPS=(
  Launcher3 QuickSearchBox DeskClock Contacts Dialer Telecom TeleService
  Camera2 Gallery2 Calendar Email Browser2 Mms Messaging Music MusicFX
  SoundRecorder VideoEditor LiveWallpapers HoloSpiralWallpaper BasicSmsReceiver
  CellBroadcastReceiver CarrierConfig Stk BookmarkProvider CalendarProvider
  ContactsProvider DownloadProviderUi PrintSpooler WallpaperPicker
  ManagedProvisioning OneTimeInitializer Protips EasterEgg Traceur
)

echo "==> تنزيل $ISO_NAME"
mkdir -p "$WORK" "$OUT"
[ -f "$WORK/$ISO_NAME" ] || wget -O "$WORK/$ISO_NAME" "$ISO_URL"

echo "==> فك الـ ISO"
rm -rf "$WORK/iso" && mkdir -p "$WORK/iso"
7z x -o"$WORK/iso" "$WORK/$ISO_NAME" >/dev/null

echo "==> فك system.sfs"
rm -rf "$WORK/system" && mkdir -p "$WORK/system"
unsquashfs -d "$WORK/system" "$WORK/iso/system.sfs" >/dev/null

echo "==> حذف التطبيقات الافتراضية"
for app in "${REMOVE_APPS[@]}"; do
  rm -rf "$WORK/system/system/app/$app" \
         "$WORK/system/system/priv-app/$app" \
         "$WORK/system/system/product/app/$app" \
         "$WORK/system/system/product/priv-app/$app" 2>/dev/null || true
done

echo "==> تثبيت Kiosk Home كمشغّل وحيد للنظام"
mkdir -p "$WORK/system/system/priv-app/KioskHome"
cp ./kiosk-home.apk "$WORK/system/system/priv-app/KioskHome/KioskHome.apk"

echo "==> ضبط build.prop (ADB عبر الشبكة + تقليل الزوائد)"
BP="$WORK/system/system/build.prop"
cat >> "$BP" <<'EOF'

# --- BM-DROID kiosk ---
service.adb.tcp.port=5555
persist.sys.usb.config=adb
ro.secure=0
ro.debuggable=1
ro.setupwizard.mode=DISABLED
persist.sys.timezone=Africa/Cairo
EOF

echo "==> إعادة ضغط system.sfs"
rm -f "$WORK/iso/system.sfs"
mksquashfs "$WORK/system" "$WORK/iso/system.sfs" -noappend -comp gzip >/dev/null

echo "==> ضبط الإقلاع: بدون قائمة اختيار، وضع هاتف 480x854، بيانات على data.img"
ISOLINUX="$WORK/iso/isolinux/isolinux.cfg"
cat > "$ISOLINUX" <<'EOF'
DEFAULT android
PROMPT 0
TIMEOUT 1
LABEL android
  KERNEL /kernel
  APPEND initrd=/initrd.img root=/dev/ram0 quiet androidboot.selinux=permissive androidboot.hardware=android_x86_64 SRC=/android-9.0-r2 DATA=sda1 UVESA_MODE=480x854
EOF

echo "==> إعادة بناء الـ ISO"
( cd "$WORK/iso" && genisoimage -o "$OUT/android.iso" -V "BMDROID" \
    -b isolinux/isolinux.bin -c isolinux/boot.cat \
    -no-emul-boot -boot-load-size 4 -boot-info-table \
    -R -J -quiet . )

echo "==> إنشاء قرص البيانات data.img (4G, قسم ext4 واحد)"
rm -f "$OUT/data.img"
dd if=/dev/zero of="$OUT/data.img" bs=1M count=4096 status=none
parted -s "$OUT/data.img" mklabel msdos mkpart primary ext4 1MiB 100%
LOOP=$(sudo losetup -f --show -P "$OUT/data.img")
sudo mkfs.ext4 -q "${LOOP}p1"
sudo losetup -d "$LOOP"

echo ""
echo "✓ تم:"
echo "   $OUT/android.iso"
echo "   $OUT/data.img"
echo "الخطوة التالية على ويندوز: runtime/make-runtime.ps1"
