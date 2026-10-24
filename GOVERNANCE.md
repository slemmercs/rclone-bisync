# Governance

## Governance Model

Bisync Hub follows a maintainer-led governance model.

The Lead Maintainer retains final decision authority over:

- Architecture
- Roadmap
- Security decisions
- Release management
- Pull request acceptance
- Product direction

Community participation is strongly encouraged.

## Open Source Licensing

Bisync Hub is licensed under the Apache License 2.0.

The project welcomes contributions from the community and follows the principle that accepted contributions become part of the Apache-licensed codebase.

For contribution terms, refer to:

- LICENSE
- NOTICE
- CONTRIBUTING.md

---

## Project Priorities

The project prioritizes:

1. User safety
2. Reliability
3. Maintainability
4. Simplicity
5. Transparency

Feature quantity is not a primary goal.

A smaller, coherent system is preferred over an oversized and difficult-to-maintain platform.

---

## Decision Making

Community feedback is welcome through:

- Issues
- Discussions
- Pull Requests

However, acceptance of contributions remains at the discretion of the maintainer.

---

## Architectural Direction

Bisync Hub is evolving toward:

"Review. Decide. Sync."

The project's primary focus is:

- Rclone Bisync usability
- Synchronization safety
- Conflict visibility
- Deletion review
- Guided recovery workflows

rather than becoming a generic infrastructure management platform.

---

## Maintainer Responsibilities

The maintainer is responsible for:

- Reviewing code
- Managing releases
- Preserving architectural consistency
- Protecting project security
- Maintaining documentation

The long-term health of the project takes precedence over short-term feature growth.

---

## Branch Protection & Review Enforcement

To guarantee software reliability, zero regression, and code integrity, the `main` branch is protected by strict policies:

- **Sole Maintainer Approval**: All changes require a Pull Request approved by the Code Owner (`@slemmercs`).
- **Automated Cloud Quality Gate**: All GitHub Actions CI checks (formatting, staticcheck, govulncheck, betterleaks, test coverage $\ge 80\%$, cross-platform build) must pass before merging.
- **Dismiss Stale Approvals**: Approvals are automatically dismissed when new commits are pushed to an open pull request.
- **No Direct or Force Pushes**: Direct pushes and force pushes (`git push --force`) to `main` are strictly forbidden.
- **Linear & Atomic Git History**: Squash merging or clean rebase merges are enforced to maintain an auditable release trail.

