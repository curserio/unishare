package config

import "testing"

func TestLoadDefaultsAndEnv(t *testing.T) {
	t.Setenv("UNISHARE_ADDR", "127.0.0.1:18080")
	t.Setenv("UNISHARE_DATA_DIR", "/tmp/unishare")
	t.Setenv("UNISHARE_STATIC_DIR", "public")
	t.Setenv("UNISHARE_PUBLIC_BASE_URL", "https://share.example.com/")
	t.Setenv("UNISHARE_USERS", "main:secret,mom:another-secret")
	t.Setenv("UNISHARE_COOKIE_SECURE", "true")
	t.Setenv("UNISHARE_MAX_UPLOAD_MB", "12")
	t.Setenv("UNISHARE_BASE_PATH", "/unishare/")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Addr != "127.0.0.1:18080" {
		t.Fatalf("unexpected addr: %s", cfg.Addr)
	}
	if cfg.PublicBaseURL != "https://share.example.com" {
		t.Fatalf("unexpected public base url: %s", cfg.PublicBaseURL)
	}
	if cfg.StaticDir != "public" {
		t.Fatalf("unexpected static dir: %s", cfg.StaticDir)
	}
	if cfg.CookieSecure != "true" {
		t.Fatalf("unexpected cookie secure mode: %s", cfg.CookieSecure)
	}
	if cfg.MaxUploadMB != 12 || cfg.MaxUploadBytes != 12<<20 {
		t.Fatalf("unexpected upload limit: %d MB / %d bytes", cfg.MaxUploadMB, cfg.MaxUploadBytes)
	}
	if cfg.BasePath != "/unishare" {
		t.Fatalf("unexpected base path: %s", cfg.BasePath)
	}
	if len(cfg.Users) != 2 || cfg.Users[0].ID != "main" || cfg.Users[0].Token != "secret" {
		t.Fatalf("users were not loaded: %+v", cfg.Users)
	}
}

func TestParseUsersFallbackToken(t *testing.T) {
	users, err := ParseUsers("", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].ID != "default" || users[0].Token != "secret" {
		t.Fatalf("unexpected fallback users: %+v", users)
	}
}

func TestParseUsersRejectsInvalidConfig(t *testing.T) {
	for _, raw := range []string{"bad", "bad id:token", "one:", "one:a,one:b"} {
		if _, err := ParseUsers(raw, ""); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

func TestNormalizeBasePath(t *testing.T) {
	tests := map[string]string{
		"":             "",
		"/":            "",
		"/unishare":    "/unishare",
		"/unishare/":   "/unishare",
		"/apps//share": "",
		"unishare":     "",
	}
	for input, want := range tests {
		got, err := NormalizeBasePath(input)
		if want == "" && input != "" && input != "/" {
			if err == nil {
				t.Fatalf("expected error for %q", input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizeBasePath(%q) = %q, want %q", input, got, want)
		}
	}
}
