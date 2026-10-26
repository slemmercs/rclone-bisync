# Contributor Workflow

## Purpose

This document describes the preferred workflow for implementing changes in Bisync Hub.

The goal is to maintain code quality, architectural consistency, and operational safety while enabling efficient contributions.

---

# 1. Understand The Problem First

Before making significant changes:

- Understand the user problem.
- Understand the architectural impact.
- Review relevant documentation.

Major work should begin with a clear implementation approach.

---

# 2. Design Before Implementation

Significant changes should be planned before code is written.

Plans should identify:

- affected components
- data flow changes
- architectural impacts
- validation strategy
- testing strategy

---

# 3. Break Work Into Small Tasks

Prefer small, verifiable changes over large multi-purpose commits.

Benefits:

- Easier review
- Easier testing
- Easier rollback
- Reduced risk

---

# 4. Verify Continuously

Changes should be validated as they are implemented.

Examples include:

- unit tests
- integration tests
- linting
- static analysis

Verification is part of implementation, not an afterthought.

---

# 5. Avoid Debugging Spirals

If repeated attempts fail to solve a problem:

- Stop.
- Reassess assumptions.
- Review the design.
- Seek feedback.

Persistent trial-and-error often indicates a deeper issue.

---

# 6. Commit Working Checkpoints

Commits should represent coherent, working units of change.

Prefer:

- small commits
- clear commit messages
- verified changes

over large batches of unrelated edits.

---

# 7. Review Architecture, Not Just Code

A successful contribution is more than working code.

Reviewers should evaluate:

- safety implications
- architectural consistency
- testability
- maintainability
- product principles

---

# 8. Respect Product Principles

All contributions should align with:

- Safety Before Automation
- Human Decision Authority
- Explainability Over Black Boxes
- Review Before Mutation
- Audit Significant Decisions

Features that conflict with these principles should be reconsidered regardless of implementation quality.

---

# 9. Preserve Rclone Boundaries

Bisync Hub builds on Rclone.

Contributions should avoid duplicating responsibilities already handled effectively by Rclone.

---

# 10. Leave The System Better Than You Found It

Contributors are encouraged to improve:

- documentation
- tests
- observability
- maintainability

while implementing new functionality.

Every change should move the project forward.
