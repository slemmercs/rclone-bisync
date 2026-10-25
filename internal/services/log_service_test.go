package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestLogService_HardenedEdgeCases(t *testing.T) {
	svc := NewLogService()
	tmpDir := t.TempDir()

	// 1. Open error on unreadable file
	unreadableFile := filepath.Join(tmpDir, "unreadable.log")
	_ = os.WriteFile(unreadableFile, []byte("data"), 0000)
	defer os.Chmod(unreadableFile, 0644)

	// Attempting to read unreadable file returns error
	_, err := svc.FilterDaemonLogs(unreadableFile, time.Time{}, time.Time{}, 10)
	if err == nil {
		t.Fatal("expected error when file is unreadable")
	}

	// 2. Scanner buffer overflow error (> 512KB line)
	hugeFile := filepath.Join(tmpDir, "huge.log")
	hugeLine := strings.Repeat("A", 600*1024) + "\n"
	if err := os.WriteFile(hugeFile, []byte(hugeLine), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = svc.FilterDaemonLogs(hugeFile, time.Time{}, time.Time{}, 10)
	if err == nil {
		t.Fatal("expected scanner error for line exceeding 512KB buffer")
	}

	// 3. Nested log without inner time, falling back to outer wrapper line time
	baseTime := time.Date(2026, 10, 17, 20, 0, 0, 0, time.UTC)
	wrapTimeStr := baseTime.Add(2 * time.Second).Format(time.RFC3339Nano)
	nestedNoTime := filepath.Join(tmpDir, "nested-notime.log")

	innerNoTime := `{"level":"info","msg":"nested without timestamp"}`
	wrapperBytes, _ := json.Marshal(map[string]interface{}{
		"level": "info",
		"time":  wrapTimeStr,
		"msg":   innerNoTime,
	})
	_ = os.WriteFile(nestedNoTime, append(wrapperBytes, '\n'), 0644)

	// Filter with window matching outer wrapper time
	matched, err := svc.FilterDaemonLogs(nestedNoTime, baseTime, baseTime.Add(5*time.Second), 1)
	if err != nil {
		t.Fatalf("unexpected filter error: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("expected 1 matched nested line using outer timestamp, got %d", len(matched))
	}

	// Filter with window outside outer wrapper time
	matchedOut, err := svc.FilterDaemonLogs(nestedNoTime, baseTime.Add(10*time.Second), baseTime.Add(20*time.Second), 10)
	if err != nil {
		t.Fatalf("unexpected filter error: %v", err)
	}
	if len(matchedOut) != 0 {
		t.Fatalf("expected 0 matched lines when outer timestamp is outside window, got %d", len(matchedOut))
	}

	// 4. Test window matching with limit clamping during accumulation
	multiLines := filepath.Join(tmpDir, "multi.log")
	l1 := fmt.Sprintf(`{"level":"info","time":"%s","msg":"msg 1"}`, baseTime.Add(1*time.Second).Format(time.RFC3339Nano))
	l2 := fmt.Sprintf(`{"level":"info","time":"%s","msg":"msg 2"}`, baseTime.Add(2*time.Second).Format(time.RFC3339Nano))
	l3 := fmt.Sprintf(`{"level":"info","time":"%s","msg":"msg 3"}`, baseTime.Add(3*time.Second).Format(time.RFC3339Nano))
	_ = os.WriteFile(multiLines, []byte(fmt.Sprintf("%s\n%s\n%s\n", l1, l2, l3)), 0644)

	// Limit 2 with 3 matching lines inside window
	limited, err := svc.FilterDaemonLogs(multiLines, baseTime, baseTime.Add(5*time.Second), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("expected limit of 2 lines, got %d", len(limited))
	}

	// Also limit with nested lines inside window
	nestedMulti := filepath.Join(tmpDir, "nested-multi.log")
	nestedWrapped := fmt.Sprintf(`{"level":"info","time":"%s","msg":"{\"time\":\"%s\",\"msg\":\"n1\"}\n{\"time\":\"%s\",\"msg\":\"n2\"}\n{\"time\":\"%s\",\"msg\":\"n3\"}"}`,
		baseTime.Format(time.RFC3339Nano),
		baseTime.Add(1*time.Second).Format(time.RFC3339Nano),
		baseTime.Add(2*time.Second).Format(time.RFC3339Nano),
		baseTime.Add(3*time.Second).Format(time.RFC3339Nano),
	)
	_ = os.WriteFile(nestedMulti, []byte(nestedWrapped+"\n"), 0644)
	limitedNested, err := svc.FilterDaemonLogs(nestedMulti, baseTime, baseTime.Add(5*time.Second), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
		// Also test limit overflow for subEntry using wrapper timestamp
		nestedNoTimeMulti := filepath.Join(tmpDir, "nested-notime-multi.log")
		innerNoTimeMulti := fmt.Sprintf("%s\n%s\n", innerNoTime, innerNoTime)
		wrapperMultiBytes, _ := json.Marshal(map[string]interface{}{
			"level": "info",
			"time":  wrapTimeStr,
			"msg":   innerNoTimeMulti,
		})
		_ = os.WriteFile(nestedNoTimeMulti, append(wrapperMultiBytes, '\n'), 0644)
		matchedLimit, err := svc.FilterDaemonLogs(nestedNoTimeMulti, baseTime, baseTime.Add(5*time.Second), 1)
		if err != nil || len(matchedLimit) != 1 {
			t.Fatalf("expected 1 clamped nested line using outer timestamp, got %d, err: %v", len(matchedLimit), err)
		}

	}
	if len(limitedNested) != 2 {
		t.Fatalf("expected nested limit of 2 lines, got %d", len(limitedNested))
	}
}

func TestExtractNestedJSONLogs_EdgeCasesAndFallbacks(t *testing.T) {
	// 1. Invalid JSON starting with {
	_, ok := extractNestedJSONLogs("{broken json line")
	if ok {
		t.Fatal("expected false for broken JSON")
	}

	// 2. Non-string msg field
	_, ok = extractNestedJSONLogs(`{"msg": 12345}`)
	if ok {
		t.Fatal("expected false when msg is an int")
	}

	// 3. msg with non-JSON text followed by line-by-line valid JSON (triggers fallback parser)
	msgWithPreamble := "Log header preamble text\n{\"level\":\"info\",\"msg\":\"valid inner line\"}\nFooter summary"
	encodedMsg, _ := json.Marshal(map[string]interface{}{
		"level": "info",
		"msg":   msgWithPreamble,
	})
	entries, ok := extractNestedJSONLogs(string(encodedMsg))
	if !ok || len(entries) != 1 {
		t.Fatalf("expected line-by-line fallback to extract 1 valid entry, got ok=%v, count=%d", ok, len(entries))
		// 5. msg contains {" but has no valid inner JSON objects (reaches end fallback)
		msgBrokenJSON := `Header text {" unclosed syntax`
		encodedBroken, _ := json.Marshal(map[string]interface{}{
			"msg": msgBrokenJSON,
		})
		_, ok = extractNestedJSONLogs(string(encodedBroken))
		if ok {
			t.Fatal("expected false for unclosed JSON in msg")
		}

	}

	// 4. Line-by-line fallback where no line is valid JSON
	msgNoJSON := "First line text\nSecond line text\nThird line text"
	encodedNoJSON, _ := json.Marshal(map[string]interface{}{
		"msg": msgNoJSON,
	})
	_, ok = extractNestedJSONLogs(string(encodedNoJSON))
	if ok {
		t.Fatal("expected false when no lines contain JSON")
	}
}

func TestParseLogTime_AdditionalFormats(t *testing.T) {
	// Dash format
	dashTime := "2026-10-17 20:45:00 [INFO] test log"
	parsed, ok := parseLogTime(dashTime)
	if !ok || parsed.Hour() != 20 {
		t.Fatalf("expected valid parse for dash format, got %v, ok=%v", parsed, ok)
	}

	// Length >= 19 but invalid time characters
	invalidPrefix := "This is a random 19-char string"
	_, ok = parseLogTime(invalidPrefix)
	if ok {
		t.Fatal("expected false for arbitrary 19-char prefix")
	}

	// JSON with empty time field
	emptyTimeJSON := `{"time":"","msg":"something"}`
	_, ok = parseLogTime(emptyTimeJSON)
	if ok {
		t.Fatal("expected false for empty time in JSON")
	}
}
