package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogService_FilterDaemonLogs(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "rclone-daemon.log")

	svc := NewLogService()

	// 1. Missing file
	logs, err := svc.FilterDaemonLogs(filepath.Join(tmpDir, "missing.log"), time.Time{}, time.Time{}, 100)
	if err != nil || len(logs) != 0 {
		t.Fatalf("expected empty slice and nil error for missing file: %v, %v", logs, err)
	}

	// 2. Prepare test data
	baseTime := time.Date(2026, 10, 17, 20, 0, 0, 0, time.UTC)
	lines := []string{
		fmt.Sprintf(`{"level":"info","msg":"before window","time":"%s"}`, baseTime.Add(-10*time.Second).Format(time.RFC3339Nano)),
		fmt.Sprintf(`{"level":"info","msg":"start of window","time":"%s"}`, baseTime.Format(time.RFC3339Nano)),
		fmt.Sprintf(`{"level":"info","msg":"middle of window","time":"%s"}`, baseTime.Add(5*time.Second).Format(time.RFC3339)),
		fmt.Sprintf(`{"level":"error","msg":"transfer error","time":"%s"}`, baseTime.Add(10*time.Second).Format(time.RFC3339Nano)),
		`{"level":"notice","msg":"corrupt line with no time"}`,
		"",
		`2026-10-17 20:00:10 [ERROR] legacy error inside window`,
		`invalid random non-json line`,
		fmt.Sprintf(`{"level":"info","msg":"after window","time":"%s"}`, baseTime.Add(30*time.Second).Format(time.RFC3339Nano)),
	}

	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Filter with window [baseTime, baseTime + 10s]
	// Window will be [baseTime-1s, baseTime+11s]
	matched, err := svc.FilterDaemonLogs(logPath, baseTime, baseTime.Add(10*time.Second), 100)
	if err != nil {
		t.Fatalf("unexpected filter error: %v", err)
	}
	// Expected lines: start of window, middle of window, transfer error, and the legacy line at 20:00:10
	if len(matched) != 4 {
		t.Fatalf("expected 4 matched lines within window, got %d: %v", len(matched), matched)
	}

	// 4. Zero startTime: tail mode
	tail, err := svc.FilterDaemonLogs(logPath, time.Time{}, time.Time{}, 3)
	if err != nil {
		t.Fatalf("unexpected tail error: %v", err)
	}
	if len(tail) != 3 {
		t.Fatalf("expected 3 tail lines, got %d", len(tail))
	}

	// 5. Zero endTime (window to now)
	nowWindow, err := svc.FilterDaemonLogs(logPath, baseTime.Add(-20*time.Second), time.Time{}, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nowWindow) == 0 {
		t.Fatal("expected matched lines when endTime is zero")
	}

	// 6. Limit clamping (<0 and >2000)
	clamped, err := svc.FilterDaemonLogs(logPath, time.Time{}, time.Time{}, -10)
	if err != nil {
		t.Fatalf("unexpected error with negative limit: %v", err)
	}
	if len(clamped) == 0 {
		t.Fatal("expected fallback to default limit")
	}

	clampedMax, err := svc.FilterDaemonLogs(logPath, time.Time{}, time.Time{}, 5000)
	if err != nil {
		t.Fatalf("unexpected error with large limit: %v", err)
	}
	if len(clampedMax) == 0 {
		t.Fatal("expected max clamped limit")
	}

	// 7. Test nested JSON unwrapping (bisync RC log wrapper pattern)
	nestedLogPath := filepath.Join(tmpDir, "nested-daemon.log")
	t1 := baseTime.Add(1 * time.Second).Format(time.RFC3339Nano)
	t2 := baseTime.Add(2 * time.Second).Format(time.RFC3339Nano)
	tOut := baseTime.Add(100 * time.Second).Format(time.RFC3339Nano)

	inner1 := fmt.Sprintf(`{"level":"info","msg":"nested event 1","time":"%s"}`, t1)
	inner2 := fmt.Sprintf(`{"level":"notice","msg":"Duplicate object found","object":"pic.jpg","objectType":"*drive.Object","time":"%s"}`, t2)
	innerOutOfWindow := fmt.Sprintf(`{"level":"info","msg":"late event","time":"%s"}`, tOut)

	// Combine into wrapper line
	rawMsg := fmt.Sprintf("%s\n%s\n%s\n", inner1, inner2, innerOutOfWindow)
	wrapperLine, err := json.Marshal(map[string]interface{}{
		"level":  "notice",
		"msg":    rawMsg,
		"source": "bisync/rc.go:186",
		"time":   t2,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(nestedLogPath, append(wrapperLine, '\n'), 0644); err != nil {
		t.Fatal(err)
	}

	// Window [baseTime, baseTime + 5s] should match inner1 and inner2, but not innerOutOfWindow
	unwrapped, err := svc.FilterDaemonLogs(nestedLogPath, baseTime, baseTime.Add(5*time.Second), 100)
	if err != nil {
		t.Fatalf("unexpected error filtering nested logs: %v", err)
	}
	if len(unwrapped) != 2 {
		t.Fatalf("expected 2 unwrapped inner entries matching time window, got %d: %v", len(unwrapped), unwrapped)
	}

	// Tail mode on nested log
	tailNested, err := svc.FilterDaemonLogs(nestedLogPath, time.Time{}, time.Time{}, 2)
	if err != nil {
		t.Fatalf("unexpected error in tail mode: %v", err)
	}
	if len(tailNested) != 2 {
		t.Fatalf("expected 2 clamped tail entries, got %d", len(tailNested))
	}
}

func TestExtractNestedJSONLogs(t *testing.T) {
	// Not a JSON line
	_, ok := extractNestedJSONLogs("plain text")
	if ok {
		t.Fatal("expected false for plain text")
	}

	// JSON line without nested JSON in msg
	simpleJSON := `{"level":"info","msg":"regular message"}`
	_, ok = extractNestedJSONLogs(simpleJSON)
	if ok {
		t.Fatal("expected false for regular JSON message")
	}

	// JSON with nested sub-lines
	nestedJSON := `{"level":"notice","msg":"{\"level\":\"info\",\"msg\":\"sub1\"}\n{\"level\":\"error\",\"msg\":\"sub2\"}"}`
	entries, ok := extractNestedJSONLogs(nestedJSON)
	if !ok || len(entries) != 2 {
		t.Fatalf("expected 2 extracted entries, got ok=%v, count=%d", ok, len(entries))
	}
}

func TestParseLogTime(t *testing.T) {
	// JSON RFC3339
	jsonLine := `{"time":"2026-10-17T20:45:00Z","msg":"test"}`
	ts, ok := parseLogTime(jsonLine)
	if !ok || ts.Year() != 2026 {
		t.Fatalf("failed to parse valid JSON time: %v, ok=%v", ts, ok)
	}

	// Legacy slash format
	legacySlash := "2026/10/17 20:45:00 NOTICE: test"
	ts, ok = parseLogTime(legacySlash)
	if !ok || ts.Minute() != 45 {
		t.Fatalf("failed to parse legacy slash time: %v, ok=%v", ts, ok)
	}

	// Corrupt
	_, ok = parseLogTime("not a time at all")
	if ok {
		t.Fatal("expected false for invalid time")
	}
}
