package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTailLines_RedactsSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "tail_secret.log")

	content := "line 1: bisync startup\n" +
		"line 2: authenticating with --rc-pass=supersecretpassword\n" +
		"line 3: token ya29.a0AfH6SMB1234567890abcdef obtained\n" +
		"line 4: Authorization: Bearer secret-oauth-token-here\n" +
		"line 5: sync finished\n"

	if err := os.WriteFile(logPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test log: %v", err)
	}

	lines := TailLines(logPath, 5)
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}

	for _, line := range lines {
		if strings.Contains(line, "supersecretpassword") {
			t.Errorf("line contains unredacted password: %s", line)
		}
		if strings.Contains(line, "ya29.a0AfH6SMB") {
			t.Errorf("line contains unredacted access token: %s", line)
		}
		if strings.Contains(line, "secret-oauth-token-here") {
			t.Errorf("line contains unredacted bearer token: %s", line)
		}
	}

	if !strings.Contains(lines[1], "--rc-pass=********") {
		t.Errorf("expected masked rc-pass in line 2, got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "token ******** obtained") {
		t.Errorf("expected masked token in line 3, got: %s", lines[2])
	}
	if !strings.Contains(lines[3], "Authorization: Bearer ********") {
		t.Errorf("expected masked Authorization header in line 4, got: %s", lines[3])
	}
}

func TestTailJSONLog_RedactsSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "tail_json_secret.log")

	line1 := `{"time":"2026-10-27T10:00:00Z","level":"info","msg":"connecting with --rc-pass=secret123"}`
	line2 := `{"time":"2026-10-27T10:01:00Z","level":"notice","msg":"{\"token\":\"ya29.secretToken456\"}"}`
	if err := os.WriteFile(logPath, []byte(line1+"\n"+line2+"\n"), 0600); err != nil {
		t.Fatalf("failed to write test log: %v", err)
	}

	entries := TailJSONLog(logPath, 2)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	for _, entry := range entries {
		if strings.Contains(entry, "secret123") {
			t.Errorf("entry contains unredacted password: %s", entry)
		}
		if strings.Contains(entry, "ya29.secretToken456") {
			t.Errorf("entry contains unredacted token: %s", entry)
		}
		if !strings.Contains(entry, "********") {
			t.Errorf("expected masked entry, got: %s", entry)
		}
	}
}
