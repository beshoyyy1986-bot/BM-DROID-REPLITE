// BM-DROID Builder — يبني EXE واحد محمي: launcher + APK + config مدمجين.
// يُستدعى من لوحة الأدمن: node builder/build.js <job.json>
'use strict';

const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

const ROOT = path.resolve(__dirname, '..');
const LAUNCHER_DIR = path.join(ROOT, 'launcher');
const DIST_DIR = path.join(ROOT, 'dist');

function goBinary() {
  if (process.env.GO_PATH) return process.env.GO_PATH;
  const portable = path.join(ROOT, 'tools', 'go', 'bin', 'go.exe');
  if (fs.existsSync(portable)) return portable;
  return 'go'; // يفترض وجوده في PATH
}

async function build(job) {
  const { appName, packageName, licenseKey, apkPath, supabaseUrl, supabaseAnonKey, runtimeUrl, runtimeSha256 } = job;

  if (!fs.existsSync(apkPath)) throw new Error('ملف APK غير موجود: ' + apkPath);

  // 1) كتابة الأصول المدمجة داخل نسخة العمل من الـ launcher
  const assetsDir = path.join(LAUNCHER_DIR, 'assets');
  fs.mkdirSync(assetsDir, { recursive: true });
  fs.copyFileSync(apkPath, path.join(assetsDir, 'app.apk'));
  fs.writeFileSync(path.join(assetsDir, 'config.json'), JSON.stringify({
    app_name: appName,
    package_name: packageName,
    license_key: licenseKey,
    supabase_url: supabaseUrl,
    supabase_anon_key: supabaseAnonKey,
    runtime_url: runtimeUrl,
    runtime_sha256: runtimeSha256 || ''
  }, null, 2));

  // 2) التجميع
  fs.mkdirSync(DIST_DIR, { recursive: true });
  const safe = appName.replace(/[^\w؀-ۿ-]+/g, '_');
  const out = path.join(DIST_DIR, `${safe}-Setup.exe`);

  console.log('[build] go build ...');
  execFileSync(goBinary(), ['build', '-ldflags', '-s -w', '-o', out, '.'], {
    cwd: LAUNCHER_DIR,
    stdio: 'inherit',
    env: { ...process.env, GOOS: 'windows', GOARCH: 'amd64', CGO_ENABLED: '0' }
  });

  const sizeMB = (fs.statSync(out).size / 1048576).toFixed(1);
  console.log(`[build] تم: ${out} (${sizeMB} MB)`);
  return out;
}

// تشغيل مستقل: node build.js job.json
if (require.main === module) {
  const job = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
  build(job)
    .then(out => { console.log(out); process.exit(0); })
    .catch(err => { console.error('[build] فشل:', err.message); process.exit(1); });
}

module.exports = { build };
