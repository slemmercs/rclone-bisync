---
layout: default
title: "4. Cryptographic Audit Ledger"
parent: Core Pillars
nav_order: 4
---

# Cryptographic Immutable Audit Ledger

<span class="badge-online">● Shipped & Available</span>

{: .note }
> **Production Ledger**: The Cryptographic Immutable Audit Ledger is fully implemented and included in the native binary. Every synchronization action, conflict resolution, and file mutation is cryptographically sealed into an append-only SQLite database backed by SHA-256 hash chains.

Compliance, security audits, and forensic debugging require an undeniable, tamper-evident record of all storage mutations: **Who deleted file X? When was conflict Y resolved? What was the decision confidence score?**

**Bisync Hub embeds an immutable, SQLite-backed cryptographic ledger with trigger-enforced append-only constraints.**

---

## Architectural Guarantees

```text
┌────────────────────────────────────────────────────────┐
│                   Ledger Block (N)                     │
│  - Event: FILE_DELETION_CONFIRMED                      │
│  - Path: /documents/tax_2025.pdf                       │
│  - Actor: operator@admin                               │
│  - Previous Hash: 7f83b165...                          │
│  - Block Hash: sha256(PrevHash + Payload + Timestamp)  │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│                  Ledger Block (N+1)                    │
│  - Event: SYNC_CYCLE_COMPLETED                         │
│  - Payload: { files_transferred: 14, bytes: 492102 }   │
│  - Previous Hash: sha256(Block N)                      │
│  - Block Hash: sha256(...)                             │
└────────────────────────────────────────────────────────┘
```

### 1. SQLite Trigger-Enforced Immutability
The internal SQLite database employs schema-level `BEFORE UPDATE` and `BEFORE DELETE` triggers on audit tables:
```sql
CREATE TRIGGER prevent_ledger_tampering
BEFORE UPDATE ON audit_events
BEGIN
    SELECT RAISE(FAIL, 'FATAL: audit_events table is strictly append-only.');
END;
```
Even if an attacker or misconfigured script gains write access to the database file, existing audit entries cannot be modified or purged without breaking database integrity.

### 2. SHA-256 Hash Chaining
Each audit entry computes a SHA-256 digest over the concatenation of:
- `Previous_Record_Hash`
- `Event_Type` and `Timestamp` (UTC millisecond precision)
- `Payload_JSON` (redacted secrets, sanitized paths, decision scores)

The application provides a built-in cryptographic verification command (`bisync-hub verify-ledger`) to audit chain integrity and identify any external byte tampering.

### 3. Redaction by Default
All sensitive credentials—including `RC_PASS`, OAuth refresh tokens, webhook URLs, and private auth headers—are recursively redacted via `security.RedactSecrets` before entering the audit stream.

{: .safety }
> The audit ledger ensures full compliance with operational standards while protecting organizational data sovereignty.

---

## Visual Ledger & Forensic Inspection

### 1. Unified Cryptographic Audit Ledger
The dedicated Audit Ledger interface displays continuous sequence IDs, timestamps, profile targets, actions, and actors. Operators can run live cryptographic verification directly in the browser and export compliance records to CSV or JSONL:

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_audit-ledger-v2.png' | relative_url }}" alt="Unified Cryptographic Audit Ledger" />
  <div class="caption">
    <span class="title">1. Unified Audit Ledger Interface</span>
    <span>Live SHA-256 chain verification, action/actor filters, and compliance export</span>
  </div>
</div>

### 2. Tamper-Evident Block Inspection
Selecting any record opens the forensic block inspector, exposing the previous hash, block SHA-256 digest, execution duration, and fully redacted payload details:

<div class="screenshot-frame">
  <img src="{{ '/assets/images/screenshots/bisync-hub_audit-block.png' | relative_url }}" alt="Cryptographic Audit Block Inspection" />
  <div class="caption">
    <span class="title">2. Cryptographic Block Detail</span>
    <span>Immutable block parameters, chained SHA-256 hashes, and sanitized JSON payload</span>
  </div>
</div>
