package app

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/slemmercs/rclone-bisync/internal/audit"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/core/client"
	"github.com/slemmercs/rclone-bisync/internal/core/logger"
	"github.com/slemmercs/rclone-bisync/internal/engine/runner"
	"github.com/slemmercs/rclone-bisync/internal/profiles"
	"github.com/slemmercs/rclone-bisync/internal/remoteauth"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

// Container manages dependency injection, service instantiation, and lifecycle hooks for bisync-hub.
type Container struct {
	mu               sync.Mutex
	Config           *config.Config
	Client           *client.RcloneClient
	SyncService      *services.SyncService
	TelemetryService *services.TelemetryService
	CanaryService    *services.CanaryService
	DeletionService  *services.DeletionService
	ExplorerService  *services.ExplorerService
	ReviewService    *services.ReviewService
	AuditService     *services.AuditService
	LogService       *services.LogService
	AuthService      *remoteauth.RemoteAuthService
	ProfileStore     *profiles.Store
	ProfileScheduler *profiles.Scheduler
	ProfileService   *profiles.Service
	ProfileExecutor  profiles.SyncExecutor
	started          bool
	stopped          bool
}

// Option configures a Container instance during initialization.
type Option func(*Container)

// WithClient overrides the default Rclone RC client.
func WithClient(c *client.RcloneClient) Option {
	return func(ctr *Container) {
		ctr.Client = c
	}
}

// WithSyncService overrides the SyncService and pre-populates services if provided.
func WithSyncService(s *services.SyncService) Option {
	return func(ctr *Container) {
		ctr.SyncService = s
	}
}

// WithProfileExecutor overrides the profile sync executor.
func WithProfileExecutor(exec profiles.SyncExecutor) Option {
	return func(ctr *Container) {
		ctr.ProfileExecutor = exec
	}
}

// DefaultProfileExecutor returns a SyncExecutor wiring profile parameters to SyncRunner.
func DefaultProfileExecutor(cfg *config.Config) profiles.SyncExecutor {
	return func(ctx context.Context, p *profiles.Profile, opts profiles.SyncOptions) error {
		// #nosec G301 - local profile directory creation
		if err := os.MkdirAll(p.LocalPath, 0750); err != nil {
			return fmt.Errorf("failed to create local path %s: %w", p.LocalPath, err)
		}
		profCfg := *cfg
		profCfg.ProfileID = p.ID
		profCfg.ProfileName = p.Name
		profCfg.RemotePath = p.FullRemoteTarget()
		profCfg.LocalPath = p.LocalPath
		profCfg.LockFile = p.LockFilePath()

		// Build operational arguments from 3-tier flag hierarchy
		cliArgs, err := profiles.BuildCLIArgs(p, cfg.GlobalDefaults, &profiles.ExecutionOverrides{
			DryRun: &opts.DryRun,
			Resync: &opts.Resync,
			Force:  &opts.Force,
		})
		if err != nil {
			return fmt.Errorf("failed to build CLI arguments for profile %s: %w", p.ID, err)
		}

		commonArgs := append([]string{}, profCfg.RcloneCommonArgs...)
		commonArgs = append(commonArgs, cliArgs...)
		profCfg.RcloneCommonArgs = commonArgs

		syncRunner := runner.NewSyncRunner(&profCfg)
		origin := opts.Origin
		if origin == "" {
			if opts.Force || opts.Resync || opts.DryRun {
				origin = "dashboard"
			} else {
				origin = "scheduler"
			}
		}
		success := syncRunner.Run(ctx, opts.Resync, opts.Force, opts.DryRun, origin, "", false)
		if !success {
			return fmt.Errorf("sync failed for profile %s", p.ID)
		}
		return nil
	}
}

// NewContainer builds a centralized dependency container from configuration and options.
func NewContainer(cfg *config.Config, opts ...Option) (*Container, error) {
	if cfg == nil {
		cfg = &config.Config{}
	}

	c := &Container{
		Config: cfg,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	// 1. Unpack SyncService dependencies if passed as option
	if c.SyncService != nil {
		if c.Client == nil && c.SyncService.Client != nil {
			c.Client = c.SyncService.Client
		}
		if c.TelemetryService == nil {
			c.TelemetryService = c.SyncService.Telemetry
		}
		if c.CanaryService == nil {
			c.CanaryService = c.SyncService.Canary
		}
		if c.DeletionService == nil {
			c.DeletionService = c.SyncService.Deletion
		}
		if c.ExplorerService == nil {
			c.ExplorerService = c.SyncService.Explorer
		}
		if c.ReviewService == nil {
			c.ReviewService = c.SyncService.Review
		}
		if c.AuditService == nil {
			c.AuditService = c.SyncService.Audit
		}
	}

	// 2. Initialize Rclone RC client if needed
	if c.Client == nil && cfg.RCAPIURL != "" {
		user := ""
		pass := ""
		if cfg.RCAuthEnabled {
			user = cfg.RCUser
			pass = cfg.RCPass
		}
		c.Client = client.NewRcloneClient(cfg.RCAPIURL, user, pass)
	}

	// 3. Initialize Core Domain Services
	if c.CanaryService == nil {
		c.CanaryService = services.NewCanaryService(cfg, c.Client)
	}
	if c.DeletionService == nil {
		c.DeletionService = services.NewDeletionService(cfg)
	}
	if c.TelemetryService == nil {
		c.TelemetryService = services.NewTelemetryService(cfg, c.Client)
	}
	if c.ExplorerService == nil {
		c.ExplorerService = services.NewExplorerService(cfg, c.Client, c.CanaryService)
	}
	if c.ReviewService == nil {
		c.ReviewService = services.NewReviewService(cfg, c.Client)
	}

	// 4. Initialize Audit Service
	if c.AuditService == nil && cfg.AuditDBPath != "" {
		if db, err := audit.OpenDB(cfg.AuditDBPath); err == nil && db != nil {
			var runLogger *logger.Logger
			if cfg.LogFile != "" {
				runLogger, _ = logger.New(cfg.LogFile, true)
			}
			c.AuditService = services.NewAuditService(db, runLogger)
		}
	}
	if c.ReviewService != nil && c.ReviewService.Audit == nil {
		c.ReviewService.Audit = c.AuditService
	}

	// 5. Initialize Logging and Auth Services
	if c.LogService == nil {
		c.LogService = services.NewLogService()
	}
	if c.AuthService == nil {
		c.AuthService = remoteauth.NewRemoteAuthService(cfg, c.Client, nil)
	}

	// 6. Assemble primary SyncService if not already provided
	if c.SyncService == nil {
		c.SyncService = &services.SyncService{
			Config:    cfg,
			Client:    c.Client,
			Canary:    c.CanaryService,
			Deletion:  c.DeletionService,
			Telemetry: c.TelemetryService,
			Explorer:  c.ExplorerService,
			Review:    c.ReviewService,
			Audit:     c.AuditService,
		}
	}

	// 7. Initialize Profiles Subsystem
	if cfg.ProfilesFile != "" {
		profStore, err := profiles.NewStore(cfg.ProfilesFile, cfg.ProfilesHistoryFile)
		if err == nil {
			_, _ = profStore.BootstrapDefault(cfg.LocalPath, cfg.RemotePath)
			c.ProfileStore = profStore
			if c.ProfileExecutor == nil {
				c.ProfileExecutor = DefaultProfileExecutor(cfg)
			}
			c.ProfileScheduler = profiles.NewScheduler(profStore, c.ProfileExecutor)
			c.ProfileService = profiles.NewService(profStore, c.Client, c.ProfileScheduler)
		}
	}

	return c, nil
}

// Start begins background services including the profile scheduler.
func (c *Container) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return nil
	}
	c.started = true
	if c.ProfileScheduler != nil {
		if err := c.ProfileScheduler.Start(ctx); err != nil {
			return fmt.Errorf("failed to start profile scheduler: %w", err)
		}
	}
	return nil
}

// Close gracefully terminates background workers and releases database connections.
func (c *Container) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return nil
	}
	c.stopped = true

	if c.ProfileScheduler != nil {
		c.ProfileScheduler.Stop()
	}

	var errs []error
	if c.AuditService != nil {
		if err := c.AuditService.Close(); err != nil {
			errs = append(errs, fmt.Errorf("audit service close: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("container close errors: %v", errs)
	}
	return nil
}
