package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type licenseResult struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error"`
	ExpiresAt string `json:"expires_at"`
	AppName   string `json:"app_name"`
}

// activateLicense يستدعي دالة activate_license في Supabase (RPC آمن بمفتاح anon).
func activateLicense(cfg *Config, hwid string) (*licenseResult, error) {
	body, _ := json.Marshal(map[string]string{
		"p_key":  cfg.LicenseKey,
		"p_hwid": hwid,
	})

	url := cfg.SupabaseURL + "/rest/v1/rpc/activate_license"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", cfg.SupabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+cfg.SupabaseAnonKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}

	var res licenseResult
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("رد غير مفهوم من الخادم: %s", string(data))
	}
	return &res, nil
}
