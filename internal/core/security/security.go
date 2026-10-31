package security

import (
	"context"
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type contextKey string

const actorContextKey contextKey = "rclone_bisync_actor"

// ContextWithActor returns a copy of parent context with actor identity attached.
func ContextWithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, actorContextKey, actor)
}

// ActorFromContext retrieves the actor identity from context, defaulting to fallback if unset.
func ActorFromContext(ctx context.Context, fallback string) string {
	if ctx == nil {
		return fallback
	}
	if val, ok := ctx.Value(actorContextKey).(string); ok && val != "" {
		return val
	}
	return fallback
}

// SensitiveKeywords identifies identifiers hinting at sensitive configuration values.
var SensitiveKeywords = []string{
	"PASS",
	"SECRET",
	"TOKEN",
	"KEY",
	"CREDENTIAL",
	"AUTH",
	"WEBHOOK",
	"PRIVATE",
}

// SensitiveValueRegexes detects token strings matching high-risk patterns.
var SensitiveValueRegexes = []*regexp.Regexp{
	regexp.MustCompile(`ghp_[A-Za-z0-9_]{36,}`),        // GitHub PAT
	regexp.MustCompile(`xox[baprs]-[0-9a-zA-Z-]{10,}`), // Slack Token
	regexp.MustCompile(`ya29\.[0-9A-Za-z\-_]+`),        // Google OAuth Access Token
}

const RedactedString = "********"

// isRedactableValue checks if a value is of a type that can represent a secret string or byte sequence.
func isRedactableValue(val reflect.Value) bool {
	if !val.IsValid() {
		return false
	}
	k := val.Kind()
	return k == reflect.String || k == reflect.Slice || k == reflect.Array
}

// IsSensitiveKeyName checks if a string identifier hints at confidential data.
func IsSensitiveKeyName(key string) bool {
	upper := strings.ToUpper(key)
	if strings.HasPrefix(upper, "HAS_") || strings.HasPrefix(upper, "IS_") {
		return false
	}
	if strings.HasSuffix(upper, "_REQUIRED") || strings.HasSuffix(upper, "_STATUS") || strings.HasSuffix(upper, "_COUNT") || strings.HasSuffix(upper, "_MSG") || strings.HasSuffix(upper, "_MESSAGE") {
		return false
	}
	for _, kw := range SensitiveKeywords {
		if strings.Contains(upper, kw) {
			return true
		}
	}
	return false
}

// IsSensitiveValue checks if a string value matches known high-risk token signatures.
func IsSensitiveValue(val string) bool {
	for _, pat := range SensitiveValueRegexes {
		if pat.MatchString(val) {
			return true
		}
	}
	return false
}

// RedactSecrets recursively traverses maps, slices, arrays, and structs,
// sanitizing values with sensitive keys or values matching token patterns.
func RedactSecrets(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	val := reflect.ValueOf(data)

	// If pointer or interface, unwrap
	for val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Map:
		out := make(map[string]interface{})
		iter := val.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()
			kStr := k.String()

			vInterface := v.Interface()
			vVal := reflect.ValueOf(vInterface)
			isNested := false
			if vVal.IsValid() {
				kind := vVal.Kind()
				for kind == reflect.Pointer || kind == reflect.Interface {
					if vVal.IsNil() {
						break
					}
					vVal = vVal.Elem()
					kind = vVal.Kind()
				}
				if (kind == reflect.Map || kind == reflect.Slice || kind == reflect.Array || kind == reflect.Struct) && vVal.Type() != reflect.TypeOf(time.Time{}) {
					isNested = true
				}
			}

			if isNested {
				out[kStr] = RedactSecrets(vInterface)
			} else if IsSensitiveKeyName(kStr) && isRedactableValue(vVal) {
				if vInterface != nil && vInterface != "" {
					out[kStr] = RedactedString
				} else {
					out[kStr] = vInterface
				}
			} else {
				out[kStr] = RedactSecrets(vInterface)
			}
		}
		return out

	case reflect.Slice, reflect.Array:
		if val.Type() == reflect.TypeOf(json.RawMessage{}) {
			var unmarshaled interface{}
			if err := json.Unmarshal(val.Bytes(), &unmarshaled); err == nil {
				redacted := RedactSecrets(unmarshaled)
				if redBytes, err := json.Marshal(redacted); err == nil {
					return json.RawMessage(redBytes)
				}
			}
			return val.Interface()
		}
		if val.Type() == reflect.TypeOf([]byte{}) {
			return val.Interface()
		}

		out := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			out[i] = RedactSecrets(val.Index(i).Interface())
		}
		return out

	case reflect.Struct:
		if val.Type() == reflect.TypeOf(time.Time{}) {
			return val.Interface()
		}

		// Convert struct to map based on json tags or field names
		out := make(map[string]interface{})
		t := val.Type()
		for i := 0; i < val.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}

			key := f.Name
			if tag := f.Tag.Get("json"); tag != "" && tag != "-" {
				parts := strings.Split(tag, ",")
				if parts[0] != "" {
					key = parts[0]
				}
			}

			fieldVal := val.Field(i).Interface()
			fVal := reflect.ValueOf(fieldVal)
			isNested := false
			if fVal.IsValid() {
				kind := fVal.Kind()
				for kind == reflect.Pointer || kind == reflect.Interface {
					if fVal.IsNil() {
						break
					}
					fVal = fVal.Elem()
					kind = fVal.Kind()
				}
				if (kind == reflect.Map || kind == reflect.Slice || kind == reflect.Array || kind == reflect.Struct) && fVal.Type() != reflect.TypeOf(time.Time{}) {
					isNested = true
				}
			}

			if isNested {
				out[key] = RedactSecrets(fieldVal)
			} else if IsSensitiveKeyName(key) && isRedactableValue(fVal) {
				if fieldVal != nil && fieldVal != "" {
					out[key] = RedactedString
				} else {
					out[key] = fieldVal
				}
			} else {
				out[key] = RedactSecrets(fieldVal)
			}
		}
		return out

	case reflect.String:
		s := val.String()
		if IsSensitiveValue(s) {
			return RedactedString
		}
		return s

	default:
		return data
	}
}
