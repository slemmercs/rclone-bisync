package config

import (
	"encoding/json"
	"testing"
)

func TestDefaultGlobalDefaults(t *testing.T) {
	d := DefaultGlobalDefaults()

	if d.MaxDelete != 0 {
		t.Errorf("expected MaxDelete 0, got %d", d.MaxDelete)
	}
	if !d.CanaryValidation {
		t.Errorf("expected CanaryValidation true")
	}
	if d.ConflictResolve != "newer" {
		t.Errorf("expected ConflictResolve 'newer', got '%s'", d.ConflictResolve)
	}
	if d.CompareMode != "size,modtime" {
		t.Errorf("expected CompareMode 'size,modtime', got '%s'", d.CompareMode)
	}
	if !d.RemoveEmptyDirs {
		t.Errorf("expected RemoveEmptyDirs true")
	}
	if !d.CreateEmptyDirs {
		t.Errorf("expected CreateEmptyDirs true")
	}
	if !d.LockExplorerRemote {
		t.Errorf("expected LockExplorerRemote true by default")
	}

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("failed to marshal GlobalDefaults: %v", err)
	}

	var decoded GlobalDefaults
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal GlobalDefaults: %v", err)
	}

	if decoded != d {
		t.Errorf("expected decoded struct to match original, got %+v", decoded)
	}
}
