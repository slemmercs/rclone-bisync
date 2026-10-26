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

When proposing non-trivial architectural decisions or contract shifts, create an [Architecture Decision Record (ADR)](adr/README.md).

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

- unit tests (`go test -count=1 ./...`)
- integration and end-to-end smoke tests
- linting and static analysis (`go vet ./...`)
- repository quality gate checks

Contributors should run the pre-flight quality check before opening or updating pull requests:

```bash
# Run all quality stages (fmt, lint, templates, loc, test, build)
./scripts/check.sh all

# Or run individual checks during inner-loop development
./scripts/check.sh fmt
./scripts/check.sh test
```

Verification is part of implementation, not an afterthought.

---

# 5. Avoid Debugging Spirals

If repeated attempts fail to solve a problem:

- Stop.
- Reassess assumptions.
- Review the design.
- Seek feedback.

Enforce a **3-try limit**: after 2–3 unresolved attempts on a bug or test flake, halt, summarize the hypothesis, and reassess before continuing. Persistent trial-and-error often indicates an architectural misalignment.

---

# 6. Commit Working Checkpoints

Commits should represent coherent, working units of change.

Prefer:

- small, focused commits
- conventional commit format (`feat(...)`, `fix(...)`, `chore(...)`, `docs(...)`)
- verified changes passing local tests

Avoid large batches of unrelated edits.

---

# 7. Review Architecture, Not Just Code

A successful contribution is more than working code.

Reviewers should evaluate:

- safety implications and circuit breakers
- architectural consistency
- testability and statement coverage (>80%)
- maintainability and lines-of-code thresholds
- adherence to product and security principles

---

# 8. Respect Product Principles

All contributions must align with the core [Product Principles](PRODUCT_PRINCIPLES.md):

1. **Safety Before Automation**
2. **Human Decision Authority**
3. **Explainability Over Black Boxes**
4. **Review Before Mutation**
5. **Audit Significant Decisions**
6. **Profiles Are The Primary User Object**
7. **Rclone Remains The Source Of Truth**
8. **Local First**

Features that conflict with these principles should be reconsidered regardless of implementation quality.

---

# 9. Preserve Rclone Boundaries

Bisync Hub builds on Rclone.

Contributions should avoid duplicating responsibilities already handled effectively by Rclone. Rclone remains responsible for file transport, cloud vendor APIs, and direct storage operations. Bisync Hub provides safety, orchestration, governance, and human review.

---

# 10. Leave The System Better Than You Found It

Contributors are encouraged to improve:

- documentation
- test coverage and mock reliability
- observability and logging clarity
- maintainability and component cohesion

while implementing new functionality. Every change should move the project forward.

---

## Related Documentation

- [Architecture Principles](ARCHITECTURE_PRINCIPLES.md)
- [Product Principles](PRODUCT_PRINCIPLES.md)
- [Engineering Principles](ENGINEERING_PRINCIPLES.md)
- [Security Principles](SECURITY_PRINCIPLES.md)
- [Architecture Decision Records (ADRs)](adr/README.md)

