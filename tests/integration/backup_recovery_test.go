package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

func TestBackupRestoreCycle(t *testing.T) {
	// Get the project root by finding go.mod
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}

	// Build kairo
	kairoBinary := filepath.Join(projectRoot, "kairo_test")
	if runtime.GOOS == "windows" {
		kairoBinary += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", kairoBinary, ".")
	cmd.Dir = projectRoot
	if err := cmd.Run(); err != nil {
		t.Fatalf("build failed: %v", err)
	}
	defer os.Remove(kairoBinary)

	tmpDir := t.TempDir()

	// Setup a test configuration programmatically
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		Providers: map[string]config.Provider{
			"anthropic": {
				Name:    "Native Anthropic",
				BaseURL: "",
				Model:   "",
			},
			"zai": {
				Name:    "Z.AI",
				BaseURL: "https://api.z.ai/api/anthropic",
				Model:   "glm-4.7",
			},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create encryption key
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Create encrypted secrets
	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secretsContent := "ZAI_API_KEY=test-api-key-12345\nANTHROPIC_API_KEY=sk-ant-api-key-12345\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("failed to encrypt secrets: %v", err)
	}

	// Verify initial setup
	configPath := filepath.Join(tmpDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config not created")
	}

	// Create backup using CLI
	backupCmd := exec.Command(kairoBinary, "--config", tmpDir, "backup")
	backupOutput, _ := backupCmd.CombinedOutput()

	if !strings.Contains(string(backupOutput), "Backup created") {
		t.Errorf("backup command failed: %s", string(backupOutput))
	}

	// Remove original files
	os.Remove(keyPath)
	os.Remove(secretsPath)
	os.Remove(configPath)

	// Find backup file
	backupsDir := filepath.Join(tmpDir, "backups")
	backups, err := os.ReadDir(backupsDir)
	if err != nil {
		t.Fatalf("failed to read backups dir: %v", err)
	}
	if len(backups) == 0 {
		t.Fatal("no backup file created")
	}

	backupPath := filepath.Join(backupsDir, backups[0].Name())

	// Restore from backup using CLI
	restoreCmd := exec.Command(kairoBinary, "--config", tmpDir, "restore", backupPath)
	restoreOutput, _ := restoreCmd.CombinedOutput()

	if !strings.Contains(string(restoreOutput), "Backup restored") {
		t.Errorf("restore command failed: %s", string(restoreOutput))
	}

	// Verify all files restored
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("age.key not restored")
	}

	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		t.Error("secrets.age not restored")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config.yaml not restored")
	}

	// Verify restored secrets can be decrypted
	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Errorf("failed to decrypt restored secrets: %v", err)
	}
	if !strings.Contains(decrypted, "test-api-key-12345") {
		t.Error("restored secrets do not contain expected API key")
	}

	// Verify restored config
	restoredCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Errorf("failed to load restored config: %v", err)
	}
	if len(restoredCfg.Providers) != 2 {
		t.Errorf("expected 2 providers in restored config, got %d", len(restoredCfg.Providers))
	}
	if restoredCfg.DefaultProvider != "anthropic" {
		t.Errorf("expected default provider 'anthropic', got '%s'", restoredCfg.DefaultProvider)
	}
}

func findProjectRoot() (string, error) {
	// Start from the current working directory and search upward for go.mod
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the filesystem root without finding go.mod
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
