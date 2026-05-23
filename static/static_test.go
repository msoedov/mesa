package static

import (
	"io/fs"
	"testing"
)

func TestFSEmbedded(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatalf("ReadDir embedded FS: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("embedded FS has no entries")
	}
}

func TestFSFilesExist(t *testing.T) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	found := map[string]bool{}
	for _, e := range entries {
		found[e.Name()] = true
	}

	required := []string{"static.go", "favicon.svg", "demo.gif"}
	for _, name := range required {
		if !found[name] {
			t.Errorf("embedded FS missing file %q", name)
		}
	}
}

func TestFSReadFile(t *testing.T) {
	data, err := FS.ReadFile("favicon.svg")
	if err != nil {
		t.Fatalf("ReadFile(favicon.svg): %v", err)
	}
	if len(data) == 0 {
		t.Error("favicon.svg is empty")
	}
	content := string(data)
	if content[:4] != "<svg" && content[:4] != "<?xm" {
		t.Errorf("favicon.svg doesn't look like SVG, starts with: %q", content[:min(len(content), 40)])
	}
}

func TestFSReadStaticGo(t *testing.T) {
	data, err := FS.ReadFile("static.go")
	if err != nil {
		t.Fatalf("ReadFile(static.go): %v", err)
	}
	if len(data) == 0 {
		t.Error("static.go is empty")
	}
	content := string(data)
	if content[:7] != "package" {
		t.Errorf("static.go doesn't start with 'package': %q", content[:min(len(content), 30)])
	}
}

func TestFSIsDir(t *testing.T) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			t.Errorf("unexpected directory in embed: %s", e.Name())
		}
	}
}

func TestFSNoSubdirs(t *testing.T) {
	// Verify the embed only contains flat files (no subdirectories)
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			t.Errorf("embed should not contain directories: %s", e.Name())
		}
	}
	// At minimum we have static.go and favicon.svg
	if len(entries) < 2 {
		t.Errorf("expected at least 2 files in embed, got %d", len(entries))
	}
}
