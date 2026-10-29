@echo off
chcp 65001 >nul
title BM-DROID - لوحة الأدمن
cd /d "%~dp0admin-panel"

echo ===============================================
echo   BM-DROID - لوحة تحكم تصدير التطبيقات
echo ===============================================
echo.

where node >nul 2>nul
if errorlevel 1 (
    echo [خطأ] Node.js مش مثبت على الجهاز
    echo نزّله من: https://nodejs.org
    pause
    exit /b 1
)

if not exist ".env" (
    echo [!] ملف الإعدادات .env مش موجود
    echo     هيفتحلك دلوقتي — املأ القيم واحفظ ثم شغّل الملف تاني
    copy .env.example .env >nul
    notepad .env
    pause
    exit /b 0
)

if not exist "node_modules" (
    echo [*] أول تشغيل - تثبيت الحزم...
    call npm install --no-audit --no-fund
    echo.
)

echo [*] تشغيل اللوحة...
start "" http://localhost:3000
node server.js
pause
