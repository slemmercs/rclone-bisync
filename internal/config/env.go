package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
