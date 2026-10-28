# Running BM-DROID on Replit

The Replit preview runs the admin panel from `admin-panel/` on port 5000:

```bash
npm --prefix admin-panel install
PORT=5000 npm --prefix admin-panel start
```

Add these values in Replit Secrets before starting the panel:

- `ADMIN_USER` and `ADMIN_PASS`: the dashboard login
- `SUPABASE_URL`: the Supabase project URL
- `SUPABASE_SERVICE_KEY`: the server-side Supabase secret key
- `SUPABASE_ANON_KEY`: the Supabase publishable key
- `SESSION_SECRET`: session signing secret

Apply `supabase/schema.sql` to the Supabase project before using license-management features. `RUNTIME_URL` and `RUNTIME_SHA256` are needed to export Windows executables, but not to open the dashboard.
