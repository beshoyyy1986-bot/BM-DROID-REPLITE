// لوحة أدمن BM-DROID — تسجيل دخول، رفع APK، إصدار ترخيص، تصدير EXE.
'use strict';

require('dotenv').config({ path: require('path').join(__dirname, '.env') });

const path = require('path');
const fs = require('fs');
const crypto = require('crypto');
const express = require('express');
const session = require('express-session');
const multer = require('multer');
const { createClient } = require('@supabase/supabase-js');
const { build } = require('../builder/build');

const PORT = process.env.PORT || 3000;
const SUPABASE_URL = process.env.SUPABASE_URL;
const SUPABASE_SERVICE_KEY = process.env.SUPABASE_SERVICE_KEY;
const SUPABASE_ANON_KEY = process.env.SUPABASE_ANON_KEY;
const RUNTIME_URL = process.env.RUNTIME_URL || '';
const RUNTIME_SHA256 = process.env.RUNTIME_SHA256 || '';

if (!SUPABASE_URL || !SUPABASE_SERVICE_KEY || !SUPABASE_ANON_KEY) {
  console.error('✗ راجع ملف admin-panel/.env — انسخ .env.example واملأ القيم');
  process.exit(1);
}

const supabase = createClient(SUPABASE_URL, SUPABASE_SERVICE_KEY);
const app = express();

app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use(session({
  secret: process.env.SESSION_SECRET || crypto.randomBytes(32).toString('hex'),
  resave: false,
  saveUninitialized: false,
  cookie: { maxAge: 12 * 60 * 60 * 1000 }
}));

const UPLOADS = path.join(__dirname, 'uploads');
fs.mkdirSync(UPLOADS, { recursive: true });
const upload = multer({
  dest: UPLOADS,
  limits: { fileSize: 500 * 1024 * 1024 },
  fileFilter: (req, file, cb) => {
    if (file.originalname.toLowerCase().endsWith('.apk')) cb(null, true);
    else cb(new Error('الملف يجب أن يكون APK'));
  }
});

// ---------- المصادقة ----------
function requireAuth(req, res, next) {
  if (req.session && req.session.admin) return next();
  res.status(401).json({ error: 'غير مسجل الدخول' });
}

app.post('/api/login', (req, res) => {
  const { username, password } = req.body || {};
  if (username === process.env.ADMIN_USER && password === process.env.ADMIN_PASS) {
    req.session.admin = true;
    return res.json({ ok: true });
  }
  res.status(401).json({ error: 'بيانات الدخول غير صحيحة' });
});

app.post('/api/logout', (req, res) => req.session.destroy(() => res.json({ ok: true })));

// ---------- التصدير ----------
app.post('/api/export', requireAuth, upload.single('apk'), async (req, res) => {
  try {
    const { appName, packageName, customerName, durationDays } = req.body;
    if (!req.file) return res.status(400).json({ error: 'ارفع ملف APK' });
    if (!appName || !packageName) return res.status(400).json({ error: 'اسم التطبيق واسم الحزمة مطلوبان' });

    // 1) إنشاء مفتاح ترخيص
    const key = 'BM-' + crypto.randomBytes(4).toString('hex').toUpperCase() + '-' +
                crypto.randomBytes(4).toString('hex').toUpperCase();

    // 2) تسجيل الترخيص في Supabase
    const { error: dbErr } = await supabase.from('licenses').insert({
      key,
      customer_name: customerName || null,
      app_name: appName,
      package_name: packageName,
      duration_days: parseInt(durationDays, 10) || 30
    });
    if (dbErr) return res.status(500).json({ error: 'فشل حفظ الترخيص: ' + dbErr.message });

    // 3) بناء الـ EXE (الـ APK مدمج بداخله)
    const exePath = await build({
      appName, packageName, licenseKey: key, apkPath: req.file.path,
      supabaseUrl: SUPABASE_URL, supabaseAnonKey: SUPABASE_ANON_KEY,
      runtimeUrl: RUNTIME_URL, runtimeSha256: RUNTIME_SHA256
    });

    fs.unlink(req.file.path, () => {}); // تنظيف الرفع المؤقت

    res.json({
      ok: true,
      licenseKey: key,
      file: '/api/download/' + encodeURIComponent(path.basename(exePath))
    });
  } catch (e) {
    console.error(e);
    res.status(500).json({ error: e.message });
  }
});

// ---------- التراخيص ----------
app.get('/api/licenses', requireAuth, async (req, res) => {
  const { data, error } = await supabase
    .from('licenses')
    .select('*')
    .order('created_at', { ascending: false });
  if (error) return res.status(500).json({ error: error.message });
  res.json(data);
});

app.post('/api/licenses/:id/revoke', requireAuth, async (req, res) => {
  const { error } = await supabase.from('licenses').update({ status: 'revoked' }).eq('id', req.params.id);
  if (error) return res.status(500).json({ error: error.message });
  res.json({ ok: true });
});

app.post('/api/licenses/:id/restore', requireAuth, async (req, res) => {
  const { error } = await supabase.from('licenses').update({ status: 'active' }).eq('id', req.params.id);
  if (error) return res.status(500).json({ error: error.message });
  res.json({ ok: true });
});

// فك ربط الجهاز — يسمح بنقل الترخيص لجهاز آخر
app.post('/api/licenses/:id/reset-hwid', requireAuth, async (req, res) => {
  const { error } = await supabase.from('licenses').update({ hwid: null }).eq('id', req.params.id);
  if (error) return res.status(500).json({ error: error.message });
  res.json({ ok: true });
});

app.delete('/api/licenses/:id', requireAuth, async (req, res) => {
  const { error } = await supabase.from('licenses').delete().eq('id', req.params.id);
  if (error) return res.status(500).json({ error: error.message });
  res.json({ ok: true });
});

// ---------- التنزيل ----------
app.get('/api/download/:file', requireAuth, (req, res) => {
  const file = path.join(__dirname, '..', 'dist', path.basename(req.params.file));
  if (!fs.existsSync(file)) return res.status(404).json({ error: 'الملف غير موجود' });
  res.download(file);
});

app.use(express.static(path.join(__dirname, 'public')));

app.listen(PORT, () => console.log(`✓ لوحة الأدمن تعمل: http://localhost:${PORT}`));
