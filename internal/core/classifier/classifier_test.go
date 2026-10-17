package classifier

import (
	"testing"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		msg      string
		expected string
	}{
		{"Fatal: cannot find prior path1 or path2 listings", "NO_LISTINGS"},
		{"HTTP 429 Too Many Requests: user rate limit exceeded", "RATE_LIMIT"},
		{"Google drive quota exceeded for user", "RATE_LIMIT"},
		{"401 Unauthorized", "PERMISSION"},
		{"Permission Denied: access to remote refused", "PERMISSION"},
		{"oauth2: cannot fetch token: 400 Bad Request Response: invalid_grant", "PERMISSION"},
		{"Token expired or revoked by user", "PERMISSION"},
		{"Prior lock file found on remote path", "STALE_LOCK"},
		{"Safety abort: too many deletes found in path2", "TOO_MANY_DELETES"},
		{"RCLONE_TEST missing from local folder", "CHECK_ACCESS_FAILED"},
		{"Connection timed out after 30 seconds", "NETWORK"},
		{"TLS handshake error: remote host unreachable", "NETWORK"},
		{"Some random unexpected error string", "UNKNOWN"},
		{"", "UNKNOWN"},
	}

	for _, tc := range tests {
		act := ClassifyError(tc.msg)
		if act != tc.expected {
			t.Errorf("msg: %s, expected: %s, got: %s", tc.msg, tc.expected, act)
		}
	}
}
