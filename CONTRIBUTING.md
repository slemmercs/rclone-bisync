# Contributing

Thank you for your interest in contributing to Bisync Hub.

Contributions of all sizes are welcome.

## Contributor License

By submitting a contribution to this project, you agree that your contribution will be licensed under the Apache License 2.0, unless explicitly stated otherwise.

Contributions intentionally submitted for inclusion in the project are accepted under the same license as the project itself, consistent with Section 5 ("Submission of Contributions") of the Apache License 2.0.

You must have the legal right to submit any code, documentation, or other material you contribute.

---

## Before Opening a Pull Request

For significant changes:

1. Open an issue first.
2. Describe the problem being solved.
3. Discuss the proposed approach.
4. Wait for maintainer feedback before implementation.

This helps avoid duplicate work and architectural drift.

---

## Development Principles

Please review the architectural governance and principles suite before submitting major changes:

- [Architecture Principles](docs/architecture/ARCHITECTURE_PRINCIPLES.md) (Master Charter & Invariants)
- [Product Principles](docs/architecture/PRODUCT_PRINCIPLES.md) (Review. Decide. Sync.)
- [Engineering Principles](docs/architecture/ENGINEERING_PRINCIPLES.md) (Engineering Standards & Modularity)
- [Security Principles](docs/architecture/SECURITY_PRINCIPLES.md) (Safety & Confinement)
- [Contributor Workflow](docs/architecture/CONTRIBUTOR_WORKFLOW.md) (Development Lifecycle & Quality Gate)
- [Architecture Decision Records (ADRs)](docs/architecture/adr/README.md) (Decision Registry)
- [GOVERNANCE.md](GOVERNANCE.md)
- [AI_DISCLOSURE.md](AI_DISCLOSURE.md)

---

## Pull Request Requirements

Pull requests should:

- Have a clear purpose
- Be reasonably scoped
- Include tests where appropriate
- Include documentation updates when required
- Pass CI checks

---

## Pull Request Checklist

- [ ] Code builds successfully
- [ ] Tests added or updated
- [ ] Pre-flight quality gate passed locally (`./scripts/check.sh all`)
- [ ] Documentation updated
- [ ] ADR proposed under `docs/architecture/adr/` (if introducing architectural or schema changes)
- [ ] No secrets introduced
- [ ] Existing tests pass
- [ ] Architecture principles respected


---

## Use of AI

Contributors may use AI-assisted development tools.

However, contributors remain responsible for:

- Understanding their changes
- Testing their changes
- Reviewing generated code
- Maintaining project quality

Generated code that is unreviewed or poorly understood may be rejected.

---

## Good First Contributions

Examples include:

- Documentation improvements
- Test coverage improvements
- Bug fixes
- UX enhancements
- Accessibility improvements
- Installation documentation

---

## Questions

If unsure whether an idea fits the project direction, please open a discussion or issue before investing significant implementation effort.

Thank you for helping improve Bisync Hub.
