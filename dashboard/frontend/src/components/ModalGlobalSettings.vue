<template>
  <Dialog
    :visible="settingsStore.showGlobalSettingsModal"
    @update:visible="val => !val && settingsStore.closeGlobalSettingsModal()"
    modal
    header="Global Fleet Settings"
    :style="{ width: '94vw', maxWidth: '980px' }"
    :pt="{
      root: { class: 'bg-carbon-surface border border-carbon-border shadow-2xl rounded-2xl overflow-hidden' },
      header: { class: 'p-4 border-b border-carbon-border bg-carbon-surface text-white flex items-center justify-between' },
      content: { class: 'p-0 bg-carbon-canvas text-xs font-mono overflow-hidden' },
      footer: { class: 'p-4 border-t border-carbon-border bg-carbon-surface flex justify-between items-center' }
    }"
  >
    <template #header>
      <div class="flex items-center gap-2.5">
        <div class="w-8 h-8 rounded-xl bg-electric-blue/10 border border-electric-blue/30 text-electric-blue flex items-center justify-center shrink-0">
          <i class="pi pi-cog text-base"></i>
        </div>
        <div>
          <h3 class="text-sm font-bold text-white tracking-wide">Global Fleet Settings</h3>
          <p class="text-[11px] text-slate-400">Daemon engine tuning, fallback flags, and alert integrations</p>
        </div>
      </div>
    </template>

    <div class="flex flex-col md:flex-row min-h-[500px] max-h-[74vh]">
      <!-- Left Sidebar Navigation -->
      <aside class="w-full md:w-56 shrink-0 border-b md:border-b-0 md:border-r border-carbon-border bg-carbon-surface/50 p-3 space-y-1 overflow-y-auto">
        <button
          v-for="t in tabs"
          :key="t.id"
          type="button"
          @click="activeTab = t.id"
          :class="activeTab === t.id
            ? 'bg-electric-blue text-black font-bold shadow-md'
            : 'text-slate-400 hover:text-white hover:bg-carbon-surface border border-transparent'"
          class="w-full px-3 py-2.5 rounded-xl transition-all cursor-pointer text-xs flex items-center gap-2.5 text-left"
        >
          <i :class="t.icon" class="text-sm shrink-0"></i>
          <span class="truncate">{{ t.label }}</span>
        </button>
      </aside>

      <!-- Right Content Canvas -->
      <main class="flex-1 p-5 md:p-6 overflow-y-auto space-y-5">
        <!-- Error Banner -->
        <div v-if="saveError || settingsStore.configError" class="p-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-rose-300 text-xs">
          {{ saveError || settingsStore.configError }}
        </div>

        <!-- TAB 1: OPERATIONAL DEFAULTS -->
        <div v-if="activeTab === 'defaults'" class="space-y-5">
          <!-- Card 1: Bootstrap Target Fallbacks -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Bootstrap Target Fallbacks</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">
                Default paths configured in <span class="text-electric-blue font-mono">bisync.env</span> (<code class="text-slate-300">LOCAL_PATH</code> & <code class="text-slate-300">REMOTE_PATH</code>). Used as bootstrap defaults for manual CLI bisync runs.
                <span class="text-amber-300 block sm:inline">Active sync profiles manage their own isolated path mappings in Profiles.</span>
              </p>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5 pt-1">
              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Fallback Local Bootstrap Path</label>
                <input
                  type="text"
                  v-model="formDefaults.default_local_path"
                  placeholder="~/sync_folder"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
              </div>
              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Fallback Remote Bootstrap Target</label>
                <input
                  type="text"
                  v-model="formDefaults.default_remote_path"
                  placeholder="drive:backup"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
              </div>
            </div>
          </div>

          <!-- Card 2: Profile Baseline Safeguards -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Sync Profile Baseline Safeguards</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Default safety and conflict heuristics applied when creating new sync profiles.</p>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Default Max Delete %</label>
                <input
                  type="number"
                  v-model.number="formDefaults.max_delete"
                  min="0"
                  max="100"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
                <span class="text-[10px] text-slate-500 mt-1 block">0 = Abort on any deletion; 1-100 = allowable % threshold.</span>
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Conflict Resolution Mode</label>
                <select
                  v-model="formDefaults.conflict_resolve"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                >
                  <option value="newer">newer (Keep newest file)</option>
                  <option value="older">older (Keep older file)</option>
                  <option value="larger">larger (Keep larger file)</option>
                  <option value="smaller">smaller (Keep smaller file)</option>
                  <option value="path1">path1 (Prefer Local)</option>
                  <option value="path2">path2 (Prefer Remote)</option>
                </select>
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Comparison Mode</label>
                <select
                  v-model="formDefaults.compare_mode"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                >
                  <option value="modtime">modtime</option>
                  <option value="size,modtime">size,modtime (Standard)</option>
                  <option value="size,modtime,checksum">size,modtime,checksum (Thorough)</option>
                  <option value="checksum">checksum (Hash only)</option>
                </select>
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Canary Token Validation</label>
                <select
                  :value="formDefaults.canary_validation ? 'true' : 'false'"
                  @change="formDefaults.canary_validation = ($event.target as HTMLSelectElement).value === 'true'"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                >
                  <option value="true">Enabled (Strict RCLONE_TEST Guard)</option>
                  <option value="false">Disabled (Bypass Probing)</option>
                </select>
              </div>
            </div>
          </div>

          <!-- Card 3: Pruning & Explorer Remote Scoping -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Pruning & Explorer Scoping</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Directory pruning behavior and dual explorer workspace confinement.</p>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1.5">Empty Directory Cleanup</label>
                <div class="space-y-2">
                  <label class="flex items-center gap-2 cursor-pointer p-2 rounded-lg bg-carbon-canvas border border-carbon-border">
                    <input type="checkbox" v-model="formDefaults.remove_empty_dirs" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
                    <span class="text-xs text-slate-300">Remove Empty Dirs</span>
                  </label>
                  <label class="flex items-center gap-2 cursor-pointer p-2 rounded-lg bg-carbon-canvas border border-carbon-border">
                    <input type="checkbox" v-model="formDefaults.create_empty_dirs" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
                    <span class="text-xs text-slate-300">Create Empty Dirs</span>
                  </label>
                </div>
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Explorer Remote Scoping</label>
                <select
                  :value="formDefaults.lock_explorer_remote ? 'true' : 'false'"
                  @change="formDefaults.lock_explorer_remote = ($event.target as HTMLSelectElement).value === 'true'"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                >
                  <option value="false">Unlocked (Freely select any remote)</option>
                  <option value="true">Locked to Active Profile Remote</option>
                </select>
                <span class="text-[10px] text-slate-500 mt-1 block">Locks dual explorer right pane strictly to active profile remote.</span>
              </div>
            </div>
          </div>
        </div>

        <!-- TAB 2: FLEET FLAGS -->
        <div v-else-if="activeTab === 'flags'" class="space-y-4">
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-2">
            <h4 class="text-xs font-bold text-white">Global Engine Concurrency & Networking Flags</h4>
            <p class="text-[11px] text-slate-400">
              Baseline Rclone engine concurrency, rate limits, and network parameters applied to the background daemon. Process-global options (<code class="text-amber-400">--tpslimit</code>, <code class="text-amber-400">--checkers</code>) define the daemon baseline; profiles with custom overrides queue safely and dynamically switch rates.
            </p>
            <p class="text-[11px] text-slate-500 italic">
              Note: Profile-specific operational flags (e.g. rename tracking, checksums) and cloud provider options are configured within individual sync profiles.
            </p>
          </div>

          <FlagsBuilder v-model="formDefaults.custom_flags" scopeFilter="all" remoteType="all" />
        </div>

        <!-- TAB 3: ENGINE & POLLING -->
        <div v-else-if="activeTab === 'engine'" class="space-y-5">
          <!-- Card 1: Job Timeouts & Stale Lock Protection -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Job Timeouts & Stale Lock Protection</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Execution limits ensuring hung sync processes are safely aborted and reclaimed.</p>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Max Job Timeout (Seconds)</label>
                <input
                  type="number"
                  v-model.number="formDaemon.job_timeout_seconds"
                  min="60"
                  max="86400"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
                <span class="text-[10px] text-slate-500 mt-1 block">Default: 1800s (30m). Terminate run if runtime exceeds this threshold.</span>
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Stale Lock Timeout (Seconds)</label>
                <input
                  type="number"
                  v-model.number="formDaemon.lock_stale_timeout_seconds"
                  min="60"
                  max="86400"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
                <span class="text-[10px] text-slate-500 mt-1 block">Default: 300s (5m). Reclaim locks older than this duration if process died.</span>
              </div>
            </div>
          </div>

          <!-- Card 2: Rate Limiting & Backoff Multipliers -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Rate Limiting & Backoff Multipliers</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Automated cooldown multipliers to protect against cloud provider API rate limits.</p>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Rate Limit Backoff Period (Hours)</label>
                <input
                  type="number"
                  v-model.number="formDaemon.rate_limit_backoff_hours"
                  min="0.1"
                  max="48"
                  step="0.5"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
                <span class="text-[10px] text-slate-500 mt-1 block">Hours to pause sync operations when an upstream 429 is triggered.</span>
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Poll Backoff Multiplier</label>
                <input
                  type="number"
                  v-model.number="formDaemon.job_poll_backoff_multiplier"
                  min="1.0"
                  max="5.0"
                  step="0.1"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
                <span class="text-[10px] text-slate-500 mt-1 block">Exponential backoff factor for background sync polling loops.</span>
              </div>

              <div class="sm:col-span-2">
                <label class="block text-xs font-semibold text-slate-300 mb-1">Poll Sleep Interval (Initial / Max Seconds)</label>
                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <input
                      type="number"
                      v-model.number="formDaemon.job_poll_initial_sleep_sec"
                      min="0.5"
                      step="0.5"
                      placeholder="Initial sleep (e.g. 2s)"
                      class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                    />
                    <span class="text-[10px] text-slate-500 mt-0.5 block">Initial poll sleep interval.</span>
                  </div>
                  <div>
                    <input
                      type="number"
                      v-model.number="formDaemon.job_poll_max_sleep_sec"
                      min="1"
                      step="1"
                      placeholder="Max sleep (e.g. 10s)"
                      class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                    />
                    <span class="text-[10px] text-slate-500 mt-0.5 block">Maximum backed-off sleep interval.</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Card 3: Daemon Automated Safeguards -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Daemon Automated Safeguards</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Automated self-healing toggles for background daemon engine workers.</p>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
              <label class="flex items-start gap-2.5 cursor-pointer p-3 rounded-lg bg-carbon-canvas border border-carbon-border">
                <input type="checkbox" v-model="formDaemon.check_stale_locks" class="rounded bg-carbon-surface border-carbon-border text-electric-blue mt-0.5" />
                <div>
                  <span class="text-xs font-semibold text-slate-200 block">Check & Clear Stale Locks</span>
                  <span class="text-[10px] text-slate-500">Scan for abandoned PID lockfiles before launching jobs.</span>
                </div>
              </label>

              <label class="flex items-start gap-2.5 cursor-pointer p-3 rounded-lg bg-carbon-canvas border border-carbon-border">
                <input type="checkbox" v-model="formDaemon.rate_limit_retry_after_backoff" class="rounded bg-carbon-surface border-carbon-border text-electric-blue mt-0.5" />
                <div>
                  <span class="text-xs font-semibold text-slate-200 block">Auto-Retry Jobs Post-Backoff</span>
                  <span class="text-[10px] text-slate-500">Automatically re-queue rate-limited jobs when backoff expires.</span>
                </div>
              </label>
            </div>
          </div>
        </div>

        <!-- TAB 4: LOGGING & CLEANUP -->
        <div v-else-if="activeTab === 'logging'" class="space-y-5">
          <!-- Card 1: Log Rotation & Retention -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Log Rotation & Archival Retention</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Bounded disk consumption rules and archive retention count for <code class="text-slate-300">bisync.log</code>.</p>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Max Log Size (MB)</label>
                <input
                  type="number"
                  v-model.number="formLogging.max_log_size_mb"
                  min="1"
                  max="500"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
                <span class="text-[10px] text-slate-500 mt-1 block">Rotates log file when this size threshold is reached.</span>
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Log Backup Retention Count</label>
                <input
                  type="number"
                  v-model.number="formLogging.log_backup_count"
                  min="1"
                  max="50"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
                <span class="text-[10px] text-slate-500 mt-1 block">Number of archived log backups to retain on disk.</span>
              </div>
            </div>
          </div>

          <!-- Card 2: Post-Sync Cleanup -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Post-Sync Ephemeral File Cleanup</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Comma-separated filename patterns automatically pruned after successful sync completion.</p>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-300 mb-1">Cleanup Patterns (Globs)</label>
              <input
                type="text"
                v-model="formCleanup.files_to_cleanup"
                placeholder="*.tmp, *.temp, *.conflict, *.~lock.*"
                class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
              />
              <span class="text-[10px] text-slate-500 mt-1 block">Matches are safely removed from the synced root after each completed run.</span>
            </div>
          </div>
        </div>

        <!-- TAB 5: TELEMETRY & FEATURES -->
        <div v-else-if="activeTab === 'telemetry'" class="space-y-5">
          <!-- Card 1: Telemetry Metrics -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Metrics Ledger & Memory Retention</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Execution throughput tracking and in-memory time-series limits.</p>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Execution Metrics</label>
                <select
                  :value="formTelemetry.enable_metrics ? 'true' : 'false'"
                  @change="formTelemetry.enable_metrics = ($event.target as HTMLSelectElement).value === 'true'"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                >
                  <option value="true">Enabled (Track durations, throughput & counts)</option>
                  <option value="false">Disabled</option>
                </select>
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Metrics Max In-Memory Records</label>
                <input
                  type="number"
                  v-model.number="formTelemetry.metrics_max_entries"
                  min="100"
                  max="20000"
                  step="100"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
              </div>
            </div>
          </div>

          <!-- Card 2: Logging Verbosity & MCP Agent Authority -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Logging Verbosity & MCP Agent Authority</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Control daemon diagnostic verbosity and AI agent write privileges.</p>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Debug Output Verbosity</label>
                <select
                  :value="formFeatures.debug_mode ? 'true' : 'false'"
                  @change="formFeatures.debug_mode = ($event.target as HTMLSelectElement).value === 'true'"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                >
                  <option value="false">Standard (Notice, Info, Warnings)</option>
                  <option value="true">Verbose Debug Logging (DEBUG_MODE=true)</option>
                </select>
                <span class="text-[10px] text-slate-500 mt-1 block">Verbose mode logs raw Rclone JSON-RPC packets and HTTP access traces.</span>
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">MCP Agent Authority</label>
                <select
                  :value="formFeatures.mcp_read_only ? 'true' : 'false'"
                  @change="formFeatures.mcp_read_only = ($event.target as HTMLSelectElement).value === 'true'"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                >
                  <option value="false">Read & Write (Full Agent Execution)</option>
                  <option value="true">Read Only (Inspection & Safe Dry-Runs Only)</option>
                </select>
                <span class="text-[10px] text-slate-500 mt-1 block">Locks MCP tools against destructive mutation when in Read Only mode.</span>
              </div>
            </div>
          </div>

          <!-- Card 3: Subsystem Feature Toggles -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Subsystem Feature Toggles</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Granular feature flags enabling or bypassing internal sub-modules.</p>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5 pt-1">
              <label class="flex items-center gap-2.5 cursor-pointer p-2.5 rounded-lg bg-carbon-canvas border border-carbon-border">
                <input type="checkbox" v-model="formFeatures.stale_lock_detection" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
                <div>
                  <span class="text-xs font-semibold text-slate-200 block">Stale Lock Detection</span>
                  <span class="text-[10px] text-slate-500">Auto-detect crashed process locks.</span>
                </div>
              </label>

              <label class="flex items-center gap-2.5 cursor-pointer p-2.5 rounded-lg bg-carbon-canvas border border-carbon-border">
                <input type="checkbox" v-model="formFeatures.rate_limit_backoff" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
                <div>
                  <span class="text-xs font-semibold text-slate-200 block">Rate Limit Backoff</span>
                  <span class="text-[10px] text-slate-500">Provider throttling protection.</span>
                </div>
              </label>

              <label class="flex items-center gap-2.5 cursor-pointer p-2.5 rounded-lg bg-carbon-canvas border border-carbon-border">
                <input type="checkbox" v-model="formFeatures.daemon_health_check" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
                <div>
                  <span class="text-xs font-semibold text-slate-200 block">Daemon Health Checks</span>
                  <span class="text-[10px] text-slate-500">Periodic Rclone RC socket probes.</span>
                </div>
              </label>

              <label class="flex items-center gap-2.5 cursor-pointer p-2.5 rounded-lg bg-carbon-canvas border border-carbon-border">
                <input type="checkbox" v-model="formFeatures.smart_notifications" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
                <div>
                  <span class="text-xs font-semibold text-slate-200 block">Smart Alert Throttling</span>
                  <span class="text-[10px] text-slate-500">Suppress repetitive error spam.</span>
                </div>
              </label>

              <label class="flex items-center gap-2.5 cursor-pointer p-2.5 rounded-lg bg-carbon-canvas border border-carbon-border sm:col-span-2">
                <input type="checkbox" v-model="formFeatures.webhook_alerts" class="rounded bg-carbon-surface border-carbon-border text-electric-blue" />
                <div>
                  <span class="text-xs font-semibold text-slate-200 block">External Webhook Alerts</span>
                  <span class="text-[10px] text-slate-500">Dispatch sync failure alerts to configured webhook endpoint.</span>
                </div>
              </label>
            </div>
          </div>
        </div>

        <!-- TAB 6: NOTIFICATIONS & WEBHOOKS -->
        <div v-else-if="activeTab === 'notifications'" class="space-y-5">
          <!-- Card 1: Notification Engine -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">Desktop & System Notification Engine</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Desktop notification parameters and throttle intervals.</p>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-3.5 pt-1">
              <div class="sm:col-span-1">
                <label class="block text-xs font-semibold text-slate-300 mb-1">Application Name</label>
                <input
                  type="text"
                  v-model="formNotifications.app_name"
                  placeholder="Rclone Bisync"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Success Alert (min)</label>
                <input
                  type="number"
                  v-model.number="formNotifications.success_interval_min"
                  min="0"
                  max="1440"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-300 mb-1">Error Repeat (hrs)</label>
                <input
                  type="number"
                  v-model.number="formNotifications.error_repeat_hours"
                  min="0.1"
                  max="72"
                  step="0.5"
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
              </div>
            </div>
          </div>

          <!-- Card 2: External Endpoints -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <div>
              <h4 class="text-xs font-bold text-white">External Integration Endpoints</h4>
              <p class="text-[11px] text-slate-400 mt-0.5">Webhooks for operational alerts and external uptime monitoring.</p>
            </div>
            <div class="space-y-3 pt-1">
              <div>
                <div class="flex items-center justify-between mb-1">
                  <label class="text-xs font-semibold text-slate-300">Alert Webhook URL</label>
                  <span class="text-[10px] text-slate-500 font-mono">Timeout: {{ formNotifications.webhook_timeout_sec || 5 }}s</span>
                </div>
                <input
                  type="url"
                  v-model="formNotifications.webhook_url"
                  placeholder="https://hooks.slack.com/services/... or https://discord.com/api/webhooks/..."
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
                <span class="text-[10px] text-slate-500 mt-1 block">POST JSON payload dispatched on sync failure, canary trip, or excessive deletion.</span>
              </div>

              <div>
                <div class="flex items-center justify-between mb-1">
                  <label class="text-xs font-semibold text-slate-300">Watchdog Heartbeat Ping URL</label>
                  <span class="text-[10px] text-slate-500 font-mono">Timeout: {{ formNotifications.watchdog_timeout_sec || 5 }}s</span>
                </div>
                <input
                  type="url"
                  v-model="formNotifications.watchdog_ping_url"
                  placeholder="https://nosnch.in/... or https://hc-ping.com/..."
                  class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue"
                />
                <span class="text-[10px] text-slate-500 mt-1 block">GET ping dispatched following every successfully completed scheduled cycle.</span>
              </div>
            </div>
          </div>
        </div>

        <!-- TAB 7: ENCRYPTION -->
        <div v-else-if="activeTab === 'encryption'" class="space-y-5">
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-2">
            <h4 class="text-xs font-bold text-white">Configuration Encryption at Rest</h4>
            <p class="text-[11px] text-slate-400">
              Protect sensitive tokens, passwords, and custom paths in <code class="text-slate-300">bisync.env</code> using AES-256-GCM encryption with Argon2id key derivation.
            </p>
          </div>

          <!-- Encryption Status Banner -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl flex items-center justify-between gap-3">
            <div class="flex items-center gap-3">
              <div
                class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0"
                :class="settingsStore.encryption.is_encrypted ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' : 'bg-slate-800 text-slate-400'"
              >
                <i :class="settingsStore.encryption.is_encrypted ? 'pi pi-lock' : 'pi pi-lock-open'" class="text-sm"></i>
              </div>
              <div>
                <span class="text-xs font-bold text-white block">
                  {{ settingsStore.encryption.is_encrypted ? 'Configuration Encrypted' : 'Configuration Stored in Plaintext' }}
                </span>
                <span class="text-[10px] text-slate-500">
                  {{ settingsStore.encryption.is_encrypted ? 'Sensitive keys protected via local passphrase.' : 'Tokens and secrets are stored in plaintext on disk.' }}
                </span>
              </div>
            </div>
            <span
              class="text-[10px] uppercase font-mono px-2 py-0.5 rounded border"
              :class="settingsStore.encryption.is_encrypted ? 'bg-emerald-950 text-emerald-300 border-emerald-800/60' : 'bg-slate-800 text-slate-400 border-slate-700'"
            >
              {{ settingsStore.encryption.is_encrypted ? 'SECURE' : 'PLAINTEXT' }}
            </span>
          </div>

          <div v-if="encError" class="p-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-rose-300 text-xs">
            {{ encError }}
          </div>

          <!-- Set / Update Passphrase -->
          <div class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <h4 class="text-xs font-bold text-white">
              {{ settingsStore.encryption.is_encrypted ? 'Change Master Passphrase' : 'Set Master Passphrase' }}
            </h4>
            <div v-if="settingsStore.encryption.is_encrypted">
              <label class="block text-xs text-slate-400 mb-1">Current Passphrase</label>
              <input type="password" v-model="oldPassword" class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label class="block text-xs text-slate-400 mb-1">New Passphrase</label>
                <input type="password" v-model="newPassword" class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
              </div>
              <div>
                <label class="block text-xs text-slate-400 mb-1">Confirm New Passphrase</label>
                <input type="password" v-model="confirmPassword" class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
              </div>
            </div>
            <button
              type="button"
              @click="handleSetPassword"
              :disabled="!newPassword || settingsStore.loadingEncryption"
              class="px-3.5 py-1.5 rounded-lg bg-electric-blue hover:bg-sky-400 text-black font-bold text-xs cursor-pointer disabled:opacity-50 transition-colors"
            >
              {{ settingsStore.loadingEncryption ? 'Applying...' : (settingsStore.encryption.is_encrypted ? 'Update Passphrase' : 'Save Passphrase') }}
            </button>
          </div>

          <!-- Remove Passphrase (if encrypted) -->
          <div v-if="settingsStore.encryption.is_encrypted" class="p-4 bg-carbon-surface border border-carbon-border rounded-xl space-y-3">
            <h4 class="text-xs font-bold text-rose-300">Remove Encryption</h4>
            <div>
              <label class="block text-xs text-slate-400 mb-1">Current Passphrase</label>
              <input type="password" v-model="oldPassword" class="w-full bg-carbon-canvas border border-carbon-border rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-electric-blue" />
            </div>
            <button
              type="button"
              @click="handleRemovePassword"
              :disabled="!oldPassword || settingsStore.loadingEncryption"
              class="px-3.5 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-500 text-white font-bold text-xs cursor-pointer disabled:opacity-50 transition-colors"
            >
              Remove Passphrase
            </button>
          </div>
        </div>
      </main>
    </div>

    <template #footer>
      <button
        type="button"
        @click="settingsStore.closeGlobalSettingsModal()"
        class="px-3.5 py-1.5 rounded-xl border border-carbon-border text-slate-300 hover:text-white cursor-pointer transition-colors"
      >
        Cancel
      </button>

      <button
        type="button"
        @click="handleSave"
        :disabled="settingsStore.savingConfig"
        class="px-4 py-1.5 rounded-xl bg-electric-blue hover:bg-sky-400 text-black font-bold cursor-pointer flex items-center gap-1.5 disabled:opacity-50 transition-colors"
      >
        <i class="pi" :class="settingsStore.savingConfig ? 'pi-spin pi-spinner' : 'pi-check'"></i>
        <span>{{ settingsStore.savingConfig ? 'Saving...' : 'Save Settings' }}</span>
      </button>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { useSettingsStore } from '../stores/settings'
import FlagsBuilder from './FlagsBuilder.vue'

const settingsStore = useSettingsStore()
const activeTab = ref<'defaults' | 'flags' | 'engine' | 'logging' | 'telemetry' | 'notifications' | 'encryption'>('defaults')
const saveError = ref<string | null>(null)

const formDefaults = ref({ ...settingsStore.globalDefaults })
const formDaemon = ref({ ...settingsStore.daemonSettings })
const formLogging = ref({ ...settingsStore.loggingSettings })
const formCleanup = ref({ ...settingsStore.cleanupSettings })
const formTelemetry = ref({ ...settingsStore.telemetrySettings })
const formFeatures = ref({ ...settingsStore.featureSettings })
const formNotifications = ref({ ...settingsStore.notificationSettings })

const newPassword = ref('')
const confirmPassword = ref('')
const oldPassword = ref('')
const daemonPassword = ref('')
const encError = ref<string | null>(null)

watch(() => settingsStore.globalDefaults, val => { formDefaults.value = { ...val } }, { deep: true })
watch(() => settingsStore.daemonSettings, val => { formDaemon.value = { ...val } }, { deep: true })
watch(() => settingsStore.loggingSettings, val => { formLogging.value = { ...val } }, { deep: true })
watch(() => settingsStore.cleanupSettings, val => { formCleanup.value = { ...val } }, { deep: true })
watch(() => settingsStore.telemetrySettings, val => { formTelemetry.value = { ...val } }, { deep: true })
watch(() => settingsStore.featureSettings, val => { formFeatures.value = { ...val } }, { deep: true })
watch(() => settingsStore.notificationSettings, val => { formNotifications.value = { ...val } }, { deep: true })
watch(() => settingsStore.globalSettingsTab, tab => { activeTab.value = tab as any })

const tabs: Array<{ id: 'defaults' | 'flags' | 'engine' | 'logging' | 'telemetry' | 'notifications' | 'encryption'; label: string; icon: string }> = [
  { id: 'defaults', label: 'Defaults', icon: 'pi pi-sliders-h' },
  { id: 'flags', label: 'Fleet Flags', icon: 'pi pi-server' },
  { id: 'engine', label: 'Engine & Polling', icon: 'pi pi-clock' },
  { id: 'logging', label: 'Logging & Cleanup', icon: 'pi pi-file' },
  { id: 'telemetry', label: 'Telemetry & Features', icon: 'pi pi-chart-bar' },
  { id: 'notifications', label: 'Alert Webhooks', icon: 'pi pi-bell' },
  { id: 'encryption', label: 'Encryption', icon: 'pi pi-lock' },
]

async function handleSave() {
  saveError.value = null
  try {
    await settingsStore.saveConfig({
      global_defaults: formDefaults.value,
      daemon: formDaemon.value,
      logging: formLogging.value,
      cleanup: formCleanup.value,
      telemetry: formTelemetry.value,
      features: formFeatures.value,
      notifications: formNotifications.value,
    })
    settingsStore.closeGlobalSettingsModal()
  } catch (err: any) {
    saveError.value = err.message || 'Failed to save configuration'
  }
}

async function handleSetPassword() {
  encError.value = null
  if (newPassword.value !== confirmPassword.value) {
    encError.value = 'Passwords do not match'
    return
  }
  try {
    await settingsStore.setPassword(newPassword.value, oldPassword.value)
    newPassword.value = ''
    confirmPassword.value = ''
    oldPassword.value = ''
  } catch (e: any) {
    encError.value = e.message || 'Failed to set encryption password'
  }
}

async function handleRemovePassword() {
  encError.value = null
  try {
    await settingsStore.removePassword(oldPassword.value)
    oldPassword.value = ''
  } catch (e: any) {
    encError.value = e.message || 'Failed to remove password'
  }
}

async function handleSaveDaemonPassphrase() {
  encError.value = null
  try {
    await settingsStore.saveDaemonPassphrase(daemonPassword.value)
    daemonPassword.value = ''
  } catch (e: any) {
    encError.value = e.message || 'Failed to save daemon passphrase'
  }
}
</script>
