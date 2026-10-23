package heuristics

import (
	"context"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/models"
)

// Action aliases models.ActionDecision for consistent review decisions.
type Action = models.ActionDecision

const (
	ActionKeepLocal  Action = models.DecisionKeepLocal
	ActionKeepRemote Action = models.DecisionKeepRemote
	ActionKeepBoth   Action = models.DecisionKeepBoth
	ActionManual     Action = "manual-review"
)

// RiskLevel defines the safety risk associated with automated decision application.
type RiskLevel string

const (
	RiskLow      RiskLevel = "LOW"
	RiskMedium   RiskLevel = "MEDIUM"
	RiskHigh     RiskLevel = "HIGH"
	RiskCritical RiskLevel = "CRITICAL"
)

// FileMeta captures endpoint file properties for conflict analysis.
type FileMeta struct {
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	Hash     string    `json:"hash"`
	MimeType string    `json:"mime_type"`
}

// Candidate represents a conflicting file pair across local and remote endpoints.
type Candidate struct {
	Path   string   `json:"path"`
	Local  FileMeta `json:"local"`
	Remote FileMeta `json:"remote"`
}

// Analyzer defines the contract implemented by each heuristic rule module.
type Analyzer interface {
	Name() string
	Analyze(ctx context.Context, c Candidate) (models.RuleResult, error)
}
