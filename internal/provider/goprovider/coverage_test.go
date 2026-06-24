package goprovider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- recoverPartialProfile Tests ---

func TestRecoverPartialProfile_ExistsWithData(t *testing.T) {
	// go test exits non-zero but profile exists with data.
	// Verify recoverPartialProfile returns the profile path and
	// emits a warning to stderr.
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "cover.out")
	err := os.WriteFile(profilePath, []byte("mode: set\nsome/pkg/file.go:1.1,5.1 1 1\n"), 0644)
	if err != nil {
		t.Fatalf("writing test profile: %v", err)
	}

	var stderr strings.Builder
	testErr := fmt.Errorf("exit status 1")

	got, recoverErr := recoverPartialProfile(profilePath, testErr, &stderr)
	if recoverErr != nil {
		t.Fatalf("expected nil error, got: %v", recoverErr)
	}
	if got != profilePath {
		t.Errorf("expected profile path %q, got %q", profilePath, got)
	}

	// Verify warning was emitted.
	warning := stderr.String()
	if !strings.Contains(warning, "warning:") {
		t.Errorf("expected warning in stderr, got %q", warning)
	}
	if !strings.Contains(warning, "partial coverage used") {
		t.Errorf("expected 'partial coverage used' in warning, got %q", warning)
	}
	if !strings.Contains(warning, "exit status 1") {
		t.Errorf("expected test error in warning, got %q", warning)
	}

	// Verify profile was NOT deleted.
	if _, statErr := os.Stat(profilePath); statErr != nil {
		t.Errorf("profile should still exist after recovery, got: %v", statErr)
	}
}

func TestRecoverPartialProfile_Missing(t *testing.T) {
	// go test exits non-zero and profile is missing.
	// Verify hard error returned.
	profilePath := filepath.Join(t.TempDir(), "nonexistent.out")
	testErr := fmt.Errorf("exit status 1")

	var stderr strings.Builder
	_, recoverErr := recoverPartialProfile(profilePath, testErr, &stderr)
	if recoverErr == nil {
		t.Fatal("expected error for missing profile, got nil")
	}

	// Verify no warning was emitted (error path, not warning path).
	if stderr.Len() != 0 {
		t.Errorf("expected no stderr output for missing profile, got %q", stderr.String())
	}
}

func TestRecoverPartialProfile_Empty(t *testing.T) {
	// go test exits non-zero and profile is empty (0 bytes).
	// Verify hard error returned and file is cleaned up.
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "empty.out")
	err := os.WriteFile(profilePath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("writing empty profile: %v", err)
	}

	var stderr strings.Builder
	testErr := fmt.Errorf("exit status 2")

	_, recoverErr := recoverPartialProfile(profilePath, testErr, &stderr)
	if recoverErr == nil {
		t.Fatal("expected error for empty profile, got nil")
	}

	// Verify file was cleaned up.
	if _, statErr := os.Stat(profilePath); !os.IsNotExist(statErr) {
		t.Errorf("expected empty profile to be removed, but it still exists")
	}

	// Verify no warning was emitted (error path, not warning path).
	if stderr.Len() != 0 {
		t.Errorf("expected no stderr output for empty profile, got %q", stderr.String())
	}
}

func TestRecoverPartialProfile_NilStderr(t *testing.T) {
	// Verify that nil stderr writer doesn't panic — warning is
	// suppressed when no writer is provided.
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "cover.out")
	err := os.WriteFile(profilePath, []byte("mode: set\n"), 0644)
	if err != nil {
		t.Fatalf("writing test profile: %v", err)
	}

	testErr := fmt.Errorf("exit status 1")

	got, recoverErr := recoverPartialProfile(profilePath, testErr, nil)
	if recoverErr != nil {
		t.Fatalf("expected nil error with nil stderr, got: %v", recoverErr)
	}
	if got != profilePath {
		t.Errorf("expected profile path %q, got %q", profilePath, got)
	}
}
