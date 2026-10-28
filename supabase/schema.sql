-- ============================================================
-- BM-DROID License Schema
-- نفّذ هذا الملف مرة واحدة في Supabase SQL Editor
-- ============================================================

create table if not exists public.licenses (
  id            uuid primary key default gen_random_uuid(),
  key           text unique not null,          -- مفتاح الترخيص المدمج في الـ EXE
  customer_name text,                          -- اسم العميل
  app_name      text not null,                 -- اسم التطبيق (يظهر في الـ EXE)
  package_name  text not null,                 -- اسم حزمة الـ APK
  duration_days integer not null default 30,   -- مدة الاشتراك بالأيام
  created_at    timestamptz not null default now(),
  activated_at  timestamptz,                   -- أول تفعيل (يبدأ العد من هنا)
  expires_at    timestamptz,                   -- تاريخ الانتهاء = activated_at + duration
  hwid          text,                          -- بصمة الجهاز (ترخيص لجهاز واحد)
  status        text not null default 'active' -- active | revoked
);

-- حماية: ممنوع على anon يقرأ الجدول مباشرة، التفعيل فقط عبر الدالة الآمنة
alter table public.licenses enable row level security;

-- دالة التفعيل/التحقق — يستدعيها الـ EXE بمفتاح anon فقط
create or replace function public.activate_license(p_key text, p_hwid text)
returns json
language plpgsql
security definer
set search_path = public
as $$
declare
  lic record;
begin
  select * into lic from licenses where key = p_key;
  if not found then
    return json_build_object('ok', false, 'error', 'invalid_key');
  end if;

  if lic.status <> 'active' then
    return json_build_object('ok', false, 'error', 'revoked');
  end if;

  -- ترخيص مربوط بجهاز آخر
  if lic.hwid is not null and lic.hwid <> p_hwid then
    return json_build_object('ok', false, 'error', 'hwid_mismatch');
  end if;

  -- أول تفعيل: اربط الجهاز وابدأ العد
  if lic.activated_at is null then
    update licenses
       set activated_at = now(),
           expires_at   = now() + make_interval(days => lic.duration_days),
           hwid         = p_hwid
     where id = lic.id
     returning expires_at into lic.expires_at;
    return json_build_object('ok', true, 'expires_at', lic.expires_at, 'app_name', lic.app_name);
  end if;

  -- انتهت المدة؟
  if lic.expires_at is not null and lic.expires_at < now() then
    return json_build_object('ok', false, 'error', 'expired', 'expires_at', lic.expires_at);
  end if;

  return json_build_object('ok', true, 'expires_at', lic.expires_at, 'app_name', lic.app_name);
end;
$$;

grant execute on function public.activate_license(text, text) to anon;
grant execute on function public.activate_license(text, text) to authenticated;
