package classifier

import (
	"strings"
)

// ClassifyError classifies error message into categories for targeted automated recovery and alerting.
func ClassifyError(errorMsg string) string {
	if errorMsg == "" {
		return "UNKNOWN"
	}

	msgLower := strings.ToLower(errorMsg)

	if strings.Contains(msgLower, "prior lock file found") || strings.Contains(msgLower, "lock") {
		return "STALE_LOCK"
	}

	if strings.Contains(msgLower, "cannot find prior") {
		return "NO_LISTINGS"
	}

	for _, x := range []string{"rate", "429", "quota exceeded"} {
		if strings.Contains(msgLower, x) {
			return "RATE_LIMIT"
		}
	}

	for _, x := range []string{
		"permission denied", "401", "403", "unauthorized",
		"oauth", "invalid_grant", "token expired", "token revoked",
	} {
		if strings.Contains(msgLower, x) {
			return "PERMISSION"
		}
	}

	for _, x := range []string{"timeout", "connection", "network", "unreachable", "handshake"} {
		if strings.Contains(msgLower, x) {
			return "NETWORK"
		}
	}

	if strings.Contains(msgLower, "conflict") {
		return "CONFLICT"
	}

	if strings.Contains(msgLower, "daemon") || strings.Contains(msgLower, "rcd") {
		return "DAEMON_ERROR"
	}

	if strings.Contains(msgLower, "too many deletes") {
		return "TOO_MANY_DELETES"
	}

	if strings.Contains(msgLower, "rclone_test") || strings.Contains(msgLower, "checkaccess") {
		return "CHECK_ACCESS_FAILED"
	}

	return "UNKNOWN"
}
