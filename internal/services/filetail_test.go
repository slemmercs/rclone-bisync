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

func TestTailJSONLogPaginated(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "paginated.log")

	// Missing file case
	resEmpty := TailJSONLogPaginated(filepath.Join(tmpDir, "nonexistent.log"), 1, 50)
	if resEmpty.Total != 0 || resEmpty.TotalPages != 1 || len(resEmpty.Logs) != 0 {
		t.Fatalf("expected empty response for nonexistent file, got %+v", resEmpty)
	}

	// Create 120 lines
	var sb strings.Builder
	for i := 1; i <= 120; i++ {
		sb.WriteString(strings.Repeat(" ", 0))
		sb.WriteString(`{"time":"2026-10-27T10:00:00Z","level":"info","msg":"line ` + strings.TrimSpace(string(rune('0'+i%10))) + ` with --rc-pass=secret"}` + "\n")
	}
	if err := os.WriteFile(logPath, []byte(sb.String()), 0600); err != nil {
		t.Fatalf("failed to write test log: %v", err)
	}

	// Page 1 (most recent 50)
	p1 := TailJSONLogPaginated(logPath, 1, 50)
	if p1.Total != 120 || p1.TotalPages != 3 || len(p1.Logs) != 50 || p1.Page != 1 {
		t.Fatalf("unexpected p1: Total=%d, TotalPages=%d, len=%d, Page=%d", p1.Total, p1.TotalPages, len(p1.Logs), p1.Page)
	}
	// Verify redaction
	if strings.Contains(p1.Logs[0], "secret") && !strings.Contains(p1.Logs[0], "********") {
		t.Errorf("expected secret to be redacted in paginated logs: %s", p1.Logs[0])
	}

	// Page 2 (middle 50)
	p2 := TailJSONLogPaginated(logPath, 2, 50)
	if len(p2.Logs) != 50 || p2.Page != 2 {
		t.Fatalf("unexpected p2: len=%d, Page=%d", len(p2.Logs), p2.Page)
	}

	// Page 3 (oldest 20)
	p3 := TailJSONLogPaginated(logPath, 3, 50)
	if len(p3.Logs) != 20 || p3.Page != 3 {
		t.Fatalf("unexpected p3: len=%d, Page=%d", len(p3.Logs), p3.Page)
	}
}
