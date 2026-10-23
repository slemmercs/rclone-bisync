package heuristics

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// ClockSkewAnalyzer detects exact 1-hour/2-hour DST jumps or micro filesystem drift (<=2s).
type ClockSkewAnalyzer struct{}

func (a *ClockSkewAnalyzer) Name() string { return "ClockSkewAnalyzer" }

func (a *ClockSkewAnalyzer) Analyze(_ context.Context, c Candidate) (models.RuleResult, error) {
	if c.Local.ModTime.IsZero() || c.Remote.ModTime.IsZero() {
		return models.RuleResult{RuleName: a.Name(), Matched: false}, nil
	}

	// Clock skew analysis is only relevant when sizes are identical or near-identical
	if c.Local.Size != c.Remote.Size {
		return models.RuleResult{RuleName: a.Name(), Matched: false}, nil
	}

	delta := math.Abs(c.Local.ModTime.Sub(c.Remote.ModTime).Seconds())

	// FAT32 / filesystem timestamp micro-drift <= 2s
	if delta <= 2.0 && delta > 0.0 {
		return models.RuleResult{
			RuleName: a.Name(),
			Action:   string(ActionKeepLocal),
			Score:    -0.8,
			Weight:   1.2,
			Risk:     string(RiskLow),
			Reason:   "Timestamp delta is within 2-second filesystem clock drift threshold with identical sizes.",
			Matched:  true,
		}, nil
	}

	// 1-hour Daylight Saving Time shift (3600s +- 2s)
	if math.Abs(delta-3600.0) <= 2.0 {
		return models.RuleResult{
			RuleName: a.Name(),
			Action:   string(ActionKeepLocal),
			Score:    -0.8,
			Weight:   1.2,
			Risk:     string(RiskLow),
			Reason:   "Timestamp delta matches exact 1-hour Daylight Saving Time boundary with identical file sizes.",
			Matched:  true,
		}, nil
	}

	// 2-hour Daylight Saving Time shift (7200s +- 2s)
	if math.Abs(delta-7200.0) <= 2.0 {
		return models.RuleResult{
			RuleName: a.Name(),
			Action:   string(ActionKeepLocal),
			Score:    -0.8,
			Weight:   1.2,
			Risk:     string(RiskLow),
			Reason:   "Timestamp delta matches exact 2-hour Daylight Saving Time boundary with identical file sizes.",
			Matched:  true,
		}, nil
	}

	return models.RuleResult{RuleName: a.Name(), Matched: false}, nil
}

// SizeDivergenceAnalyzer flags high divergence in binary file sizes (>20%).
type SizeDivergenceAnalyzer struct{}

func (a *SizeDivergenceAnalyzer) Name() string { return "SizeDivergenceAnalyzer" }

func (a *SizeDivergenceAnalyzer) Analyze(_ context.Context, c Candidate) (models.RuleResult, error) {
	ext := strings.ToLower(filepath.Ext(c.Path))
	binaryExts := map[string]bool{
		".sqlite": true, ".db": true, ".zip": true, ".tar": true,
		".gz": true, ".7z": true, ".iso": true, ".psd": true,
		".ai": true, ".mp4": true, ".mov": true, ".docx": true,
		".xlsx": true, ".pptx": true, ".pdf": true,
	}

	if !binaryExts[ext] {
		return models.RuleResult{RuleName: a.Name(), Matched: false}, nil
	}

	lSize := c.Local.Size
	rSize := c.Remote.Size

	if lSize <= 0 || rSize <= 0 {
		return models.RuleResult{RuleName: a.Name(), Matched: false}, nil
	}

	maxSize := math.Max(float64(lSize), float64(rSize))
	diff := math.Abs(float64(lSize) - float64(rSize))
	ratio := diff / maxSize

	if ratio > 0.20 {
		pct := int(ratio * 100)
		return models.RuleResult{
			RuleName: a.Name(),
			Action:   string(ActionManual),
			Score:    0.0,
			Weight:   1.5,
			Risk:     string(RiskHigh),
			Reason:   fmt.Sprintf("Significant binary size divergence detected (%d%% difference in %s). Manual review required.", pct, ext),
			Matched:  true,
		}, nil
	}

	return models.RuleResult{RuleName: a.Name(), Matched: false}, nil
}

// TimestampRecencyAnalyzer evaluates modification timestamp recency deltas.
type TimestampRecencyAnalyzer struct{}

func (a *TimestampRecencyAnalyzer) Name() string { return "TimestampRecencyAnalyzer" }

func (a *TimestampRecencyAnalyzer) Analyze(_ context.Context, c Candidate) (models.RuleResult, error) {
	if c.Local.ModTime.IsZero() || c.Remote.ModTime.IsZero() {
		return models.RuleResult{RuleName: a.Name(), Matched: false}, nil
	}

	delta := c.Local.ModTime.Sub(c.Remote.ModTime)
	sec := delta.Seconds()

	if math.Abs(sec) <= 1.0 {
		return models.RuleResult{
			RuleName: a.Name(),
			Action:   string(ActionKeepBoth),
			Score:    0.0,
			Weight:   0.5,
			Risk:     string(RiskLow),
			Reason:   "Modification times are nearly identical across endpoints.",
			Matched:  true,
		}, nil
	}

	if sec > 0 {
		durStr := formatHumanDuration(delta)
		risk := string(RiskLow)
		if delta < 5*time.Minute {
			risk = string(RiskMedium)
		}
		return models.RuleResult{
			RuleName: a.Name(),
			Action:   string(ActionKeepLocal),
			Score:    -0.8,
			Weight:   1.0,
			Risk:     risk,
			Reason:   fmt.Sprintf("Local file is newer by %s.", durStr),
			Matched:  true,
		}, nil
	}

	absDelta := -delta
	durStr := formatHumanDuration(absDelta)
	risk := string(RiskLow)
	if absDelta < 5*time.Minute {
		risk = string(RiskMedium)
	}
	return models.RuleResult{
		RuleName: a.Name(),
		Action:   string(ActionKeepRemote),
		Score:    0.8,
		Weight:   1.0,
		Risk:     risk,
		Reason:   fmt.Sprintf("Remote file is newer by %s.", durStr),
		Matched:  true,
	}, nil
}

func formatHumanDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		if mins == 0 {
			return fmt.Sprintf("%d hours", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	days := int(d.Hours()) / 24
	return fmt.Sprintf("%d days", days)
}
