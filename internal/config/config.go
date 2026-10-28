package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config represents runtime settings for rclone-bisync.
type Config struct {
	BaseDir    string
	LocalPath  string
	RemotePath string
	ProfileID  string

	LogFile               string
	LockFile              string
	NotificationStateFile string
	MetricsFile           string
	RateLimitBackoffFile  string
	DaemonHealthFile      string
	PendingDeletionsFile  string
	ReviewDir             string
	ActiveReviewFile      string
	ReviewBackupsDir      string
	ReviewBackupTTLHours  float64
	AuditDBPath           string
	ProfilesFile          string
	ProfilesHistoryFile   string

	RCHost        string
	RCPort        int
	RCAPIURL      string
	RCAuthEnabled bool
	RCUser        string
	RCPass        string

	NotificationAppName            string
	SuccessNotificationIntervalMin float64
	ErrorNotificationRepeatHours   float64

	RcloneCommonArgs []string

	MaxLogSizeMB   int
	LogBackupCount int

	DaemonStartupTimeoutSec  int
	DaemonHealthCheckRetries int
	RetryDelaySec            float64

	JobPollInitialSleepSec   float64
	JobPollMaxSleepSec       float64
	JobPollBackoffMultiplier float64
	JobTimeoutSeconds        int

	RateLimitBackoffHours      float64
	RateLimitRetryAfterBackoff bool

	LockStaleTimeoutSeconds int
	CheckStaleLocks         bool

	EnableMetrics     bool
	MetricsMaxEntries int
	MetricsMaxBackups int

	FilesToCleanup []string

	ValidateConfigOnStartup bool
	RequireLocalPathExists  bool

	DebugMode          bool
	WebhookURL         string
	WebhookTimeoutSec  int
	WatchdogPingURL    string
	WatchdogTimeoutSec int

	GlobalDefaults GlobalDefaults

	Features map[string]bool
}

// ParseEnvFile reads a simple .env file and parses its key-value pairs into a map.
// It supports comments (#), balanced single/double quotes, and ignores missing files.
func ParseEnvFile(path string) (map[string]string, error) {
	// #nosec G304 - reading configured environment file
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	defer f.Close()

	envMap := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		// Strip balanced matching quotes only
		if len(val) >= 2 {
			if (strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) ||
				(strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`)) {
				val = val[1 : len(val)-1]
			}
		}

		if key != "" {
			envMap[key] = val
		}
	}
	return envMap, scanner.Err()
}

// LoadEnvFile reads a simple .env file and sets values into the environment if not already set.
func LoadEnvFile(path string) error {
	envMap, err := ParseEnvFile(path)
	if err != nil {
		return err
	}
	for key, val := range envMap {
		if current := os.Getenv(key); current == "" {
			_ = os.Setenv(key, val)
		}
	}
	return nil
}

func str2bool(val string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(val))
	switch v {
	case "true", "1", "t", "yes", "y", "on":
		return true
	case "false", "0", "f", "no", "n", "off":
		return false
	default:
		return def
	}
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func getEnvInt(key string, def int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if val := os.Getenv(key); val != "" {
		return str2bool(val, def)
	}
	return def
}

func splitArgs(raw string) []string {
	var args []string
	fields := strings.Fields(raw)
	for _, f := range fields {
		if tr := strings.TrimSpace(f); tr != "" {
			args = append(args, tr)
		}
	}
	return args
}

// LoadConfig discovers configuration files and loads runtime settings.
func LoadConfig(searchDirs ...string) (*Config, error) {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, "rclone-bisync")

	var dirs []string
	for _, d := range searchDirs {
		if strings.TrimSpace(d) != "" {
			dirs = append(dirs, strings.TrimSpace(d))
		}
	}
	if len(dirs) == 0 {
		dirs = []string{
			baseDir,
			".",
		}
	}

	for _, d := range dirs {
		fi, err := os.Stat(d)
		if err == nil && !fi.IsDir() {
			_ = LoadEnvFile(d)
			_ = LoadEnvFile(filepath.Join(filepath.Dir(d), "rc-secrets.env"))
			baseDir = filepath.Dir(d)
			continue
		}
		_ = LoadEnvFile(filepath.Join(d, "rc-secrets.env"))
		_ = LoadEnvFile(filepath.Join(d, "bisync.env"))
		if err == nil && fi.IsDir() && d != "." {
			baseDir = d
		}
	}

	baseDir = getEnv("BASE_DIR", baseDir)

	rcPass := os.Getenv("RC_PASS")
	rcAuthEnabled := true
	if rcAuthEnabled && rcPass == "" {
		return nil, fmt.Errorf("RC_PASS is not set in environment, bisync.env, or rc-secrets.env. Please configure RC_PASS to ensure authenticated access to Rclone RC")
	}

	localPathRaw := getEnv("LOCAL_PATH", "~/gdrive")
	if strings.HasPrefix(localPathRaw, "~/") {
		localPathRaw = filepath.Join(home, localPathRaw[2:])
	}

	rcPort := getEnvInt("RC_PORT", 5572)
	rcHost := getEnv("RC_HOST", "localhost")
	rcAPIURL := fmt.Sprintf("http://%s:%d", rcHost, rcPort)

	defaultArgs := "--use-json-log --color NEVER " +
		"--check-access --check-first --track-renames --remove-empty-dirs --resilient --recover " +
		"--retries 4 --retries-sleep 15s --low-level-retries 10 --tpslimit 10 --tpslimit-burst 10 " +
		"--fast-list --buffer-size 32M --conflict-resolve newer --conflict-loser num"
	rawArgs := getEnv("RCLONE_COMMON_ARGS", defaultArgs)

	var filesToCleanup []string
	if rawClean := os.Getenv("FILES_TO_CLEANUP"); rawClean != "" {
		for _, p := range strings.Split(rawClean, ",") {
			if s := strings.TrimSpace(p); s != "" {
				filesToCleanup = append(filesToCleanup, s)
			}
		}
	}

	features := map[string]bool{
		"stale_lock_detection": getEnvBool("FEATURE_STALE_LOCK_DETECTION", true),
		"rate_limit_backoff":   getEnvBool("FEATURE_RATE_LIMIT_BACKOFF", true),
		"daemon_health_check":  getEnvBool("FEATURE_DAEMON_HEALTH_CHECK", true),
		"metrics_collection":   getEnvBool("FEATURE_METRICS_COLLECTION", true),
		"smart_notifications":  getEnvBool("FEATURE_SMART_NOTIFICATIONS", true),
		"webhook_alerts":       getEnvBool("FEATURE_WEBHOOK_ALERTS", false),
	}

	cfg := &Config{
		BaseDir:    baseDir,
		LocalPath:  localPathRaw,
		RemotePath: getEnv("REMOTE_PATH", "gdrive:"),

		LogFile:               filepath.Join(baseDir, "bisync.log"),
		LockFile:              filepath.Join(baseDir, "bisync_gdrive.lock"),
		NotificationStateFile: filepath.Join(baseDir, "notification_state.json"),
		MetricsFile:           filepath.Join(baseDir, "metrics.jsonl"),
		RateLimitBackoffFile:  filepath.Join(baseDir, "rate_limit.flag"),
		DaemonHealthFile:      filepath.Join(baseDir, "daemon_health.json"),
		PendingDeletionsFile:  filepath.Join(baseDir, "pending_deletions.json"),
		ReviewDir:             filepath.Join(baseDir, "review_sessions"),
		ActiveReviewFile:      filepath.Join(baseDir, "active_review_session.json"),
		ReviewBackupsDir:      filepath.Join(baseDir, "review_backups"),
		ReviewBackupTTLHours:  getEnvFloat("REVIEW_BACKUP_TTL_HOURS", 48.0),
		AuditDBPath:           getEnv("AUDIT_DB_PATH", filepath.Join(baseDir, "audit.db")),
		ProfilesFile:          getEnv("PROFILES_FILE", filepath.Join(baseDir, "profiles.json")),
		ProfilesHistoryFile:   getEnv("PROFILES_HISTORY_FILE", filepath.Join(baseDir, "profiles_history.json")),

		RCHost:        rcHost,
		RCPort:        rcPort,
		RCAPIURL:      rcAPIURL,
		RCAuthEnabled: rcAuthEnabled,
		RCUser:        getEnv("RC_USER", "rclone"),
		RCPass:        rcPass,

		NotificationAppName:            getEnv("NOTIFICATION_APP_NAME", "Rclone Bisync"),
		SuccessNotificationIntervalMin: getEnvFloat("SUCCESS_NOTIFICATION_INTERVAL_MIN", 30),
		ErrorNotificationRepeatHours:   getEnvFloat("ERROR_NOTIFICATION_REPEAT_HOURS", 6),

		RcloneCommonArgs: splitArgs(rawArgs),

		MaxLogSizeMB:   getEnvInt("MAX_LOG_SIZE_MB", 5),
		LogBackupCount: getEnvInt("LOG_BACKUP_COUNT", 5),

		DaemonStartupTimeoutSec:  getEnvInt("DAEMON_STARTUP_TIMEOUT_SEC", 30),
		DaemonHealthCheckRetries: getEnvInt("DAEMON_HEALTH_CHECK_RETRIES", 2),
		RetryDelaySec:            getEnvFloat("RETRY_DELAY", 1),

		JobPollInitialSleepSec:   getEnvFloat("JOB_POLL_INITIAL_SLEEP", 2),
		JobPollMaxSleepSec:       getEnvFloat("JOB_POLL_MAX_SLEEP", 10),
		JobPollBackoffMultiplier: getEnvFloat("JOB_POLL_BACKOFF_MULTIPLIER", 1.5),
		JobTimeoutSeconds:        getEnvInt("JOB_TIMEOUT_SECONDS", 1800),

		RateLimitBackoffHours:      getEnvFloat("RATE_LIMIT_BACKOFF_HOURS", 1),
		RateLimitRetryAfterBackoff: getEnvBool("RATE_LIMIT_RETRY_AFTER_BACKOFF", true),

		LockStaleTimeoutSeconds: getEnvInt("LOCK_STALE_TIMEOUT_SECONDS", 3600),
		CheckStaleLocks:         getEnvBool("CHECK_STALE_LOCKS", true),

		EnableMetrics:     getEnvBool("ENABLE_METRICS", true),
		MetricsMaxEntries: getEnvInt("METRICS_MAX_ENTRIES", 1000),
		MetricsMaxBackups: getEnvInt("METRICS_MAX_BACKUPS", 5),

		FilesToCleanup: filesToCleanup,

		ValidateConfigOnStartup: getEnvBool("VALIDATE_CONFIG_ON_STARTUP", true),
		RequireLocalPathExists:  getEnvBool("REQUIRE_LOCAL_PATH_EXISTS", true),

		DebugMode:          getEnvBool("DEBUG_MODE", false),
		WebhookURL:         os.Getenv("WEBHOOK_URL"),
		WebhookTimeoutSec:  getEnvInt("WEBHOOK_TIMEOUT", 5),
		WatchdogPingURL:    os.Getenv("WATCHDOG_PING_URL"),
		WatchdogTimeoutSec: getEnvInt("WATCHDOG_TIMEOUT", 5),

		GlobalDefaults: GlobalDefaults{
			MaxDelete:        getEnvInt("GLOBAL_MAX_DELETE", 50),
			CanaryValidation: getEnvBool("GLOBAL_CANARY_VALIDATION", true),
			ConflictResolve:  getEnv("GLOBAL_CONFLICT_RESOLVE", "newer"),
			CompareMode:      getEnv("GLOBAL_COMPARE_MODE", "modtime"),
			RemoveEmptyDirs:  getEnvBool("GLOBAL_REMOVE_EMPTY_DIRS", true),
			CreateEmptyDirs:  getEnvBool("GLOBAL_CREATE_EMPTY_DIRS", true),
		},

		Features: features,
	}

	return cfg, nil
}
