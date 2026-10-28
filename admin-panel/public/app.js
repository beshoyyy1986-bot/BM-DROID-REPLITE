/* لوحة أدمن BM-DROID */
'use strict';

const $ = id => document.getElementById(id);

async function api(path, opts = {}) {
  const res = await fetch(path, opts);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || ('HTTP ' + res.status));
  return data;
}

// ---------- دخول ----------
async function tryShowDash() {
  try {
    await api('/api/licenses');
    $('loginView').classList.add('hidden');
    $('dashView').classList.remove('hidden');
    loadLicenses();
  } catch {
    $('loginView').classList.remove('hidden');
  }
}

$('loginBtn').onclick = async () => {
  $('loginErr').textContent = '';
  try {
    await api('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: $('loginUser').value, password: $('loginPass').value })
    });
    tryShowDash();
  } catch (e) {
    $('loginErr').textContent = e.message;
  }
};
$('loginPass').addEventListener('keydown', e => { if (e.key === 'Enter') $('loginBtn').click(); });

$('logoutBtn').onclick = async () => {
  await api('/api/logout', { method: 'POST' });
  location.reload();
};

// ---------- تبويبات ----------
document.querySelectorAll('nav .tab').forEach(btn => {
  btn.onclick = () => {
    document.querySelectorAll('nav .tab').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    document.querySelectorAll('.tabpane').forEach(p => p.classList.add('hidden'));
    $('tab-' + btn.dataset.tab).classList.remove('hidden');
    if (btn.dataset.tab === 'licenses') loadLicenses();
  };
});

// ---------- تصدير ----------
$('exportBtn').onclick = async () => {
  const file = $('apkFile').files[0];
  const status = $('exportStatus');
  const result = $('exportResult');
  result.classList.add('hidden');

  if (!file) { status.textContent = '⚠ اختر ملف APK أولًا'; return; }
  if (!$('appName').value.trim() || !$('packageName').value.trim()) {
    status.textContent = '⚠ اسم التطبيق واسم الحزمة مطلوبان'; return;
  }

  const fd = new FormData();
  fd.append('apk', file);
  fd.append('appName', $('appName').value.trim());
  fd.append('packageName', $('packageName').value.trim());
  fd.append('customerName', $('customerName').value.trim());
  fd.append('durationDays', $('durationDays').value || '30');

  $('exportBtn').disabled = true;
  status.textContent = '⏳ جارٍ البناء... قد يستغرق دقيقة (يتم دمج الـ APK داخل الـ EXE)';

  try {
    const res = await fetch('/api/export', { method: 'POST', body: fd });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'فشل التصدير');

    status.textContent = '';
    result.classList.remove('hidden');
    result.innerHTML =
      `<b>✓ تم التصدير بنجاح</b><br><br>` +
      `مفتاح الترخيص: <code>${data.licenseKey}</code><br><br>` +
      `<a href="${data.file}">⬇ تحميل ملف التثبيت (EXE)</a><br><br>` +
      `<span class="muted">سلّم ملف الـ EXE للعميل — التطبيق مدمج بداخله والترخيص يُتحقق أونلاين.</span>`;
    loadLicenses();
  } catch (e) {
    status.textContent = '✗ ' + e.message;
  } finally {
    $('exportBtn').disabled = false;
  }
};

// ---------- التراخيص ----------
async function loadLicenses() {
  const body = $('licBody');
  try {
    const list = await api('/api/licenses');
    body.innerHTML = '';
    if (!list.length) {
      body.innerHTML = '<tr><td colspan="8" class="muted">لا توجد تراخيص بعد</td></tr>';
      return;
    }
    for (const lic of list) {
      const tr = document.createElement('tr');
      const fmt = d => d ? new Date(d).toLocaleDateString('ar-EG') : '—';
      const expired = lic.expires_at && new Date(lic.expires_at) < new Date();
      tr.innerHTML =
        `<td class="key">${lic.key}</td>` +
        `<td>${lic.customer_name || '—'}</td>` +
        `<td>${lic.app_name}</td>` +
        `<td>${lic.duration_days} يوم</td>` +
        `<td>${fmt(lic.activated_at)}</td>` +
        `<td>${fmt(lic.expires_at)}${expired ? ' ⛔' : ''}</td>` +
        `<td><span class="badge ${lic.status}">${lic.status === 'active' ? 'نشط' : 'ملغى'}</span></td>` +
        `<td></td>`;

      const actions = tr.lastElementChild;
      const mk = (label, fn, title) => {
        const b = document.createElement('button');
        b.className = 'small'; b.textContent = label; b.title = title || '';
        b.onclick = fn; actions.appendChild(b);
      };
      if (lic.status === 'active') mk('إلغاء', () => licAction(lic.id, 'revoke'), 'إلغاء الترخيص');
      else mk('تفعيل', () => licAction(lic.id, 'restore'));
      mk('فك الجهاز', () => licAction(lic.id, 'reset-hwid'), 'السماح بنقل الترخيص لجهاز آخر');
      mk('حذف', () => { if (confirm('حذف الترخيص نهائيًا؟')) licAction(lic.id, '', 'DELETE'); });
      body.appendChild(tr);
    }
  } catch (e) {
    body.innerHTML = `<tr><td colspan="8" class="error">${e.message}</td></tr>`;
  }
}

async function licAction(id, action, method) {
  await api(`/api/licenses/${id}${action ? '/' + action : ''}`, { method: method || 'POST' });
  loadLicenses();
}

$('refreshLic').onclick = loadLicenses;

tryShowDash();
