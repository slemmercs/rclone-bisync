package profiles

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// UnmarshalJSON flexibly deserializes SafetySettings supporting both delete_threshold_pct
// and max_delete_percent (as float, int, or string percentage), failure_threshold, and canary_validation.
func (s *SafetySettings) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" || trimmed == "" {
		return nil
	}
	type Alias SafetySettings
	aux := (*Alias)(s)
	if err := json.Unmarshal(data, aux); err != nil {
		// Non-fatal if fields were typed differently; fallback to raw parsing
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	for _, key := range []string{"max_delete_percent", "delete_threshold_pct"} {
		if val, exists := raw[key]; exists && val != nil {
			switch v := val.(type) {
			case float64:
				s.DeleteThresholdPct = v
			case string:
				v = strings.TrimSuffix(strings.TrimSpace(v), "%")
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					s.DeleteThresholdPct = f
				}
			}
			break
		}
	}
	if val, exists := raw["failure_threshold"]; exists && val != nil {
		switch v := val.(type) {
		case float64:
			s.FailureThreshold = int(v)
		case string:
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				s.FailureThreshold = n
			}
		}
	}
	if val, exists := raw["canary_validation"]; exists && val != nil {
		switch v := val.(type) {
		case bool:
			s.CanaryValidation = v
		case string:
			s.CanaryValidation = strings.EqualFold(v, "true") || v == "1"
		}
	}
	return nil
}

// MaxDeleteValue represents a max-delete threshold as either an absolute file count (e.g. "10")
// or a percentage ratio (e.g. "10%").
type MaxDeleteValue string

func (m *MaxDeleteValue) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" || trimmed == `""` || trimmed == "" {
		*m = ""
		return nil
	}
	if strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`) {
		*m = MaxDeleteValue(strings.Trim(trimmed, `"`))
		return nil
	}
	*m = MaxDeleteValue(trimmed)
	return nil
}

func (m MaxDeleteValue) MarshalJSON() ([]byte, error) {
	s := string(m)
	if s == "" {
		return []byte("null"), nil
	}
	if strings.HasSuffix(s, "%") {
		return json.Marshal(s)
	}
	if n, err := strconv.Atoi(s); err == nil {
		return json.Marshal(n)
	}
	return json.Marshal(s)
}

func (m MaxDeleteValue) String() string {
	return string(m)
}

// ProviderOptionsMap maps provider-specific option names to string values.
// It implements flexible JSON unmarshaling to coerce booleans, numbers, and strings into strings.
type ProviderOptionsMap map[string]string

func (p *ProviderOptionsMap) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" || trimmed == "" {
		*p = make(ProviderOptionsMap)
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	res := make(ProviderOptionsMap, len(raw))
	for k, v := range raw {
		if v == nil {
			continue
		}
		switch val := v.(type) {
		case string:
			res[k] = val
		case bool:
			res[k] = strconv.FormatBool(val)
		case float64:
			if val == float64(int64(val)) {
				res[k] = strconv.FormatInt(int64(val), 10)
			} else {
				res[k] = strconv.FormatFloat(val, 'f', -1, 64)
			}
		default:
			b, err := json.Marshal(val)
			if err == nil {
				res[k] = string(b)
			} else {
				res[k] = fmt.Sprint(val)
			}
		}
	}
	*p = res
	return nil
}
