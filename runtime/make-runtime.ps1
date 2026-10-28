# ============================================================
# تجهيز حزمة الرن-تايم runtime.zip (تُنفَّذ مرة واحدة على جهازك)
# المحتوى: QEMU portable + android.iso + data.img + adb
# الناتج: runtime.zip + SHA256 — ارفعه وضع رابطه في admin-panel/.env
#
# المتطلبات: 7-Zip مثبت، وملفات android-image/out/ جاهزة
# ============================================================
$ErrorActionPreference = 'Stop'
$Root   = Split-Path $PSScriptRoot -Parent
$Out    = Join-Path $Root 'runtime\out'
$Bundle = Join-Path $Root 'runtime\bundle'

# --- غيّر لأحدث إصدار من https://qemu.weilnetz.de/w64/ ---
$QemuUrl = 'https://qemu.weilnetz.de/w64/qemu-w64-setup-20241231.exe'

Write-Host '==> تنزيل QEMU' -ForegroundColor Cyan
$qemuSetup = Join-Path $env:TEMP 'qemu-setup.exe'
if (-not (Test-Path $qemuSetup)) { Invoke-WebRequest $QemuUrl -OutFile $qemuSetup }

Write-Host '==> استخراج QEMU portable (7-Zip)' -ForegroundColor Cyan
New-Item -ItemType Directory -Force $Bundle | Out-Null
& 'C:\Program Files\7-Zip\7z.exe' x $qemuSetup -o"$Bundle\qemu" -y | Out-Null
# الاحتفاظ بالضروري فقط لتقليل الحجم
Get-ChildItem "$Bundle\qemu" -File | Where-Object {
    $_.Name -notmatch '^(qemu-system-x86_64|qemu-img)|\.dll$'
} | Remove-Item -Force

Write-Host '==> تنزيل adb (platform-tools)' -ForegroundColor Cyan
$pt = Join-Path $env:TEMP 'platform-tools.zip'
if (-not (Test-Path $pt)) {
    Invoke-WebRequest 'https://dl.google.com/android/repository/platform-tools-latest-windows.zip' -OutFile $pt
}
& 'C:\Program Files\7-Zip\7z.exe' x $pt -o"$env:TEMP\pt" -y | Out-Null
New-Item -ItemType Directory -Force "$Bundle\adb" | Out-Null
Copy-Item "$env:TEMP\pt\platform-tools\adb.exe",
          "$env:TEMP\pt\platform-tools\AdbWinApi.dll",
          "$env:TEMP\pt\platform-tools\AdbWinUsbApi.dll" "$Bundle\adb\"

Write-Host '==> نسخ صورة أندرويد المصغرة' -ForegroundColor Cyan
Copy-Item "$Root\android-image\out\android.iso" "$Bundle\android.iso"
Copy-Item "$Root\android-image\out\data.img"   "$Bundle\data.img"

Write-Host '==> ضغط runtime.zip' -ForegroundColor Cyan
New-Item -ItemType Directory -Force $Out | Out-Null
$zip = Join-Path $Out 'runtime.zip'
if (Test-Path $zip) { Remove-Item $zip }
Compress-Archive -Path "$Bundle\*" -DestinationPath $zip -CompressionLevel Optimal

$hash = (Get-FileHash $zip -Algorithm SHA256).Hash.ToLower()
Write-Host ""
Write-Host "✓ تم إنشاء: $zip" -ForegroundColor Green
Write-Host "✓ الحجم: $([math]::Round((Get-Item $zip).Length/1MB)) MB"
Write-Host ""
Write-Host 'ضع في admin-panel/.env:' -ForegroundColor Yellow
Write-Host "RUNTIME_SHA256=$hash"
Write-Host 'RUNTIME_URL=<ارفع runtime.zip على Supabase Storage أو أي استضافة وضع الرابط المباشر>'
