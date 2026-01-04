package cmd

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/audit"
)

func TestAuditCommandNoConfig(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--config", tmpDir, "audit", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestAuditCommandListEmpty(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	setConfigDir(tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--config", tmpDir, "audit", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestAuditCommandListWithEntries(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	setConfigDir(tmpDir)

	logger, err := audit.NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogSwitch("anthropic")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}

	err = logger.LogSwitch("zai")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--config", tmpDir, "audit", "list"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !containsString(output, "anthropic") {
		t.Error("audit list should contain 'anthropic'")
	}
	if !containsString(output, "zai") {
		t.Error("audit list should contain 'zai'")
	}
}

func TestAuditCommandExportCSV(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	setConfigDir(tmpDir)

	logger, err := audit.NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogSwitch("anthropic")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}

	err = logger.LogConfig("zai", "update", []audit.Change{{Field: "api_key", New: "***"}})
	if err != nil {
		t.Fatalf("LogConfig() error = %v", err)
	}

	tmpCSV := filepath.Join(t.TempDir(), "audit.csv")
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--config", tmpDir, "audit", "export", "-o", tmpCSV})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	data, err := os.ReadFile(tmpCSV)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(records) < 2 {
		t.Errorf("CSV should have header + data rows, got %d records", len(records))
	}

	if len(records) > 0 && records[0][0] != "timestamp" {
		t.Errorf("First column should be 'timestamp', got %q", records[0][0])
	}
}

func TestAuditCommandExportJSON(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	setConfigDir(tmpDir)

	logger, err := audit.NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	details := []audit.Change{
		{Field: "api_key", New: "***"},
		{Field: "base_url", New: "https://new.com", Old: "https://old.com"},
	}
	err = logger.LogSwitch("test")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}

	err = logger.LogConfig("zai", "update", details)
	if err != nil {
		t.Fatalf("LogConfig() error = %v", err)
	}

	tmpJSON := filepath.Join(t.TempDir(), "audit.json")
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--config", tmpDir, "audit", "export", "-o", tmpJSON, "-f", "json"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	data, err := os.ReadFile(tmpJSON)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("JSON export should not be empty")
	}

	if !containsString(string(data), "api_key") {
		t.Error("JSON export should contain api_key in details")
	}
	if !containsString(string(data), "base_url") {
		t.Error("JSON export should contain base_url in details")
	}
}

func TestAuditCommandInvalidFormat(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	setConfigDir(tmpDir)

	_, err := audit.NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	tmpOutput := filepath.Join(t.TempDir(), "audit.xml")
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--config", tmpDir, "audit", "export", "-o", tmpOutput, "-f", "xml"})

	err = rootCmd.Execute()
	if err == nil {
		t.Error("export with invalid format should error")
	}
}

func TestAuditCommandMissingOutput(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	setConfigDir(tmpDir)

	_, err := audit.NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--config", tmpDir, "audit", "export"})

	err = rootCmd.Execute()
	if err == nil {
		t.Error("export without --output should error")
	}
}
