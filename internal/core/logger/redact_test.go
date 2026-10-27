package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/slemmercs/rclone-bisync/internal/core/security"
)

type dummyCloser struct {
	bytes.Buffer
	closed bool
}

func (d *dummyCloser) Close() error {
	d.closed = true
	return nil
}

func TestRedactString_Tokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Google OAuth access token",
			input:    "Received token ya29.a0AfH6SMB_1234567890abcdef for profile",
			expected: "Received token ******** for profile",
		},
		{
			name:     "Google OAuth refresh token",
			input:    "Refresh token 1//0gAbCdEfGhIjKlMnOpQrStUvWxYz12345 in response",
			expected: "Refresh token ******** in response",
		},
		{
			name:     "GitHub PAT",
			input:    "git auth token ghp_123456789012345678901234567890123456 passed",
			expected: "git auth token ******** passed",
		},
		{
			name:     "Slack token",
			input:    "notifier token xoxb-1234567890-abcdefghij initialized",
			expected: "notifier token ******** initialized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := RedactString(tc.input)
			if got != tc.expected {
				t.Errorf("RedactString(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestRedactString_HeadersAndFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Authorization Bearer header",
			input:    "Authorization: Bearer secret-bearer-token\r\n",
			expected: "Authorization: Bearer ********\r\n",
		},
		{
			name:     "Authorization Basic header",
			input:    "Authorization: Basic dXNlcjpwYXNz",
			expected: "Authorization: Basic ********",
		},
		{
			name:     "Vault Token header",
			input:    "X-Vault-Token: s.123456789abcdef",
			expected: "X-Vault-Token: ********",
		},
		{
			name:     "CLI flag --rc-pass with space",
			input:    "rclone rc sync/bisync --rc-pass mypassword --dry-run",
			expected: "rclone rc sync/bisync --rc-pass ******** --dry-run",
		},
		{
			name:     "CLI flag --rc-pass with equals",
			input:    "rclone --rc-pass=mypassword sync local: remote:",
			expected: "rclone --rc-pass=******** sync local: remote:",
		},
		{
			name:     "CLI flag with quotes",
			input:    `rclone --password "quoted-secret" bisync`,
			expected: `rclone --password "********" bisync`,
		},
		{
			name:     "Environment variable assignment",
			input:    "RC_PASS=topsecret rclone-bisync run",
			expected: "RC_PASS=******** rclone-bisync run",
		},
		{
			name:     "URL query param access_token",
			input:    "https://api.example.com/oauth?access_token=secret123&scope=drive",
			expected: "https://api.example.com/oauth?access_token=********&scope=drive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MaskLine(tc.input)
			if got != tc.expected {
				t.Errorf("MaskLine(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestRedactString_JSON(t *testing.T) {
	input := `{"user": "admin", "rc_pass": "supersecret", "token": "tok123", "status": "ok"}`
	got := RedactString(input)

	if strings.Contains(got, "supersecret") {
		t.Errorf("expected rc_pass to be redacted, got: %s", got)
	}
	if strings.Contains(got, "tok123") {
		t.Errorf("expected token to be redacted, got: %s", got)
	}
	if !strings.Contains(got, `"rc_pass": "********"`) {
		t.Errorf("expected masked rc_pass field, got: %s", got)
	}
	if !strings.Contains(got, `"user": "admin"`) {
		t.Errorf("expected non-sensitive fields preserved, got: %s", got)
	}
}

func TestMaskingWriter(t *testing.T) {
	closer := &dummyCloser{}
	mw := NewMaskingWriter(closer)

	input := []byte("starting sync with --rc-pass secret123 and token ya29.secret456\n")
	n, err := mw.Write(input)
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if n != len(input) {
		t.Errorf("expected n=%d, got %d", len(input), n)
	}

	written := closer.String()
	if strings.Contains(written, "secret123") {
		t.Errorf("found unredacted pass in output: %s", written)
	}
	if strings.Contains(written, "ya29.secret456") {
		t.Errorf("found unredacted token in output: %s", written)
	}
	if !strings.Contains(written, security.RedactedString) {
		t.Errorf("expected redacted string in output: %s", written)
	}

	if err := mw.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
	if !closer.closed {
		t.Error("expected underlying closer to be closed")
	}

	// Idempotent wrapping
	same := NewMaskingWriter(mw)
	if same != mw {
		t.Error("expected NewMaskingWriter to be idempotent when wrapping another MaskingWriter")
	}

	// Nil writer handling
	nilWriter := NewMaskingWriter(nil)
	n, err = nilWriter.Write([]byte("test"))
	if err != nil || n != 4 {
		t.Errorf("expected successful no-op write on nil writer, got n=%d, err=%v", n, err)
	}
	if err := nilWriter.Close(); err != nil {
		t.Errorf("expected successful nil Close(), got: %v", err)
	}
}
