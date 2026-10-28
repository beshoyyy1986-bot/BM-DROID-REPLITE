package com.bmdroid.kiosk;

import android.app.Activity;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.os.Bundle;
import android.os.Handler;
import android.view.View;
import android.view.WindowManager;
import android.widget.TextView;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileReader;

/**
 * Kiosk Home — الشاشة الرئيسية الوحيدة في النظام.
 * تقرأ اسم الحزمة المستهدفة من /sdcard/kiosk/target.txt
 * وتفتح التطبيق مباشرة، وتعيد فتحه لو اتقفل.
 */
public class MainActivity extends Activity {

    private final Handler handler = new Handler();
    private TextView status;
    private boolean launched = false;

    private final Runnable watchdog = new Runnable() {
        @Override public void run() {
            launchTarget();
            handler.postDelayed(this, 5000);
        }
    };

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);

        status = new TextView(this);
        status.setTextAlignment(View.TEXT_ALIGNMENT_CENTER);
        status.setTextSize(18);
        status.setText("جارٍ تشغيل التطبيق…");
        setContentView(status);

        handler.post(watchdog);
    }

    private String readTargetPackage() {
        try {
            File f = new File("/sdcard/kiosk/target.txt");
            if (!f.exists()) return null;
            BufferedReader br = new BufferedReader(new FileReader(f));
            String line = br.readLine();
            br.close();
            return line == null ? null : line.trim();
        } catch (Exception e) {
            return null;
        }
    }

    private void launchTarget() {
        String pkg = readTargetPackage();
        if (pkg == null || pkg.isEmpty()) {
            status.setText("في انتظار تجهيز التطبيق…");
            return;
        }
        PackageManager pm = getPackageManager();
        Intent i = pm.getLaunchIntentForPackage(pkg);
        if (i == null) {
            status.setText("جارٍ تثبيت التطبيق…");
            return;
        }
        i.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_REORDER_TO_FRONT);
        try {
            startActivity(i);
            launched = true;
        } catch (Exception ignored) {}
    }

    @Override
    protected void onResume() {
        super.onResume();
        // لو رجعنا لشاشة الـ kiosk يعني التطبيق اتقفل — أعد فتحه
        if (launched) handler.postDelayed(this::launchTarget, 500);
    }

    @Override
    public void onBackPressed() {
        // لا خروج من وضع الـ kiosk
    }
}
