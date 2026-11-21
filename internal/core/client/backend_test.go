package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBackendCommand_Validation(t *testing.T) {
	ctx := context.Background()

	// 1. Nil client
	var nilClient *RcloneClient
	if _, err := nilClient.BackendCommand(ctx, "myremote:", "dump", nil, nil); err == nil {
		t.Fatal("expected error on nil client, got nil")
	}

	c := NewRcloneClient("http://localhost:5572", "", "")

	// 2. Empty fs
	if _, err := c.BackendCommand(ctx, "", "dump", nil, nil); err == nil {
		t.Fatal("expected error on empty fs, got nil")
	}

	// 3. Empty command
	if _, err := c.BackendCommand(ctx, "myremote:", "", nil, nil); err == nil {
		t.Fatal("expected error on empty command, got nil")
	}
}

func TestBackendCommand_SuccessAndPayload(t *testing.T) {
	ctx := context.Background()

	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/backend/command" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)

		cmd, _ := receivedPayload["command"].(string)
		switch cmd {
		case "dump":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"result": map[string]interface{}{
					"count": 42,
					"files": []string{"file1.txt", "file2.txt"},
				},
			})
		case "drop":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"result": "cache dropped successfully",
			})
		case "raw":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ok",
				"items":  []string{"a", "b"},
			})
		case "fail":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "invalid command arguments"}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	c := NewRcloneClient(server.URL, "", "")

	// 1. Test "dump" command with result unmarshaling
	res, err := c.BackendCommand(ctx, "hasher_remote:", "dump", []string{"arg1"}, map[string]string{"opt1": "val1"})
	if err != nil {
		t.Fatalf("unexpected error executing dump: %v", err)
	}

	if receivedPayload["fs"] != "hasher_remote:" {
		t.Fatalf("expected fs 'hasher_remote:', got %v", receivedPayload["fs"])
	}
	if receivedPayload["command"] != "dump" {
		t.Fatalf("expected command 'dump', got %v", receivedPayload["command"])
	}
	resMap, ok := res.(map[string]interface{})
	if !ok || resMap["count"].(float64) != 42 {
		t.Fatalf("expected result map with count 42, got %v", res)
	}

	// 2. Test "drop" command with string result
	resDrop, err := c.BackendCommand(ctx, "hasher_remote", "drop", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error executing drop: %v", err)
	}
	if resDrop != "cache dropped successfully" {
		t.Fatalf("expected 'cache dropped successfully', got %v", resDrop)
	}
	if receivedPayload["fs"] != "hasher_remote:" {
		t.Fatalf("expected NormalizeFs to append colon, got %v", receivedPayload["fs"])
	}

	// 3. Test command returning payload without explicit "result" key
	resRaw, err := c.BackendCommand(ctx, "hasher_remote:", "raw", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error executing raw: %v", err)
	}
	rawMap, ok := resRaw.(map[string]interface{})
	if !ok || rawMap["status"] != "ok" {
		t.Fatalf("expected raw payload containing status: ok, got %v", resRaw)
	}

	// 4. Test error handling
	_, err = c.BackendCommand(ctx, "hasher_remote:", "fail", nil, nil)
	if err == nil {
		t.Fatal("expected error on failure command, got nil")
	}
}
