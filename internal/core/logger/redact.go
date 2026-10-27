package logger

import (
	"io"
	"regexp"

	"github.com/slemmercs/rclone-bisync/internal/core/security"
)

type redactPattern struct {
	regex       *regexp.Regexp
	replacement string
}

var redactReplacements []redactPattern

func init() {
	redacted := security.RedactedString

	redactReplacements = []redactPattern{
		// 1. Direct high-risk token values
		{
			regex:       regexp.MustCompile(`ya29\.[0-9A-Za-z\-_]+`),
			replacement: redacted,
		},
		{
			regex:       regexp.MustCompile(`1//[0-9A-Za-z\-_]{20,}`),
			replacement: redacted,
		},
		{
			regex:       regexp.MustCompile(`ghp_[A-Za-z0-9_]{36,}`),
			replacement: redacted,
		},
		{
			regex:       regexp.MustCompile(`xox[baprs]-[0-9a-zA-Z-]{10,}`),
			replacement: redacted,
		},

		// 2. HTTP Authorization & Token Headers
		{
			regex:       regexp.MustCompile(`(?i)(Authorization:\s*(?:Bearer|Basic)\s+)[^\s\r\n]+`),
			replacement: "${1}" + redacted,
		},
		{
			regex:       regexp.MustCompile(`(?i)(X-(?:[A-Za-z0-9-_]*Auth|Vault-Token)[A-Za-z0-9-_]*:\s*)[^\s\r\n]+`),
			replacement: "${1}" + redacted,
		},

		// 3. CLI flags (--rc-pass=secret, --rc-pass secret, --password 'foo')
		{
			regex:       regexp.MustCompile(`(?i)(--(?:rc-)?(?:pass(?:word)?|secret|token|api-?key)(?:=|\s+))(["']?)([^\s"'\r\n]+)(["']?)`),
			replacement: "${1}${2}" + redacted + "${4}",
		},

		// 4. Key-Value assignments (RC_PASS=xxx, rc_pass=xxx, token=xxx, secret=xxx)
		{
			regex:       regexp.MustCompile(`(?i)(\b(?:rc_)?(?:pass(?:word)?|secret|token|api_key|webhook_secret)\s*=\s*)(["']?)([^\s,"'\r\n]+)(["']?)`),
			replacement: "${1}${2}" + redacted + "${4}",
		},

		// 5. JSON fields ("password": "xxx", "rc_pass": "xxx", "secret": "xxx", "token": "xxx")
		{
			regex:       regexp.MustCompile(`(?i)("(?:\w*_)?(?:pass(?:word)?|secret|token|credential|api_key|webhook_secret)"\s*:\s*")((?:[^"\\]|\\.)*)(")`),
			replacement: "${1}" + redacted + "${3}",
		},

		// 6. Query string params (?access_token=xxx, &token=xxx, ?pass=xxx)
		{
			regex:       regexp.MustCompile(`(?i)([?&](?:pass(?:word)?|token|secret|api_key|access_token|refresh_token)=)[^&#\s]+`),
			replacement: "${1}" + redacted,
		},
	}
}

// RedactString sanitizes known credentials, tokens, flags, and headers from a string.
func RedactString(input string) string {
	res := input
	for _, p := range redactReplacements {
		res = p.regex.ReplaceAllString(res, p.replacement)
	}
	return res
}

// MaskLine is an alias for RedactString for single-line operations.
func MaskLine(line string) string {
	return RedactString(line)
}

// MaskingWriter wraps an io.Writer and intercepts writes to sanitize sensitive information.
type MaskingWriter struct {
	writer io.Writer
}

// NewMaskingWriter wraps any io.Writer with a secret redactor.
func NewMaskingWriter(w io.Writer) *MaskingWriter {
	if mw, ok := w.(*MaskingWriter); ok {
		return mw
	}
	return &MaskingWriter{writer: w}
}

// Write intercepts bytes, applies redaction, and writes the redacted bytes to the underlying writer.
// It returns len(p) on success to satisfy io.Writer semantics when transforming data length.
func (mw *MaskingWriter) Write(p []byte) (n int, err error) {
	if mw.writer == nil {
		return len(p), nil
	}
	redacted := RedactString(string(p))
	if _, err := mw.writer.Write([]byte(redacted)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close closes the underlying writer if it implements io.Closer.
func (mw *MaskingWriter) Close() error {
	if closer, ok := mw.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
