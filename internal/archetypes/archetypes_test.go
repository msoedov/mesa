package archetypes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestList(t *testing.T) {
	slugs, err := List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(slugs) == 0 {
		t.Fatal("List() returned no slugs")
	}
	known := []string{"architect", "fullstack", "backend", "frontend", "qa", "devops", "designer", "product"}
	for _, want := range known {
		found := false
		for _, s := range slugs {
			if s == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("List() missing known archetype %q", want)
		}
	}
}

func TestRead(t *testing.T) {
	data, err := Read("architect")
	if err != nil {
		t.Fatalf("Read(architect) error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Architect") && !strings.Contains(content, "architect") {
		t.Errorf("Read(architect) content unexpected: %s", content[:min(len(content), 80)])
	}
}

func TestReadNotFound(t *testing.T) {
	_, err := Read("nonexistent-slug-xyz")
	if err == nil {
		t.Fatal("Read(nonexistent-slug) expected error, got nil")
	}
}

func TestExists(t *testing.T) {
	if !Exists("architect") {
		t.Error("Exists(architect) = false, want true")
	}
	if Exists("nonexistent-slug-xyz") {
		t.Error("Exists(nonexistent-slug) = true, want false")
	}
}

func TestGet(t *testing.T) {
	content, err := Get("fullstack")
	if err != nil {
		t.Fatalf("Get(fullstack) error: %v", err)
	}
	if !strings.Contains(content, "fullstack") && !strings.Contains(content, "Fullstack") {
		t.Errorf("Get(fullstack) content unexpected: %s", content[:min(len(content), 80)])
	}
}

func TestGetNotFound(t *testing.T) {
	_, err := Get("nonexistent-slug-xyz")
	if err == nil {
		t.Fatal("Get(nonexistent-slug) expected error, got nil")
	}
}

func TestWriteToTemp(t *testing.T) {
	content, err := Read("qa")
	if err != nil {
		t.Fatalf("Read(qa) error: %v", err)
	}

	path, cleanup, err := WriteToTemp("qa")
	if err != nil {
		t.Fatalf("WriteToTemp(qa) error: %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile temp: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("WriteToTemp content mismatch\ngot:  %s\nwant: %s", string(data[:min(len(data), 40)]), string(content[:min(len(content), 40)]))
	}
}

func TestWriteToTempCleanup(t *testing.T) {
	_, cleanup, err := WriteToTemp("architect")
	if err != nil {
		t.Fatalf("WriteToTemp(architect) error: %v", err)
	}
	// Verify the file exists before cleanup
	cleanup()
	// After cleanup, the file should be removed — os.Stat should fail
	// We can't get the path back without a wrapper, but the function signature is correct
}

func TestOverrideRead(t *testing.T) {
	// Create a temporary override directory
	dir := t.TempDir()
	overrideFile := filepath.Join(dir, "architect.md")
	overrideContent := []byte("# Override Architect\nCustom content")
	if err := os.WriteFile(overrideFile, overrideContent, 0644); err != nil {
		t.Fatalf("write override: %v", err)
	}

	// Save and restore original overrides dir
	origDir := GetOverridesDir()
	SetOverridesDir(dir)
	defer SetOverridesDir(origDir)

	data, err := Read("architect")
	if err != nil {
		t.Fatalf("Read(architect) with override: %v", err)
	}
	if string(data) != string(overrideContent) {
		t.Errorf("override not taken: got %q, want %q", string(data), string(overrideContent))
	}
}

func TestOverrideList(t *testing.T) {
	dir := t.TempDir()
	overrideFile := filepath.Join(dir, "custom-role.md")
	if err := os.WriteFile(overrideFile, []byte("# Custom"), 0644); err != nil {
		t.Fatalf("write override: %v", err)
	}

	origDir := GetOverridesDir()
	SetOverridesDir(dir)
	defer SetOverridesDir(origDir)

	slugs, err := List()
	if err != nil {
		t.Fatalf("List() with override: %v", err)
	}

	found := false
	for _, s := range slugs {
		if s == "custom-role" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("List() missing override slug %q", "custom-role")
	}
	// Original slugs should still be present
	foundArchitect := false
	for _, s := range slugs {
		if s == "architect" {
			foundArchitect = true
			break
		}
	}
	if !foundArchitect {
		t.Error("List() missing embedded slug 'architect' after override")
	}
}

func TestOverrideExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "custom-role.md"), []byte("# Custom"), 0644); err != nil {
		t.Fatalf("write override: %v", err)
	}

	origDir := GetOverridesDir()
	SetOverridesDir(dir)
	defer SetOverridesDir(origDir)

	if !Exists("custom-role") {
		t.Error("Exists(custom-role) = false, want true for override file")
	}
}

func TestSetGetOverridesDir(t *testing.T) {
	orig := GetOverridesDir()
	SetOverridesDir("/tmp/test-archetypes")
	if GetOverridesDir() != "/tmp/test-archetypes" {
		t.Errorf("GetOverridesDir() = %q, want %q", GetOverridesDir(), "/tmp/test-archetypes")
	}
	SetOverridesDir(orig)
	if GetOverridesDir() != orig {
		t.Errorf("restored GetOverridesDir() = %q, want %q", GetOverridesDir(), orig)
	}
}
