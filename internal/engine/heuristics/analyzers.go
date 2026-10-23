package heuristics

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// TempLockAnalyzer detects temporary, lock, or office swap files.
type TempLockAnalyzer struct{}

func (a *TempLockAnalyzer) Name() string { return "TempLockAnalyzer" }

func (a *TempLockAnalyzer) Analyze(_ context.Context, c Candidate) (models.RuleResult, error) {
	filename := filepath.Base(c.Path)
	lower := strings.ToLower(filename)

	isLockOrTemp := strings.HasPrefix(filename, "~$") ||
		strings.HasPrefix(filename, ".~lock.") ||
		strings.HasSuffix(lower, ".tmp") ||
		strings.HasSuffix(lower, ".swp") ||
		strings.HasSuffix(lower, ".lock") ||
		strings.HasSuffix(lower, ".bak") ||
		strings.HasSuffix(lower, ".old") ||
		lower == ".ds_store" ||
		lower == "desktop.ini" ||
		lower == "thumbs.db"

	if !isLockOrTemp {
		return models.RuleResult{RuleName: a.Name(), Matched: false}, nil
	}

	return models.RuleResult{
		RuleName: a.Name(),
		Action:   string(ActionKeepLocal),
		Score:    -0.8,
		Weight:   1.5,
		Risk:     string(RiskLow),
		Reason:   "Target matches application lock or temporary file pattern; preserving base document.",
		Matched:  true,
	}, nil
}

// HashMatchAnalyzer detects identical content hashes across local and remote.
type HashMatchAnalyzer struct{}

func (a *HashMatchAnalyzer) Name() string { return "HashMatchAnalyzer" }

func (a *HashMatchAnalyzer) Analyze(_ context.Context, c Candidate) (models.RuleResult, error) {
	if c.Local.Hash == "" || c.Remote.Hash == "" {
		return models.RuleResult{RuleName: a.Name(), Matched: false}, nil
	}

	if strings.EqualFold(c.Local.Hash, c.Remote.Hash) {
		return models.RuleResult{
			RuleName: a.Name(),
			Action:   string(ActionKeepLocal),
			Score:    -1.0,
			Weight:   2.0,
			Risk:     string(RiskLow),
			Reason:   "Content hashes are identical across endpoints despite timestamp differences.",
			Matched:  true,
		}, nil
	}

	return models.RuleResult{RuleName: a.Name(), Matched: false}, nil
}

// TruncationAnalyzer protects against 0-byte truncations or massive file shrinkage (>90%).
type TruncationAnalyzer struct{}

func (a *TruncationAnalyzer) Name() string { return "TruncationAnalyzer" }

func (a *TruncationAnalyzer) Analyze(_ context.Context, c Candidate) (models.RuleResult, error) {
	lSize := c.Local.Size
	rSize := c.Remote.Size

	// Check if one side is 0 bytes and the other is non-empty
	if lSize == 0 && rSize > 0 {
		return models.RuleResult{
			RuleName: a.Name(),
			Action:   string(ActionKeepRemote),
			Score:    1.0,
			Weight:   3.0,
			Risk:     string(RiskCritical),
			Reason:   "Potential 0-byte truncation on local endpoint. Remote non-empty file preserved.",
			Matched:  true,
		}, nil
	}

	if rSize == 0 && lSize > 0 {
		return models.RuleResult{
			RuleName: a.Name(),
			Action:   string(ActionKeepLocal),
			Score:    -1.0,
			Weight:   3.0,
			Risk:     string(RiskCritical),
			Reason:   "Potential 0-byte truncation on remote endpoint. Local non-empty file preserved.",
			Matched:  true,
		}, nil
	}

	// Check for severe size drop (>90% reduction from a valid file >= 512 bytes)
	if lSize >= 512 && rSize > 0 && float64(rSize)/float64(lSize) <= 0.10 {
		return models.RuleResult{
			RuleName: a.Name(),
			Action:   string(ActionKeepLocal),
			Score:    -1.0,
			Weight:   3.0,
			Risk:     string(RiskCritical),
			Reason:   fmt.Sprintf("Severe remote file size drop detected (%d bytes vs %d bytes local).", rSize, lSize),
			Matched:  true,
		}, nil
	}

	if rSize >= 512 && lSize > 0 && float64(lSize)/float64(rSize) <= 0.10 {
		return models.RuleResult{
			RuleName: a.Name(),
			Action:   string(ActionKeepRemote),
			Score:    1.0,
			Weight:   3.0,
			Risk:     string(RiskCritical),
			Reason:   fmt.Sprintf("Severe local file size drop detected (%d bytes vs %d bytes remote).", lSize, rSize),
			Matched:  true,
		}, nil
	}

	return models.RuleResult{RuleName: a.Name(), Matched: false}, nil
}
