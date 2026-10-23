package heuristics

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestEngine_TruncationCriticalRisk_LocalZero(t *testing.T) {
	eng := NewEngine()
	cand := Candidate{
		Path:   "documents/report.pdf",
		Local:  FileMeta{Size: 0, ModTime: time.Now()},
		Remote: FileMeta{Size: 5048576, ModTime: time.Now().Add(-10 * time.Minute)},
	}

	rec := eng.Evaluate(context.Background(), cand)

	if rec.Risk != string(RiskCritical) {
		t.Errorf("expected RiskCritical, got %s", rec.Risk)
	}
	if rec.SuggestedAction != string(ActionManual) {
		t.Errorf("expected ActionManual, got %s", rec.SuggestedAction)
	}
	if rec.Confidence < 0.90 {
		t.Errorf("expected high confidence override >= 0.90, got %f", rec.Confidence)
	}
}

func TestEngine_TruncationCriticalRisk_RemoteZero(t *testing.T) {
	eng := NewEngine()
	cand := Candidate{
		Path:   "projects/db.sqlite",
		Local:  FileMeta{Size: 1048576, ModTime: time.Now().Add(-time.Hour)},
		Remote: FileMeta{Size: 0, ModTime: time.Now()},
	}

	rec := eng.Evaluate(context.Background(), cand)

	if rec.Risk != string(RiskCritical) {
		t.Errorf("expected RiskCritical, got %s", rec.Risk)
	}
	if rec.SuggestedAction != string(ActionManual) {
		t.Errorf("expected ActionManual, got %s", rec.SuggestedAction)
	}
}

func TestEngine_TruncationCriticalRisk_SevereDrop(t *testing.T) {
	eng := NewEngine()
	cand := Candidate{
		Path:   "data/records.csv",
		Local:  FileMeta{Size: 10000, ModTime: time.Now()},
		Remote: FileMeta{Size: 50, ModTime: time.Now().Add(time.Minute)}, // dropped >90%
	}

	rec := eng.Evaluate(context.Background(), cand)

	if rec.Risk != string(RiskCritical) {
		t.Errorf("expected RiskCritical for severe shrinkage, got %s", rec.Risk)
	}
	if rec.SuggestedAction != string(ActionManual) {
		t.Errorf("expected ActionManual, got %s", rec.SuggestedAction)
	}
}

func TestEngine_TempLockFiles(t *testing.T) {
	eng := NewEngine()
	testCases := []string{
		"~$Financials.xlsx",
		".~lock.document.odt#",
		"session.tmp",
		"app.log.swp",
		"sync.lock",
		".DS_Store",
		"desktop.ini",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			cand := Candidate{
				Path:   tc,
				Local:  FileMeta{Size: 100, ModTime: time.Now()},
				Remote: FileMeta{Size: 100, ModTime: time.Now().Add(-time.Minute)},
			}
			rec := eng.Evaluate(context.Background(), cand)
			if rec.Risk != string(RiskLow) {
				t.Errorf("expected RiskLow for temp lock %s, got %s", tc, rec.Risk)
			}
			if rec.SuggestedAction != string(ActionKeepLocal) {
				t.Errorf("expected ActionKeepLocal for temp lock %s, got %s", tc, rec.SuggestedAction)
			}
		})
	}
}

func TestEngine_HashMatch(t *testing.T) {
	eng := NewEngine()
	cand := Candidate{
		Path:   "photos/vacation.jpg",
		Local:  FileMeta{Size: 2048, ModTime: time.Now(), Hash: "e3b0c44298fc1c149afbf4c8996fb924"},
		Remote: FileMeta{Size: 2048, ModTime: time.Now().Add(-2 * time.Hour), Hash: "E3B0C44298FC1C149AFBF4C8996FB924"},
	}

	rec := eng.Evaluate(context.Background(), cand)

	if rec.SuggestedAction != string(ActionKeepLocal) {
		t.Errorf("expected ActionKeepLocal for identical hashes, got %s", rec.SuggestedAction)
	}
	if rec.Confidence < 0.70 {
		t.Errorf("expected high confidence >= 0.70 for identical hash, got %f", rec.Confidence)
	}
	if rec.Risk != string(RiskLow) {
		t.Errorf("expected RiskLow for identical hash, got %s", rec.Risk)
	}
}

func TestEngine_ClockSkewDST(t *testing.T) {
	eng := NewEngine()
	now := time.Now()

	// Exactly 1 hour delta (3600s) with identical size
	cand1 := Candidate{
		Path:   "docs/readme.txt",
		Local:  FileMeta{Size: 500, ModTime: now},
		Remote: FileMeta{Size: 500, ModTime: now.Add(-3600 * time.Second)},
	}
	rec1 := eng.Evaluate(context.Background(), cand1)
	if rec1.SuggestedAction != string(ActionKeepLocal) {
		t.Errorf("expected ActionKeepLocal for 1h DST jump, got %s", rec1.SuggestedAction)
	}

	// Micro-drift (1.5s)
	cand2 := Candidate{
		Path:   "docs/readme.txt",
		Local:  FileMeta{Size: 500, ModTime: now},
		Remote: FileMeta{Size: 500, ModTime: now.Add(-1500 * time.Millisecond)},
	}
	rec2 := eng.Evaluate(context.Background(), cand2)
	if rec2.SuggestedAction != string(ActionKeepLocal) {
		t.Errorf("expected ActionKeepLocal for micro-drift, got %s", rec2.SuggestedAction)
	}
}

func TestEngine_SizeDivergence(t *testing.T) {
	eng := NewEngine()
	cand := Candidate{
		Path:   "backups/archive.zip",
		Local:  FileMeta{Size: 1000000, ModTime: time.Now()},
		Remote: FileMeta{Size: 1500000, ModTime: time.Now().Add(-time.Hour)}, // 33% difference
	}

	rec := eng.Evaluate(context.Background(), cand)

	if rec.Risk != string(RiskHigh) {
		t.Errorf("expected RiskHigh for binary divergence, got %s", rec.Risk)
	}
}

func TestEngine_TimestampRecency(t *testing.T) {
	eng := NewEngine()
	now := time.Now()

	// Remote file 2 hours newer
	cand := Candidate{
		Path:   "notes/meeting.txt",
		Local:  FileMeta{Size: 100, ModTime: now.Add(-2 * time.Hour)},
		Remote: FileMeta{Size: 120, ModTime: now},
	}
	rec := eng.Evaluate(context.Background(), cand)
	if rec.SuggestedAction != string(ActionKeepRemote) {
		t.Errorf("expected ActionKeepRemote for newer remote, got %s", rec.SuggestedAction)
	}
	if rec.Confidence < 0.40 {
		t.Errorf("expected confidence >= 0.40, got %f", rec.Confidence)
	}
}

func TestEngine_EvaluateBatch_Concurrency(t *testing.T) {
	eng := NewEngine()
	now := time.Now()

	var candidates []Candidate
	for i := 0; i < 50; i++ {
		candidates = append(candidates, Candidate{
			Path:   fmt.Sprintf("folder/file_%d.txt", i),
			Local:  FileMeta{Size: int64(100 + i), ModTime: now.Add(time.Duration(i) * time.Minute)},
			Remote: FileMeta{Size: int64(100 + i), ModTime: now},
		})
	}

	results := eng.EvaluateBatch(context.Background(), candidates)

	if len(results) != len(candidates) {
		t.Fatalf("expected %d evaluated candidates, got %d", len(candidates), len(results))
	}

	for i := 0; i < 50; i++ {
		path := fmt.Sprintf("folder/file_%d.txt", i)
		rec, ok := results[path]
		if !ok {
			t.Errorf("missing path %s in results", path)
		}
		if rec.SuggestedAction == "" {
			t.Errorf("empty action for path %s", path)
		}
	}
}

func TestEngine_FormatHumanDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{30 * time.Second, "30 seconds"},
		{5 * time.Minute, "5 minutes"},
		{2 * time.Hour, "2 hours"},
		{2*time.Hour + 30*time.Minute, "2h 30m"},
		{48 * time.Hour, "2 days"},
	}

	for _, tt := range tests {
		res := formatHumanDuration(tt.d)
		if res != tt.expected {
			t.Errorf("for duration %v expected %s, got %s", tt.d, tt.expected, res)
		}
	}
}

func TestEngine_NoAnalyzersFallback(t *testing.T) {
	eng := NewEngine([]Analyzer{}...)
	cand := Candidate{
		Path: "test.txt",
	}
	rec := eng.Evaluate(context.Background(), cand)
	if rec.SuggestedAction != string(ActionKeepBoth) {
		t.Errorf("expected fallback ActionKeepBoth, got %s", rec.SuggestedAction)
	}
}
