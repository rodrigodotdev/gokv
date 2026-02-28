package config

import (
	"flag"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Unset all GOKV env vars to ensure defaults.
	for _, key := range []string{"GOKV_PORT", "GOKV_AOF_ENABLED", "GOKV_AOF_FILE_PATH", "GOKV_CLEANUP_INTERVAL_MS", "GOKV_CONN_TIMEOUT_SEC"} {
		t.Setenv(key, "")
	}

	cfg := Load()

	if cfg.Port != DefaultPort {
		t.Errorf("Port = %q, want %q", cfg.Port, DefaultPort)
	}
	if cfg.AOFEnabled {
		t.Error("AOFEnabled = true, want false")
	}
	if cfg.AOFFilePath != DefaultAOFFilePath {
		t.Errorf("AOFFilePath = %q, want %q", cfg.AOFFilePath, DefaultAOFFilePath)
	}
	if cfg.CleanupIntervalMs != DefaultCleanupInterval {
		t.Errorf("CleanupIntervalMs = %d, want %d", cfg.CleanupIntervalMs, DefaultCleanupInterval)
	}
	if cfg.ConnTimeoutSec != DefaultConnTimeoutSec {
		t.Errorf("ConnTimeoutSec = %d, want %d", cfg.ConnTimeoutSec, DefaultConnTimeoutSec)
	}
}

func TestLoad_EnvVars(t *testing.T) {
	t.Setenv("GOKV_PORT", "9999")
	t.Setenv("GOKV_AOF_ENABLED", "true")
	t.Setenv("GOKV_AOF_FILE_PATH", "/tmp/test.aof")
	t.Setenv("GOKV_CLEANUP_INTERVAL_MS", "500")
	t.Setenv("GOKV_CONN_TIMEOUT_SEC", "120")

	cfg := Load()

	if cfg.Port != "9999" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9999")
	}
	if !cfg.AOFEnabled {
		t.Error("AOFEnabled = false, want true")
	}
	if cfg.AOFFilePath != "/tmp/test.aof" {
		t.Errorf("AOFFilePath = %q, want %q", cfg.AOFFilePath, "/tmp/test.aof")
	}
	if cfg.CleanupIntervalMs != 500 {
		t.Errorf("CleanupIntervalMs = %d, want %d", cfg.CleanupIntervalMs, 500)
	}
	if cfg.ConnTimeoutSec != 120 {
		t.Errorf("ConnTimeoutSec = %d, want %d", cfg.ConnTimeoutSec, 120)
	}
}

func TestLoad_AOFEnabled_One(t *testing.T) {
	t.Setenv("GOKV_AOF_ENABLED", "1")

	cfg := Load()
	if !cfg.AOFEnabled {
		t.Error("AOFEnabled = false, want true for GOKV_AOF_ENABLED=1")
	}
}

func TestLoad_InvalidCleanupInterval(t *testing.T) {
	t.Setenv("GOKV_CLEANUP_INTERVAL_MS", "notanumber")

	cfg := Load()
	if cfg.CleanupIntervalMs != DefaultCleanupInterval {
		t.Errorf("CleanupIntervalMs = %d, want default %d for invalid env", cfg.CleanupIntervalMs, DefaultCleanupInterval)
	}
}

func TestLoad_NegativeCleanupInterval(t *testing.T) {
	t.Setenv("GOKV_CLEANUP_INTERVAL_MS", "-5")

	cfg := Load()
	if cfg.CleanupIntervalMs != DefaultCleanupInterval {
		t.Errorf("CleanupIntervalMs = %d, want default %d for negative env", cfg.CleanupIntervalMs, DefaultCleanupInterval)
	}
}

func TestLoad_InvalidConnTimeout(t *testing.T) {
	t.Setenv("GOKV_CONN_TIMEOUT_SEC", "notanumber")

	cfg := Load()
	if cfg.ConnTimeoutSec != DefaultConnTimeoutSec {
		t.Errorf("ConnTimeoutSec = %d, want default %d for invalid env", cfg.ConnTimeoutSec, DefaultConnTimeoutSec)
	}
}

func TestLoad_NegativeConnTimeout(t *testing.T) {
	t.Setenv("GOKV_CONN_TIMEOUT_SEC", "-10")

	cfg := Load()
	if cfg.ConnTimeoutSec != DefaultConnTimeoutSec {
		t.Errorf("ConnTimeoutSec = %d, want default %d for negative env", cfg.ConnTimeoutSec, DefaultConnTimeoutSec)
	}
}

func TestRegisterFlags_Overrides(t *testing.T) {
	t.Setenv("GOKV_PORT", "")
	t.Setenv("GOKV_AOF_ENABLED", "")
	t.Setenv("GOKV_AOF_FILE_PATH", "")
	t.Setenv("GOKV_CLEANUP_INTERVAL_MS", "")
	t.Setenv("GOKV_CONN_TIMEOUT_SEC", "")

	cfg := Load()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.RegisterFlags(fs)

	err := fs.Parse([]string{"-port", "8080", "-aof", "-aof-file", "custom.aof", "-cleanup-interval", "2000", "-conn-timeout", "600"})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if !cfg.AOFEnabled {
		t.Error("AOFEnabled = false, want true")
	}
	if cfg.AOFFilePath != "custom.aof" {
		t.Errorf("AOFFilePath = %q, want %q", cfg.AOFFilePath, "custom.aof")
	}
	if cfg.CleanupIntervalMs != 2000 {
		t.Errorf("CleanupIntervalMs = %d, want %d", cfg.CleanupIntervalMs, 2000)
	}
	if cfg.ConnTimeoutSec != 600 {
		t.Errorf("ConnTimeoutSec = %d, want %d", cfg.ConnTimeoutSec, 600)
	}
}

func TestRegisterFlags_EnvThenFlag(t *testing.T) {
	// Env sets port to 9000, flag overrides to 7777.
	t.Setenv("GOKV_PORT", "9000")

	cfg := Load()
	if cfg.Port != "9000" {
		t.Fatalf("Port after Load = %q, want %q", cfg.Port, "9000")
	}

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.RegisterFlags(fs)

	err := fs.Parse([]string{"-port", "7777"})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if cfg.Port != "7777" {
		t.Errorf("Port = %q, want %q after flag override", cfg.Port, "7777")
	}
}
