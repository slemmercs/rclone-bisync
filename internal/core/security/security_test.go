package security

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

type CredentialContainer struct {
	Username string            `json:"username"`
	Password string            `json:"password"`
	Meta     map[string]string `json:"meta"`
}

func TestRedactSecretsBasic(t *testing.T) {
	input := map[string]interface{}{
		"normal_field": "visible",
		"rc_pass":      "my_secret_pass",
		"empty_pass":   "",
		"count":        42,
		"active":       true,
	}

	result := RedactSecrets(input).(map[string]interface{})

	if result["normal_field"] != "visible" {
		t.Errorf("expected visible, got %v", result["normal_field"])
	}
	if result["rc_pass"] != RedactedString {
		t.Errorf("expected %s, got %v", RedactedString, result["rc_pass"])
	}
	if result["empty_pass"] != "" {
		t.Errorf("expected empty string preserved, got %v", result["empty_pass"])
	}
	if result["count"] != 42 {
		t.Errorf("expected 42, got %v", result["count"])
	}
}

func TestRedactSecretsNested(t *testing.T) {
	input := map[string]interface{}{
		"services": []interface{}{
			map[string]interface{}{
				"name":    "google",
				"api_key": "AIzaSyD-secret",
				"user_id": 123,
			},
		},
		"nested": map[string]interface{}{
			"slack_webhook": "https://hooks.slack.com/services/T00/B00/X00",
			"safe_note":     "all good",
		},
	}

	result := RedactSecrets(input).(map[string]interface{})

	services := result["services"].([]interface{})
	firstService := services[0].(map[string]interface{})
	if firstService["api_key"] != RedactedString {
		t.Errorf("expected api_key redacted, got %v", firstService["api_key"])
	}
	if firstService["name"] != "google" {
		t.Errorf("expected google, got %v", firstService["name"])
	}

	nested := result["nested"].(map[string]interface{})
	if nested["slack_webhook"] != RedactedString {
		t.Errorf("expected webhook redacted, got %v", nested["slack_webhook"])
	}
	if nested["safe_note"] != "all good" {
		t.Errorf("expected safe_note preserved, got %v", nested["safe_note"])
	}
}

func TestRedactSensitiveValuesRegex(t *testing.T) {
	input := map[string]interface{}{
		"arbitrary_string": "bearer ya29.a0ARrdaM-12345abcdef",
		"git_ref":          "ghp_123456789012345678901234567890123456",
		"normal_url":       "https://rclone.org",
	}

	result := RedactSecrets(input).(map[string]interface{})

	if result["arbitrary_string"] != RedactedString {
		t.Errorf("expected ya29 token redacted, got %v", result["arbitrary_string"])
	}
	if result["git_ref"] != RedactedString {
		t.Errorf("expected ghp token redacted, got %v", result["git_ref"])
	}
	if result["normal_url"] != "https://rclone.org" {
		t.Errorf("expected URL preserved, got %v", result["normal_url"])
	}
}

func TestRedactStruct(t *testing.T) {
	cred := CredentialContainer{
		Username: "admin",
		Password: "verysecretpassword",
		Meta: map[string]string{
			"auth_token": "token123",
			"desc":       "testing",
		},
	}

	result := RedactSecrets(cred).(map[string]interface{})

	if result["username"] != "admin" {
		t.Errorf("expected username admin, got %v", result["username"])
	}
	if result["password"] != RedactedString {
		t.Errorf("expected password redacted, got %v", result["password"])
	}

	meta := result["meta"].(map[string]interface{})
	if meta["auth_token"] != RedactedString {
		t.Errorf("expected auth_token redacted, got %v", meta["auth_token"])
	}
	if meta["desc"] != "testing" {
		t.Errorf("expected desc preserved, got %v", meta["desc"])
	}
}

func TestRedactSecretsRawMessageAndBytes(t *testing.T) {
	raw := json.RawMessage(`{"safe":"value","secret_key":"hidden123"}`)
	redacted := RedactSecrets(raw).(json.RawMessage)

	var res map[string]interface{}
	if err := json.Unmarshal(redacted, &res); err != nil {
		t.Fatalf("failed to unmarshal redacted raw message: %v", err)
	}
	if res["safe"] != "value" {
		t.Errorf("expected safe=value, got %v", res["safe"])
	}
	if res["secret_key"] != RedactedString {
		t.Errorf("expected secret_key redacted, got %v", res["secret_key"])
	}

	byteSlice := []byte("plain binary or text")
	redactedBytes := RedactSecrets(byteSlice).([]byte)
	if string(redactedBytes) != "plain binary or text" {
		t.Errorf("expected byte slice preserved, got %s", string(redactedBytes))
	}
}

func TestRedactStructWithRawMessage(t *testing.T) {
	type TestStructWithRaw struct {
		Details json.RawMessage `json:"details"`
	}
	s := TestStructWithRaw{
		Details: json.RawMessage(`{"file":"doc.pdf"}`),
	}
	res := RedactSecrets(s).(map[string]interface{})
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshaled: %s", string(b))
	if string(b) != `{"details":{"file":"doc.pdf"}}` {
		t.Errorf("expected JSON object, got: %s", string(b))
	}
}

func TestActorContext(t *testing.T) {
	// Nil context fallback
	if act := ActorFromContext(nil, "fallback-user"); act != "fallback-user" {
		t.Errorf("expected fallback-user, got %s", act)
	}

	// Empty context fallback
	ctx := context.Background()
	if act := ActorFromContext(ctx, "default"); act != "default" {
		t.Errorf("expected default, got %s", act)
	}

	// Empty actor in context fallback
	ctxEmpty := ContextWithActor(ctx, "")
	if act := ActorFromContext(ctxEmpty, "default"); act != "default" {
		t.Errorf("expected default, got %s", act)
	}

	// Valid actor in context
	ctxActor := ContextWithActor(ctx, "operator-1")
	if act := ActorFromContext(ctxActor, "default"); act != "operator-1" {
		t.Errorf("expected operator-1, got %s", act)
	}
}

func TestIsSensitiveKeyNamePrefixSuffix(t *testing.T) {
	// Prefix ignore cases
	if IsSensitiveKeyName("has_password") {
		t.Errorf("expected has_password to not be sensitive")
	}
	if IsSensitiveKeyName("is_authenticated") {
		t.Errorf("expected is_authenticated to not be sensitive")
	}

	// Suffix ignore cases
	if IsSensitiveKeyName("token_required") {
		t.Errorf("expected token_required to not be sensitive")
	}
	if IsSensitiveKeyName("pass_status") {
		t.Errorf("expected pass_status to not be sensitive")
	}
	if IsSensitiveKeyName("key_count") {
		t.Errorf("expected key_count to not be sensitive")
	}
	if IsSensitiveKeyName("auth_msg") {
		t.Errorf("expected auth_msg to not be sensitive")
	}
	if IsSensitiveKeyName("secret_message") {
		t.Errorf("expected secret_message to not be sensitive")
	}

	// Positive cases
	if !IsSensitiveKeyName("secret_key") {
		t.Errorf("expected secret_key to be sensitive")
	}
	if !IsSensitiveKeyName("webhooks") {
		t.Errorf("expected webhooks to be sensitive")
	}
	if !IsSensitiveKeyName("private_token") {
		t.Errorf("expected private_token to be sensitive")
	}

	// Non-sensitive
	if IsSensitiveKeyName("endpoint") {
		t.Errorf("expected endpoint to not be sensitive")
	}
}

func TestIsRedactableValue(t *testing.T) {
	var invalidVal reflect.Value
	if isRedactableValue(invalidVal) {
		t.Errorf("expected invalid reflect.Value to return false")
	}

	strVal := reflect.ValueOf("test")
	if !isRedactableValue(strVal) {
		t.Errorf("expected string to return true")
	}

	intVal := reflect.ValueOf(123)
	if isRedactableValue(intVal) {
		t.Errorf("expected int to return false")
	}
}

func TestRedactSecretsPointersAndNils(t *testing.T) {
	// Direct nil
	if res := RedactSecrets(nil); res != nil {
		t.Errorf("expected nil for nil input, got %v", res)
	}

	// Nil pointer
	var nilPtr *string
	if res := RedactSecrets(nilPtr); res != nil {
		t.Errorf("expected nil for nil pointer, got %v", res)
	}

	// Pointer to sensitive string
	secStr := "ghp_123456789012345678901234567890123456"
	if res := RedactSecrets(&secStr); res != RedactedString {
		t.Errorf("expected %s, got %v", RedactedString, res)
	}

	// Map with pointers and nil pointers
	type Inner struct {
		SecretKey string `json:"secret_key"`
	}
	var nilInner *Inner
	validInner := &Inner{SecretKey: "classified"}
	nestedMap := map[string]interface{}{
		"nil_inner":   nilInner,
		"valid_inner": validInner,
		"num":         99,
	}
	redMap := RedactSecrets(nestedMap).(map[string]interface{})
	if redMap["nil_inner"] != nil {
		t.Errorf("expected nil for nil_inner, got %v", redMap["nil_inner"])
	}
	innerRes := redMap["valid_inner"].(map[string]interface{})
	if innerRes["secret_key"] != RedactedString {
		t.Errorf("expected secret_key redacted, got %v", innerRes["secret_key"])
	}
}

func TestRedactSecretsTimeAndStructUnexported(t *testing.T) {
	now := time.Now()
	// Direct time.Time
	if res := RedactSecrets(now); res != now {
		t.Errorf("expected direct time.Time preserved, got %v", res)
	}

	// Struct with unexported fields, empty password, non-redactable sensitive key
	type ConfigItem struct {
		unexported string
		Password   string    `json:"password"`
		PassCount  int       `json:"pass_count_int"`
		Timestamp  time.Time `json:"timestamp"`
		SkipMe     string    `json:"-"`
	}

	item := ConfigItem{
		unexported: "private",
		Password:   "", // empty password should not be replaced with ********
		PassCount:  10,
		Timestamp:  now,
		SkipMe:     "skipped",
	}

	res := RedactSecrets(item).(map[string]interface{})
	if _, ok := res["unexported"]; ok {
		t.Errorf("expected unexported field to be omitted")
	}
	if res["password"] != "" {
		t.Errorf("expected empty password preserved, got %v", res["password"])
	}
	if res["timestamp"] != now {
		t.Errorf("expected timestamp preserved, got %v", res["timestamp"])
	}
}

func TestRedactSecretsInvalidRawMessage(t *testing.T) {
	// json.RawMessage that is not valid JSON
	invalidRaw := json.RawMessage(`{not-valid-json`)
	res := RedactSecrets(invalidRaw)
	if _, ok := res.(json.RawMessage); !ok {
		t.Errorf("expected fallback to raw message interface, got %T", res)
	}
}

