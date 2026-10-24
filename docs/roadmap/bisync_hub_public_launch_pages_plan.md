# Bisync Hub: Public Launch, Governance, and GitHub Pages Plan

---

## 1. Executive Summary & Repository Readiness

Bisync Hub has achieved production readiness as a native single-binary Go application with zero runtime dependencies (no Python, Node.js, or CGO). Following the delivery of **Phase 4 (Interactive Sync Review)**, the **Heuristic Conflict Recommendation Engine**, and the **Frontend Template Modularization (`html/template`)**, the repository stands at an **83.2% statement test coverage** baseline across all 15 packages.

To safeguard the intellectual property and token investment (~160 Euros via Gemini Flash 3.8), this plan establishes a strict single-maintainer merge authority model, automated CI quality gates, security scanning, and a dedicated **GitHub Pages landing platform** to highlight Bisync Hub's Unique Selling Propositions (USPs).

---

## 2. Repository Governance & Sole Control Strategy

```text
                          +----------------------------------------------+
                          |         Public Community Interactions        |
                          |  - Issue Tracking (Bug / Feature Templates)  |
                          |  - GitHub Discussions                        |
                          |  - External Pull Requests (Fork Submissions) |
                          +----------------------+-----------------------+
                                                 |
                                                 v
                          +----------------------------------------------+
                          |            Automated Quality Gate            |
                          |  - GitHub Actions CI (./scripts/check.sh)    |
                          |  - Gitleaks Secret & Credential Scan         |
                          |  - Statement Test Coverage >= 80%            |
                          +----------------------+-----------------------+
                                                 |
                                                 v
                          +----------------------------------------------+
                          |         Sole Approval & Merge Barrier        |
                          |  - CODEOWNERS (* @your-username)             |
                          |  - Protected main Branch (1 Mandatory Review) |
                          |  - Block Direct Push & Force Push            |
                          +----------------------+-----------------------+
                                                 |
                                                 v
                          +----------------------------------------------+
                          |         Main Production Branch (main)        |
                          +----------------------------------------------+
```

### 2.1 Branch Protection Rules (`main` branch)
* **Required Approval Count**: Set strictly to `1` (your individual account).
* **Dismiss Stale Approvals**: Automatically revoke approvals when new commits are pushed.
* **Require Review from Code Owners**: Force all PRs to pass your explicit review.
* **Restrict Direct Pushes**: Prevent direct pushes or force-pushes to `main` for all users.
* **Require Status Checks**: Require the `CI / Quality Gate` workflow to pass before merging.

### 2.2 Code Ownership (`.github/CODEOWNERS`)
```gitignore
# Global sole ownership
* @<your-github-username>

# Core backend engine, security gates, and automation
/internal/ @<your-github-username>
/dashboard/ @<your-github-username>
/scripts/ @<your-github-username>
/main.go @<your-github-username>
```

### 2.3 Automated CI Workflow (`.github/workflows/ci.yml`)
```yaml
name: CI Quality Gate

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  check:
    name: Quality Gate & Cross-Compile
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Repository
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run Gitleaks Secret Scanner
        uses: gitleaks/gitleaks-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Execute 8-Stage Quality Gate
        run: |
          chmod +x ./scripts/check.sh
          ./scripts/check.sh
```

---

## 3. GitHub Pages Landing Site Specification

The GitHub Pages site (`https://<your-username>.github.io/rclone-bisync`) serves as the public landing page for Bisync Hub. It communicates the project's core philosophy—**"Review. Decide. Sync."**—and showcases the USP features through interactive UI previews and technical diagrams.

### 3.1 Design & Aesthetic Direction
* **Aesthetic**: Modern, slate-and-indigo dark mode matching the dashboard UI. Framework-free HTML/CSS/JS (or lightweight Tailwind build via GitHub Actions).
* **Typography & Icons**: Inter font, monospace code blocks, and offline Lucide SVG icons matching the app.
* **Performance**: 100/100 Lighthouse performance score with zero heavy JS bundles, tracking scripts, or external font loads.

### 3.2 Page Structure & Content Layout

```text
+-----------------------------------------------------------------------------------+
| HERO SECTION                                                                      |
|  - Tagline: "Review. Decide. Sync."                                              |
|  - Core Value: The Safety-First Operational Platform for Rclone Bisync            |
|  - One-Liner Install Badge & Cross-Platform Download Links                       |
|  - Interactive SVG Dashboard Preview                                              |
+-----------------------------------------------------------------------------------+
| USP FEATURE SHOWCASE                                                              |
|  [1. Interactive Sync Review]     [2. Heuristic Conflict Engine]                  |
|  [3. Customizable Safety Pipeline] [4. Immutable Audit Ledger]                    |
|  [5. Native Single-Binary Architecture]                                           |
+-----------------------------------------------------------------------------------+
| HOW IT WORKS (VISUAL WORKFLOW)                                                    |
|  Scan -> Intercept -> Analyze -> Review -> Decide -> Transactional Sync           |
+-----------------------------------------------------------------------------------+
| REAL-WORLD USAGE GUIDES                                                           |
|  - Resolving TOO_MANY_DELETES without --force                                     |
|  - Auto-Resolving 50+ File Conflicts Safely                                       |
|  - Protecting Unmounted Drives & Critical Archives                                |
+-----------------------------------------------------------------------------------+
| QUICKSTART & SYSTEMD INTEGRATION                                                  |
|  - CLI Commands & systemd timer setup                                             |
+-----------------------------------------------------------------------------------+
| ARCHITECTURE & SAFETY PRINCIPLES                                                  |
|  - Visibility Before Action | Safety Before Automation | Recovery Before Rebuild      |
+-----------------------------------------------------------------------------------+
```

---

### 3.3 USP Features Detail & Usage Showcase

#### USP 1: Interactive Sync Review & Deletion Diff Visualizer
* **The Problem**: Standard `rclone bisync` aborts on `TOO_MANY_DELETES` or conflict errors. Users must either inspect logs manually or run `rclone bisync --force`, which executes an all-or-nothing sync plan that risks deleting valid files.
* **The Solution**: An in-dashboard review session that parses aborted work files, highlights pending deletions (red), conflicts (amber), and new files (green), and allows per-item decision making.
* **Safety Net**: Creates POSIX zero-byte hard links locally and cloud-to-cloud backups prior to any change. Features a Saga rollback runner if resolution halts.
* **Usage Example**:
  ```bash
  # When bisync aborts due to 12 pending deletions:
  # 1. Open Bisync Hub Dashboard (or run `rclone-bisync dashboard`)
  # 2. Click "Review Changes" on the Cockpit Alert
  # 3. Mark 10 intentional deletions as "Accept Delete"
  # 4. Mark 2 accidental deletions as "Restore Remotely"
  # 5. Click "Apply Resolutions" -> Bisync Hub restores the 2 items, accepts the 10,
  #    and triggers `rclone bisync --resync` automatically.
  ```

#### USP 2: Heuristic Conflict Recommendation Engine
* **The Problem**: Dispersed file modifications across devices create bidirectional conflicts where modification timestamps alone do not indicate which copy is legitimate.
* **The Solution**: A pure-Go analyzer pipeline running parallel workers ($G=4$) that evaluates candidates against deterministic rules:
  * `TempLockAnalyzer`: Detects temporary/lock files (`~$*`, `*.tmp`, `.DS_Store`).
  * `TruncationAnalyzer`: Flags 0-byte truncations or >90% file size drops as `CRITICAL` risk.
  * `ClockSkewAnalyzer`: Recognizes 1-hour/2-hour Daylight Saving Time offsets.
  * `SizeDivergenceAnalyzer`: Highlights binary size differences exceeding 20%.
* **Usage Example**:
  ```text
  [Review Session] -> Click "Apply High-Confidence Recommendations"
  - 42 lock files automatically staged as "Keep Primary" (Confidence: 0.95, Risk: LOW)
  - 3 DST clock-skewed files staged as "Keep Newer" (Confidence: 0.88, Risk: LOW)
  - 1 truncated 0-byte document flagged as "MANUAL REVIEW REQUIRED" (Risk: CRITICAL)
  ```

#### USP 3: Dual-Stage Customizable Safety Policies
* **The Problem**: Unmounted local external drives or unmounted cloud shares cause sync engines to interpret empty roots as mass remote deletions.
* **The Solution**: A two-stage evaluation pipeline:
  * **Stage 1 (Pre-Execution Gate)**: Runs a single-item RC `operations/stat` lookup for required path markers (e.g., `.mount_check`) in <200ms using 1 API call before Rclone begins scanning.
  * **Stage 2 (Pre-Mutation Interceptor)**: Parses local `.lst` work files in memory with zero API calls to enforce `MAX_DELETE_PERCENTAGE` and `PROTECTED_PATH` rules.
* **Usage Example**:
  ```yaml
  # policies.yaml snippet
  policies:
    - name: "Mount Verification"
      type: "MANDATORY_SUBDIR"
      parameters: { required_paths: [".mount_check"] }
      action: "ABORT_SYNC" # Stops run before scanning if mount check fails
      
    - name: "Protect Financials"
      type: "PROTECTED_PATH"
      parameters: { patterns: ["/archive/**", "/tax-records/*"] }
      action: "BLOCK_AND_REVIEW" # Forces interactive review if items in path are modified
  ```

#### USP 4: Immutable Audit Ledger & Unified Log Architecture
* **The Problem**: Standard log files roll over or lack proof that historical conflict resolutions or deletions were approved by an operator.
* **The Solution**: Pure-Go SQLite database (`modernc.org/sqlite`) with `pressly/goose` embedded migrations, database-level `BEFORE UPDATE`/`BEFORE DELETE` abort triggers, and SHA-256 hash chaining (`prev_hash` $\rightarrow$ `hash`).
* **Usage Example**:
  ```bash
  # Verify ledger cryptographic integrity via CLI
  rclone-bisync audit verify

  # Output:
  # [✓] Cryptographically Verified (142 entries)
  # [✓] SHA-256 Chain Intact from Genesis
  # [✓] Zero Tampering Detected
  ```

#### USP 5: Native Single-Binary Architecture & Embedded UI
* **The Problem**: Competitor tools require Python virtual environments, Node.js runtimes, npm build steps, or separate frontend servers.
* **The Solution**: A single self-contained executable (~17MB) containing the Go backend engine and the Alpine.js frontend SPA embedded via `//go:embed`.
* **Usage Example**:
  ```bash
  # Download, grant execution permissions, and launch immediately
  curl -L -o rclone-bisync https://github.com/<your-username>/rclone-bisync/releases/latest/download/rclone-bisync-linux-amd64
  chmod +x rclone-bisync
  ./rclone-bisync dashboard
  ```

---

## 4. Community Scaffolding & Governance

To manage contributions while retaining absolute control over code merges:

### 4.1 Contributor Expectations (`CONTRIBUTING.md`)
* Expresses gratitude for community interest while stating that architectural consistency is paramount.
* Requires users to open an issue and discuss feature proposals **before** submitting pull requests.
* Mandates that all PRs pass `./scripts/check.sh` with 100% clean static checks and test coverage $\ge 80\%$.

### 4.2 Maintainer Governance (`GOVERNANCE.md`)
* Explicitly defines the **Maintainer-Led Governance Model**.
* Declares that the Lead Maintainer holds sole decision authority over merges, architecture, releases, and roadmap priorities.
* Specifies Apache License 2.0 terms for all accepted submissions.

### 4.3 AI Workflow Disclosure (`AI_DISCLOSURE.md`)
* Documents the use of AI development tools (Gemini Flash 3.8 / LLM engineering workflows).
* Explains that AI is utilized for refactoring, test suite generation, and documentation drafting under strict human review.
* Reaffirms that the maintainer evaluates, tests, and assumes full responsibility for all released code.

---

## 5. Phased Public Launch Execution Roadmap

| Phase | Target Milestone | Execution Actions |
| :--- | :--- | :--- |
| **Phase A: Pre-Flight Audit** | Codebase & Secret Scrub | 1. Run local `gitleaks detect --verbose` across full git commit history.<br>2. Confirm Apache 2.0 `LICENSE` file is present in repository root.<br>3. Run `./scripts/check.sh` on fresh clone to verify zero build or test regressions. |
| **Phase B: GitHub Infrastructure** | Scaffolding & Pages | 1. Push repository to GitHub.<br>2. Enable **Branch Protection Rules** on `main` (Require 1 review, restrict direct push, require status checks).<br>3. Commit `.github/workflows/ci.yml`, `.github/CODEOWNERS`, and `.github/ISSUE_TEMPLATE/` files.<br>4. Configure GitHub Pages deployment from `docs/` or `gh-pages` branch. |
| **Phase C: Public Launch** | Visibility & Beta Release | 1. Change repository visibility to **Public**.<br>2. Draft Release `v1.0.0-beta.1` with cross-compiled binaries (`linux/amd64`, `linux/arm64`) attached.<br>3. Publish the GitHub Pages landing site. |
| **Phase D: Post-Launch Execution** | Governance Implementation | 1. Begin development on **Phase 5 Milestone**: Immutable Audit Ledger & Unified Log Architecture (`internal/audit`, `partials/view_audit.html`).<br>2. Triage incoming community feedback while keeping branch protection locked. |