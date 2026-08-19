<h1 align="center">OmniVault · 万象档案袋</h1>

**An encrypted, personal-context vault. The key always stays in your hands.**

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/github/license/54wu/omnivault" alt="license"></a>
  <a href="https://github.com/54wu/omnivault/releases"><img src="https://img.shields.io/github/v/release/54wu/omnivault" alt="release"></a>
  <a href="https://github.com/54wu/omnivault/actions"><img src="https://img.shields.io/github/actions/workflow/status/54wu/omnivault/release.yml" alt="ci"></a>
  <img src="https://img.shields.io/github/go-mod/go-version/54wu/omnivault" alt="go">
</p>

---

Fill in your life once. After that, every AI agent starts from full context, not a blank page.

OmniVault stores your identity, documents, relationships, finances, addresses and preferences — every field is encrypted with a key derived from your vault password. Agents only request access within an authorized scope. You approve, tokens expire, and the key never leaves your machine.

This README is the **complete reference** for the project: from features, installation, the UI, the CLI, merge, API, and MCP, to the security model, project layout, building & releasing, and troubleshooting.

> [中文文档](./README.md) | **English**

---

## Table of Contents

1. [Features](#features)
2. [Quick Start](#quick-start)
3. [Installation](#installation)
4. [Native UI (Dossier)](#native-ui-dossier)
   - [Login & First Run](#login--first-run)
   - [Person Management](#person-management)
   - [Two Views: Documents / All Data](#two-views-documents--all-data)
   - [Editable Category Outline](#editable-category-outline)
   - [Field Editing: Custom / Move / Reorder / Hide](#field-editing-custom--move--reorder--hide)
   - [Autosave, Search & Export](#autosave-search--export)
   - [Theme, Language & Security](#theme-language--security)
   - [AI Access (Local Server Toggle)](#ai-access-local-server-toggle)
   - [Single-Process UI Architecture](#single-process-ui-architecture)
5. [Command Line (CLI)](#command-line-cli)
6. [Merge (Three-Tier)](#merge-three-tier)
7. [Encryption / Security Model](#encryption--security-model)
8. [Cross-Device Sync](#cross-device-sync)
9. [Versioned Automatic Backups & Rollback](#versioned-automatic-backups--rollback)
10. [HTTP API](#http-api)
11. [Attachments](#attachments)
12. [Service Tokens](#service-tokens)
13. [Sensitivity Levels](#sensitivity-levels)
14. [Data Model & Field ID Conventions](#data-model--field-id-conventions)
15. [Environment Variables](#environment-variables)
16. [MCP Server](#mcp-server)
17. [Local Model Access (Fully Offline)](#local-model-access-fully-offline)
18. [Local Workflow: Materials → Semantic Normalization → Edge Auto-fill](#local-workflow-materials--semantic-normalization--edge-auto-fill)
   - [One-click all-in-one entry (recommended)](#0-one-click-all-in-one-entry-recommended)
   - [Verification log (2026-08-16, real run)](#31-verification-log-2026-08-16-real-run)
19. [Shopping Demo](#shopping-demo)
20. [Project Layout](#project-layout)
21. [Build & Development](#build--development)
22. [Release](#release)
23. [Testing](#testing)
24. [FAQ](#faq)
25. [Protocol](#protocol)
26. [Credits](#credits)
27. [Contributing](#contributing)
28. [Security](#security)
29. [License](#license)

---

## Features

- **Per-field encryption** — every field is individually encrypted with AES-256-GCM.
- **Zero knowledge** — the vault password is never stored; the key only exists in memory while unlocked.
- **Native UI** — single-process WebView2 native window on Windows 10/11. No background daemon, no TCP port, no network dependency — nothing for a network filter driver to intercept.
- **Multiple persons** — one vault can manage several people (family members, colleagues…), each stored and archived independently and encrypted.
- **Document templates** — place the documents each person needs (resume, medical exam, onboarding form…) from a template library. A template is only a reference to underlying fields; data is shared with the "All data" view.
- **Editable category outline** — the left outline supports renaming, adding, and deleting categories; fields can be moved between categories.
- **Dynamic CLI fields** — any category and field name; `set/get/list` work immediately.
- **Command line** — a full CLI for scripting and agent use.
- **Merge** — human-in-the-loop three-tier merging of external material (e.g. a filled application form) into a person's dossier.
- **MCP server** — a built-in TypeScript [MCP](https://modelcontextprotocol.io/) server that lets AI agents request scoped access to your context.
- **Versioned automatic backups** — automatic snapshots after unlock and after every write, with rollback support.
- **Cross-device sync** — the encrypted `vault.db` can be synced anywhere; the key always stays on your machine.
- **Attachments** — multiple encrypted attachments per field.
- **Service tokens** — long-lived, scope-limited credentials for apps/agents.
- **Sensitivity levels** — four tiers per field: `public / standard / sensitive / critical`.
- **Audit log** — every access is written to `vault_access_log` and can be inspected.
- **Security question** — after 5 wrong password attempts, a recovery security question can reset the password.

---

## Quick Start

```sh
# Windows (double-click):
#   run OmniVault.exe at the repo root; the first run
#   walks you through creating the vault inside the window.

# macOS / Linux:
curl -fsSL https://raw.githubusercontent.com/54wu/omnivault/main/install.sh | sh
omnivault onboard            # interactive: create vault + unlock + fill common fields
```

Out of the box: `omnivault ui` opens the native UI; `omnivault set identity.full_name "Cool Cucumber"` writes a field; `omnivault get identity.full_name` reads it back.

> Save your secret key somewhere safe. Unlocking requires **both the vault password and the secret key**.

---

## Installation

### Windows

Download the latest `omnivault.exe` from the [Releases](https://github.com/54wu/omnivault/releases) page, or build from source:

```sh
git clone https://github.com/54wu/omnivault.git
cd omnivault
./build.ps1            # one-click build -> OmniVault.exe at the repo root
```

Double-click `OmniVault.exe`; the first run guides you through creating the vault inside the window.

### macOS / Linux

```sh
curl -fsSL https://raw.githubusercontent.com/54wu/omnivault/main/install.sh | sh
omnivault onboard
```

<details>
<summary>Option 2: build from source</summary>

```sh
git clone https://github.com/54wu/omnivault.git
cd omnivault && make build && make install
omnivault onboard
```
</details>

<details>
<summary>Option 3: install with Go directly</summary>

```sh
go install github.com/54wu/omnivault/cmd/omnivault@latest
omnivault onboard
```
</details>

`install.sh` supports macOS (arm64/amd64) and Linux (arm64/amd64). It downloads the matching archive from GitHub Releases and, on macOS, performs an ad-hoc signature so the binary passes Gatekeeper.

---

## Native UI (Dossier)

Double-click `omnivault.exe` (or run `omnivault ui`) to open the vault in a native WebView2 window titled **OmniVault · 万象档案袋**.

### Login & First Run

- **First run**: the window shows a "Create vault" form. Set a vault password (at least 8 characters), and a **secret key** is generated and displayed — save it somewhere safe.
- **Later runs**: enter the vault password to unlock and enter the main screen.
- The progress bar at the top shows the current person's completion (`filled / total`), **computed uniformly over all of that person's fields**, so it is identical in both the "Documents" and "All data" views.

### Person Management

The top person bar shows each person in the vault as a tab:

- **+ Add person**: create a new dossier (each person has an independent data namespace and never mixes data).
- **✎ Rename**: change the displayed name (does not affect underlying data).
- **× Delete**: delete the dossier and cascade-delete all of that person's fields and attachments (with confirmation).
- Switch tabs to view / fill different persons.

### Two Views: Documents / All Data

Toggle in the top-right of the toolbar:

- **Documents**: organize the current person by "documents". Use "+ Add document" on the left to choose which documents this dossier needs (resume, medical exam, onboarding form…). Each template is only a reference to underlying fields; data is shared with "All data" — fill once, sync everywhere. Within a document you can add/remove entries, save as a template, and unplace documents.
- **All data**: the current person's full flat form, grouped by category, showing all fields (including underlying fields and custom fields).

### Editable Category Outline

The left "Category outline" (in the "All data" view) lists all categories. Click to jump to a section, and additionally:

- **✎ Rename category**: change a category's display name and description.
- **× Delete category**: hide a category from the current person's "All data" view (the underlying field data stays in the vault).
- **+ Add category**: create a new custom category, then add fields to it; new categories also appear in the "Category" and "Move field" pickers.

All edits are persisted per person and survive restarts.

### Field Editing: Custom / Move / Reorder / Hide

- **+ Field**: create a custom field in the current category (field ID, display label, type: text/password/email/tel/date/textarea, sensitivity, placeholder).
- **Category**: choose the target category when creating a field (including user-added categories).
- **Move field**: move a field to another category; its saved data is migrated too.
- **Drag to reorder**: fields can be reordered by drag (saved per person / per document); category sections can also be reordered.
- **Remove / Delete**: non-custom fields can be removed from the dossier (underlying data is kept); custom fields can be deleted (filled content is removed, with confirmation).
- Fields show their sensitivity tier via a colored dot (public/standard/sensitive/critical).

### Autosave, Search & Export

- **Autosave**: a field saves when it loses focus (or on Enter), with saved/saving/error status hints; a base field appearing in several templates syncs live.
- **Search**: the top search box filters by label or content and switches to the "All data" view automatically.
- **Export**: export the current person's filled information as **JSON / Markdown / HTML**.

### Theme, Language & Security

- The **◐** button in the top-right toggles dark/light theme (light by default); the preference is persisted locally.
- The **EN** button toggles the Chinese/English interface.
- **Change password**: re-encrypts all fields (including attachments) with the new password; do not close the window during this.
- After 5 wrong password attempts, the recovery security question is offered.

### AI Access (Local Server Toggle)

To let an external AI / MCP client reach the vault, open the **"Enable local server"** toggle in the "AI Access guide" dialog. This listens on `127.0.0.1:7200` inside the same process (same effect as `omnivault unlock`). Closing the toggle or the window stops the listener and wipes the in-memory key (**close the window to shut down**); no background process management needed.

### Single-Process UI Architecture

The UI talks to the vault **in a single process** through a Go bridge (WebView2 Bind); by default no HTTP is involved, so there is no TCP port or named pipe and the vault can never become "unreachable". The UI HTML is embedded in the binary and written to `~/.omnivault/ui-data/onboarding.html` at each startup, then loaded from a **`file://` origin** so the browser's `localStorage` (person list, field order, custom fields, theme, language…) truly persists across restarts.

---

## Command Line (CLI)

```sh
omnivault onboard                        # create vault, unlock, and fill common fields (interactive)
omnivault init                           # create a new vault
omnivault ui                             # open the native window UI (single-process)
omnivault status                         # show vault status
omnivault schema                         # show recommended field names (--json for raw JSON)

omnivault set <id> <value>               # set a field
omnivault get <id>                       # get a field value
omnivault list [category]                # list fields
omnivault delete <id>                    # delete a field
omnivault set-sensitivity <id> <tier>    # set sensitivity (public|standard|sensitive|critical)
omnivault export                         # export all decrypted fields as JSON
omnivault audit                          # show access audit log

omnivault merge <material.json>          # merge external material into a person's dossier (three-tier)
omnivault backup <dest>                  # copy the encrypted vault.db to a sync folder (key stays local)
omnivault restore <src>                  # copy a synced vault.db back into the vault
omnivault rollback [name]                # list versioned backups, or restore one
omnivault set-security-question          # set the recovery security question (asked after 5 wrong passwords)
omnivault change-password                # change the vault password (re-encrypts all fields)

omnivault create-service-token <consumer> # create a long-lived service token
omnivault list-service-tokens            # list active service tokens
omnivault revoke-service-token <prefix>  # revoke a service token by prefix

# Daemon (for MCP / HTTP consumers only)
omnivault unlock                         # unlock (start the background serving daemon)
omnivault lock                           # lock (stop the daemon, wipe keys)
omnivault serve                          # run the serving daemon in the foreground

omnivault help                           # print help for all commands
```

Fields use dot notation: `identity.full_name`, `addresses.current.city`, `financial.filing_status`. Any category and field name may be used.

---

## Merge (Three-Tier)

Merge external material (e.g. an application form exported from a recruiting site) into a person's dossier — human-in-the-loop, with automatic backup and full auditing. Available through the **CLI**, **HTTP**, and **MCP**.

### Three-Tier Flow

1. **Ownership** — the material's `person_hint` is cross-checked against existing persons (name / ID number / phone / email weighting, 0.25/0.35/0.25/0.15). When ownership is ambiguous, candidates are returned for you to pick (`--person <id>` / `person` forces a target).
2. **Classify** — each material item is mapped to a vault field and tagged:
   - `auto` → identical value (no-op) or a safe fill of a blank low-sensitivity field (can be adopted automatically)
   - `batch` → conflict on a low-sensitivity field (needs a user decision)
   - `manual` → sensitive/critical field or list-merge (needs a user decision)
3. **Decide & apply** — you set `action` (keep/replace/fill/add/skip) on the batch/manual items; only those actions are written. Writes trigger an automatic `BackupNow()` snapshot and are audited.

> Labels that do not map to any vault field are ignored with a notice — nothing is written.

### CLI Usage

```sh
# dry-run: generate a decision plan (printed to stdout)
omnivault merge material.json --person p1

# save the decision plan to a file
omnivault merge material.json --person p1 --plan plan.json

# adopt all auto items directly and print the remaining pending items
omnivault merge material.json --person p1 --auto

# apply a decision plan (only writes fill/replace/add actions)
omnivault merge --apply plan.json
```

Material file format (see `merge-material-sample.json` at the repo root):

```json
{
  "source": "example.com",
  "person_hint": { "name": "徐小明", "id_number": "440101199501011234", "phone": "13900001111", "email": "xuxiaoming@example.com" },
  "items": [
    { "label": "硕士院校", "value": "示例大学" },
    { "label": "手机号", "value": "13900001111" }
  ]
}
```

### Label → Field Mapping

Chinese labels in the material (e.g. "硕士院校", "手机号") are resolved to canonical fields (e.g. `identity.email`, `education.postgrad_school`), then assembled with the person prefix into the actual stored field ID (e.g. `p1_identity.email`). The resolvable label set covers the common Chinese names of dossier fields.

---

## Encryption / Security Model

```
Vault Password + Secret Key (128-bit)
  → Argon2id KDF (64MB, 3 iterations)
  → Vault Key (256-bit, in-memory only)
  → HKDF per category → Category Subkeys
  → AES-256-GCM per field (12-byte random nonce)
```

- The vault password is **never stored**; the secret key lives at `~/.omnivault/secret.key` (mode 0600) and is never synced or stored in the database.
- While unlocked, the master key exists **only in memory**, zeroed on lock / window close.
- Every field is **individually encrypted** — even if the database is stolen, all that's obtained is ciphertext.
- Auto-lock after 30 minutes of inactivity (each authenticated request resets the timer).
- Session token: 32 bytes from `crypto/rand`, compared in constant time.
- Every access is written to the `vault_access_log` audit.

---

## Cross-Device Sync

Think of the vault as a safe: the safe (encrypted `vault.db`) can travel to any device, but the key (`secret.key`) always stays with you.

```
Device A                       Cloud                        Device B
omnivault backup D:\sync   →    vault.db  →→  sync →→   omnivault restore D:\sync
(secret.key stays local)     (ciphertext only)          + enter password and key once
```

1. `omnivault backup <folder>` — copy only the encrypted `vault.db` to your sync folder (OneDrive, Nutstore, git…); the key is never included.
2. On the other device, `omnivault restore <folder>` copies the synced `vault.db` back into its vault directory.
3. `omnivault unlock` — enter the vault password and key once. The key can be typed manually; only `vault.db` participates in sync.

---

## Versioned Automatic Backups & Rollback

The vault automatically snapshots the encrypted database:

- Triggered **after each unlock** and **after each write** (3-second debounce).
- Stored in `~/.omnivault/backups/` as `vault-YYYYMMDD-HHMMSS.db`.
- Keeps the most recent **3**; older ones are cleaned up automatically.

```sh
omnivault rollback                      # list versioned backups
omnivault rollback vault-20260815-120000.db   # restore a specific backup (run lock first)
```

---

## HTTP API

The serving daemon runs at `http://127.0.0.1:7200`; protected endpoints require `Authorization: Bearer <token>`.

```
GET    /vault/status                    # vault status (public)
GET    /vault/schema                    # recommended field names & sensitivity (public)
GET    /ui                              # onboarding form (public)
POST   /vault/unlock                    # unlock → session token

GET    /vault/fields                    # list field metadata (no values)
GET    /vault/fields/{id}               # get a field with its decrypted value
PUT    /vault/fields/{id}               # set a field ({ value, sensitivity? }, upsert)
DELETE /vault/fields/{id}               # delete a field
GET    /vault/fields/category/{name}    # all fields in a category (with values)

GET    /vault/context                   # full decrypted dump grouped by category

PUT    /vault/sensitivity/{id}          # update sensitivity

POST   /vault/merge/plan                # merge dry-run: classify material into a plan (no writes)
POST   /vault/merge/apply               # apply a merge decision plan (only fill/replace/add)

POST   /vault/tokens/service            # create a service token
GET    /vault/tokens/service            # list service tokens
DELETE /vault/tokens/service/{prefix}   # revoke a service token

POST   /vault/lock                      # lock (wipe keys)
GET    /vault/audit?limit=50            # access audit log
```

Example `GET /vault/context` response:

```json
{
  "categories": {
    "p1_identity": [
      { "id": "p1_identity.name", "category": "p1_identity", "field_name": "name", "value": "徐小明", "sensitivity": "standard" }
    ],
    "p1_education": [...]
  }
}
```

---

## Attachments

Each field can hold multiple attachments. Upload encrypted binary content, then list, download, or delete per field.

```
POST   /vault/attachments?field=<id>    # upload an attachment (multipart, X-Filename header)
GET    /vault/attachments?field=<id>    # list a field's attachments
GET    /vault/attachments/{id}          # download an attachment
DELETE /vault/attachments/{id}          # delete an attachment
```

Attachments are encrypted with AES-256-GCM and re-encrypted when the vault password changes. Deleting a field cascades to its attachments.

---

## Service Tokens

Service tokens let applications authenticate with the vault using long-lived, scope-limited credentials (following the 1Password service account pattern).

```sh
omnivault create-service-token tax-agent --scope "identity.*,financial.*" --ttl 1h
omnivault create-service-token life --scope "*" --ttl 8760h
omnivault list-service-tokens
omnivault revoke-service-token abc123
```

Each authenticated request resets the 30-minute auto-lock timer, keeping the vault unlocked while a consumer is active.

---

## Sensitivity Levels

| Tier | Examples | Behavior |
|------|----------|----------|
| `public` | Name, timezone | Auto-shared with authorized consumers |
| `standard` | Address, employer | Shared on request, logged |
| `sensitive` | DOB, phone, tax status | Requires explicit approval |
| `critical` | SSN, card number, card expiry | Requires approval + verification |

```sh
omnivault set-sensitivity identity.id_number critical
omnivault set-sensitivity preferences.timezone public
```

New fields default to `standard`. Recommended field names and default tiers are shown by `omnivault schema`.

---

## Data Model & Field ID Conventions

- **Underlying storage**: fields live in the SQLite `vault_fields` table, with IDs shaped `category.field_name` (e.g. `identity.name`, `p1_education.edu1_school`).
- **CLI / HTTP (single-person model)**: use `category.field` (e.g. `identity.full_name`). Any category and field name may be used.
- **UI (multi-person model)**: each person has an internal person ID (`p1`, `p2`…); their field IDs carry a person prefix: `pN_category.field`, e.g. `p1_identity.name`. All fields of one person share that prefix.
- **Person ID uniqueness**: a new person is always assigned an ID that does **not** collide with any existing data (scanning both the person list and existing `pN_` prefixes in the vault), so a new dossier is always clean and never inherits anyone else's data.
- **Orphan-dossier auto-recovery**: on startup, the app scans the vault for existing `pN_` prefixes; any person without a tab is automatically restored as one (display name taken from its `identity.name` field), so data is never invisible due to lost local settings.
- **Merge bridge**: the merge tool resolves Chinese labels to canonical fields, then assembles them with the target person's prefix into `pN_category.field` for writing.
- **Attachments**: stored in `vault_attachments`, associated by field ID.

The UI ships with 30+ categories covering: identity, addresses, employment, education, financial & social security, bank/credit cards, medical, emergency contacts, documents, social accounts, family, internships & projects, certificates & awards, career objectives, work experience, project experience, skills, certificates & honors, languages, self-evaluation, portfolio & links, training, publications, patents, student work / social practice, insurance, assets, subscriptions & memberships, devices & warranties, security notes, contacts, preferences, and more. The education category also includes domestic schooling stages (kindergarten / primary / junior high / senior high) and higher education (Education 1/2/3) sub-sections.

---

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `VAULT_DIR` | `~/.omnivault` | Vault directory |
| `VAULT_ADDR` | `http://127.0.0.1:7200` | Server address used by the CLI |
| `VAULT_PORT` | `7200` | Listen port for `serve` |
| `VAULT_TOKEN` | — | Service token (MCP/HTTP consumers; overrides the session file) |
| `VAULT_SCOPE` | — | Field scope allowed to the consumer (e.g. `*` or `identity.*,financial.*`) |
| `VAULT_CONSUMER` | — | Consumer name (used in the audit log) |

---

## MCP Server

The project ships a TypeScript [MCP](https://modelcontextprotocol.io/) server that lets AI agents access your personal context. Full details: [mcp/README.md](./mcp/README.md).

### Install & Register

```sh
npm install omnivault-mcp          # or
npx -y omnivault-mcp@latest

# Register with Claude Code
claude mcp add vault -- npx -y omnivault-mcp@latest
```

### Tools

| Tool | Description |
|------|-------------|
| `vault_status` | Check whether the vault is running and unlocked |
| `vault_get` | Get a single decrypted field by ID |
| `vault_list` | List all field metadata (no values) |
| `vault_context` | Get all decrypted fields grouped by category |
| `vault_set` | Set an encrypted field value in the vault |
| `vault_merge_plan` | Merge dry-run: classify a material source into a decision plan (no writes) |
| `vault_merge_apply` | Apply a merge decision plan (only fill/replace/add; auto backup + audit) |

### Merge Workflow (MCP)

`vault_merge_plan` + `vault_merge_apply` implement the human-in-the-loop three-tier merge (the same `internal/merge` logic as CLI/HTTP):

1. **Ownership** — the material's `person_hint` is cross-checked against existing persons; candidates are returned when ambiguous (`person` forces a target).
2. **Classify** — `auto` (identical value / safe fill of a blank low-sensitivity field) / `batch` (low-sensitivity conflict) / `manual` (high-sensitivity or list merge).
3. **Decide & apply** — set `action` (keep/replace/fill/add/skip) on batch/manual items; `vault_merge_apply` writes only those actions and auto-backups + audits.

Example `material` argument for `vault_merge_plan`:

```json
{
  "source": "example.com",
  "person_hint": { "name": "徐小明", "id_number": "4401...", "phone": "1390..." },
  "items": [
    { "label": "硕士院校", "value": "示例大学" },
    { "label": "手机号", "value": "13900001111" }
  ]
}
```

### Authentication

The MCP server resolves the token on every request:

1. `VAULT_TOKEN` env var — with service tokens, for always-on agents.
2. `~/.omnivault/.session` — the session token produced by `omnivault unlock`.

Because the token is resolved per request, the server survives vault lock/unlock cycles.

### Error Messages

| Error | Meaning |
|-------|---------|
| `vault: server not running` | Run `omnivault unlock` first to start the server |
| `vault: session expired` | Run `omnivault unlock` to refresh the session |
| `vault: vault is locked` | Run `omnivault unlock` to decrypt |
| `vault: not found` | The field ID doesn't exist |
| `vault: not configured` | No token — run `omnivault unlock` or set `VAULT_TOKEN` |

### `.mcp.json` Example

In a project-root `.mcp.json`, point directly at the compiled output and inject environment variables:

```json
{
  "mcpServers": {
    "vault": {
      "command": "node",
      "args": ["<absolute-project-path>\\mcp\\dist\\index.js"],
      "env": {
        "VAULT_ADDR": "http://127.0.0.1:7200",
        "VAULT_DIR": "C:\\Users\\<username>\\.omnivault",
        "VAULT_SCOPE": "*",
        "VAULT_CONSUMER": "trae"
      }
    }
  }
}
```

The MCP source lives in `mcp/` (TypeScript; compiled output in `mcp/dist/`). After editing, run `npm run build` to recompile.

---

## Local Model Access (Fully Offline)

**One important clarification first**: OmniVault's MCP server itself **does not call an LLM** — it only exposes the vault's read/write tools to an AI agent. The component that actually calls the model (and decides where `base_url` / `api_key` / `model` point) is **your agent host**, such as Trae, Claude Code, Cursor, etc. So the local/cloud switch lives in the **agent host's model configuration**, not in OmniVault's code.

If your core requirement is that **all materials and information are received fully offline, with data never leaving the machine**, then:

1. Run an open-source model locally with **Ollama** (7B class is enough; no discrete GPU needed — a CPU with 16GB RAM runs it smoothly).
2. Ollama exposes an **OpenAI-compatible endpoint**: `http://localhost:11434/v1`.
3. Point the agent's `base_url` at it and set `model` to the local model name — drop-in replacement, fully offline.

### Why a 7B model is enough

"Read / write the vault" are **deterministic API calls** that don't consume the model at all; the only part of "web form filling" that needs a model is **field semantics alignment** (a webpage's "联系电话" → the vault's `contact.phone`), which 7B-class models (Qwen2.5-7B / Qwen3-8B) handle reliably. The demand is on "getting it right", not "bigger parameters".

### Local / Cloud switchable configuration

In the agent host, factor the model config into environment variables (or a single toggle) to switch instantly:

```sh
# ---- Local (fully offline): Ollama provides an OpenAI-compatible endpoint ----
export LLM_BASE_URL="http://localhost:11434/v1"
export LLM_API_KEY="ollama"          # Ollama accepts any placeholder
export LLM_MODEL="qwen2.5:7b"        # or qwen3:8b

# ---- Cloud: any OpenAI-compatible provider ----
export LLM_BASE_URL="https://api.openai.com/v1"
export LLM_API_KEY="sk-xxxxxxxx"
export LLM_MODEL="gpt-4o"
```

> Agent hosts configure this in different places (Trae in its model settings, Claude Code via `ANTHROPIC_BASE_URL`, etc.), but the idea is the same: **as long as a provider exposes an OpenAI-compatible endpoint, just point `base_url` at it**. Ollama's `http://localhost:11434/v1` is exactly such a compatible endpoint.

### Connecting common agent hosts to local models

Newer Ollama versions ship an `ollama launch` command that binds a local model and configures `base_url` for you automatically — **no environment variables to hand-fill**. After pulling a local model (e.g. `qwen3:8b`), just run:

```sh
# Hermes Agent (yours)
ollama run hermes --model qwen3:8b

# OpenCode (yours)
ollama launch opencode --model qwen3:8b

# Other hosts: Claude Code / Qwen Code / VS Code
ollama launch claude --model qwen3:8b
ollama launch qwen   --model qwen3:8b
ollama launch vscode --model qwen3:8b
```

If your host does not support `ollama launch`, point `base_url` / `model` at the local endpoint manually:

| Agent host | Configuration | Local value |
|-----------|---------------|-------------|
| Hermes Agent | env `OPENAI_BASE_URL` / `OPENAI_MODEL` | `http://localhost:11434/v1` / `qwen3:8b` |
| OpenCode | `opencode` config or env `OPENAI_BASE_URL` | `http://localhost:11434/v1` / `qwen3:8b` |
| Trae IDE | add a custom model in model settings | `http://localhost:11434/v1` / `qwen3:8b` |
| WorkBuddy | model config with an OpenAI-compatible endpoint | `http://localhost:11434/v1` / `qwen3:8b` |

> Either way you only need to point `base_url` at `http://localhost:11434/v1` and set `model` to `qwen3:8b` to work fully offline with data staying on your machine.

### Deploy fully offline with Ollama

```sh
# 1. Install Ollama (download the installer on Windows, or)
curl -fsSL https://ollama.com/install.sh | sh

# 2. Pull a 7B-class model that is stable for Chinese and tool calling, and runs on CPU
ollama pull qwen2.5:7b          # recommended: ~4.7GB quantized, solid tool calling
# ollama pull qwen3:8b          # optional if you have spare RAM

# 3. Start the local service (defaults to port 11434; OpenAI-compatible endpoint at /v1)
ollama serve

# 4. Verify (should return the model list; no network needed)
curl http://localhost:11434/v1/models
```

### Recommended local setup for this project

| Hardware | Recommended model | Capability |
|----------|-------------------|------------|
| Current machine (CPU + 16GB RAM, no discrete GPU) | `qwen2.5:7b` (or `qwen3:8b`) | Read/write vault, field semantics alignment, multi-step form filling, fully offline |
| Lower spec (<16GB RAM) | `qwen2.5:3b` | Simple lookups / form filling where fields line up, offline |
| With a discrete GPU (RTX 4060 8GB+) | `qwen2.5:14b` and up | More complex dynamic pages, faster responses |

Afterward the agent connects to OmniVault via MCP (see [MCP Server](#mcp-server) above), and the model runs through the local Ollama endpoint — **materials and information never leave the machine**.

---

## Local Workflow: Materials → Semantic Normalization → Edge Auto-fill

The `tools/ocr2text/` directory ships scripts that chain a fully offline pipeline: turn everyday materials (Word / PDF / images / text) into Markdown, have a local model run field-semantic normalization, and finally **take over your real open Edge** to auto-fill forms.

> Prereq: the project uses an isolated Python 3.12 venv with dependencies pre-installed (RapidOCR, PyMuPDF, python-docx, Playwright).

### 0) One-click all-in-one entry (recommended)

Don't want to run each step by hand? **Double-click `tools/ocr2text/omniflow.bat`** (or run `python tools/ocr2text/omniflow.py`) to open the interactive program, which walks you through:

1. **① Materials folder** — drag in / type the folder containing Word/PDF/images;
2. **② Service token** — optional (obtained from OmniVault UI "AI Access" → Enable local server);
3. **③ Vault server address** — optional, defaults to `http://127.0.0.1:7200`.

It then automatically:
1. Detects / starts **Ollama** (auto-locates the real model directory) and verifies the **qwen3:8b** model;
2. Detects the **vault local server** (prompts you to start it if not running);
3. Runs **materials → text** (`convert.py`) → **field normalization** (`normalize.py`);
4. After normalization shows an interactive menu for your next step:
   - `[1]` **Write to vault** — PUTs the normalized fields into OmniVault as archive data;
   - `[2]` **Take over Edge to fill a form** — reads vault fields and fills the Edge launched by `edge_start.bat`;
   - `[3]` Only output `_normalized.json` locally, no write / no form fill.

Advanced (skip some prompts): you can pre-specify via command line; anything unspecified is still asked interactively:

```sh
python tools/ocr2text/omniflow.py "your-materials-folder" --token "your-service-token"
```

| Option | Purpose |
|--------|---------|
| `--token <TOKEN>` | Service token; with it the write/fill uses HTTP, no interactive unlock |
| `--addr <URL>` | Vault server address (default `http://127.0.0.1:7200`) |

Normalization disables **qwen3 thinking** (`/api/chat` + `think:false`) to avoid slow reasoning traces or timeouts.

> Note: besides `--token`, the `VAULT_TOKEN` env var also works; if both are missing, writing falls back to `omnivault set` (requires `omnivault unlock` first).

### 1) Materials → Markdown (offline RapidOCR)

Convert Word/PDF/images (incl. scanned PDFs) into uniform Markdown (docx tables become Markdown tables):

```sh
python tools/ocr2text/convert.py "your-materials-folder"            # outputs .md
python tools/ocr2text/convert.py "your-materials-folder" --txt       # outputs plain .txt instead
```

Output is written to `<folder>/_output/`, preserving the directory structure. Supported: `.docx`, `.pdf` (auto-distinguishes text-layer vs. scanned+OCR), `.jpg/.png/.bmp/.webp/.tif`, `.txt/.md/.csv`.

### 2) Field semantic normalization (local qwen3:8b)

Read the converted Markdown and call Ollama's OpenAI-compatible endpoint to unify synonyms such as "出生日期 / 生日 / 出生年月日" into a standard key (e.g. `birth_date`), producing structured JSON:

```sh
python tools/ocr2text/normalize.py "your-materials-folder/_output" "result.json"
```

Or run the first two steps in one command:

```sh
python tools/ocr2text/run_workflow.py "your-materials-folder" --no-fill
```

Defaults to `http://localhost:11434/v1` + `qwen3:8b`; override with `LLM_BASE_URL` / `LLM_MODEL`.

### 3) Take over Edge auto-fill (data straight from OmniVault)

This connects to your **real running Microsoft Edge** and auto-fills web forms from **OmniVault's decrypted fields**, matched against the page's field names.

**Step 1 — Start Edge on a debug port.** Double-click `tools/ocr2text/edge_start.bat`; it launches a dedicated Edge instance and keeps the browser open so you can navigate to the target form:

```bat
tools/ocr2text\edge_start.bat        :: starts with --remote-debugging-port=9222
```

> Default Edge path is `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`; edit `edge_start.bat` if yours differs.

> **Why can't we just take over your already-open Edge?** The fill script talks to Edge over CDP (Chrome DevTools Protocol), and the CDP debug port must be opened with `--remote-debugging-port` **at the moment Edge starts** — it is not a switch you can inject afterward. The Edge you open with a normal double-click has no such port and cannot be attached to. `edge_start.bat` relaunches a debug-port instance using the **same user-data directory**, so your login state and open tabs are preserved — the takeover feels seamless.

**Step 2 — Enable the local server and get a token.** In the OmniVault UI's "AI Access" dialog, turn on **Enable local server** and copy the service token.

**Step 3 — Take over and fill.** `edge_fill.py` reads all vault fields over HTTP via the token, attaches to the active tab of the debug-port Edge, and fills each field by matching the page's label:

```sh
python tools/ocr2text/edge_fill.py --token "your-service-token"
```

Useful options:

| Option | Purpose |
|--------|---------|
| `--token <TOKEN>` | Service token for HTTP field reads (recommended; no interactive unlock) |
| `--fill-unknown` | Extra fallback matching page `name` / `placeholder` |
| `--mapping map.txt` | Manual mapping, one `field=page-label-keyword` per line |
| `--addr <URL>` | Vault server address (default `http://127.0.0.1:7200`) |

**English→Chinese label matching is built in**: vault English ids (e.g. `p1_identity.date_of_birth`) auto-map to Chinese page labels (e.g. "出生日期"). Unmatched fields are listed; use `--mapping` to specify them manually.

### 3.1 Verification log (2026-08-16, real run)

A test form with 8 fields was auto-filled from the vault's 159 read fields; 7/8 were filled successfully with data identical to the vault:

```text
Read 159 field values from OmniVault.
Fill results (matched & filled):
  p1_identity.name -> 姓名 = 徐小明
  p1_identity.phone -> 手机号 = 13900001111
  p1_identity.email -> 邮箱 = xuxiaoming@example.com
  p1_identity.date_of_birth -> 出生日期 = 2002-10-26
  p1_addresses.home_address -> 家庭住址 = 安徽省阜阳市颍东区阜阳五中纬一路南博万有限公司
  p1_identity.native_place -> 籍贯 = 安徽
  p1_identity.id_number -> 身份证号 = 440101199501011234
Verification: 7/8 fields filled successfully.
```

- `company` was left empty because the vault has no "公司" value for that person — fields are only filled when a value exists, confirming correct semantic matching.
- To run a regression pass against a sample form: `python tools/ocr2text/verify_fill.py` (spawns an isolated Edge → reads fields → fills → compares before/after).

---

## Shopping Demo

A self-contained demo showing two MCP servers working together: the vault provides personal context, a mock shop handles orders. One sentence in, order confirmation out — no extra questions.

```sh
cd examples/shopping-demo && npm install
claude mcp add shop -- npx tsx examples/shopping-demo/src/index.ts
```

> Test the shopping demo — order a T-shirt using data from my vault.

The agent reads your name, email, and address from the vault, browses the shop, and places the order. If something is missing (e.g. a T-shirt size), it asks you and suggests storing it in the vault for next time. Full setup: [examples/shopping-demo/README.md](./examples/shopping-demo/README.md).

---

## Project Layout

```
.
├── cmd/omnivault/            # CLI entry point (main.go dispatches all subcommands)
│   ├── main.go               # subcommand routing + help
│   ├── cmd_*.go              # subcommand implementations (init/unlock/lock/serve/set/get/list/delete/
│   │                         #   set-sensitivity/export/audit/merge/onboard/backup/restore/
│   │                         #   rollback/security-question/change-password/tokens/ui/status/schema)
│   ├── child_windows.go      # Windows daemon start/stop
│   ├── pipe_windows.go       # named pipes (Windows IPC)
│   ├── dpi_windows.go        # Windows high-DPI & foreground window
│   └── rsrc_windows_amd64.syso  # Windows icon/version resource
├── internal/
│   ├── api/                  # HTTP server, handlers, bridge, middleware
│   │   ├── ui/onboarding.html # the dossier frontend (single-page app, embedded in the binary)
│   │   ├── ui.go             # go:embed the page + /ui handler
│   │   ├── handlers.go       # fields/context/tokens/audit handlers
│   │   ├── handlers_attachments.go  # attachment handlers
│   │   ├── handlers_merge.go # /vault/merge/plan and /vault/merge/apply
│   │   ├── server.go         # HTTP server assembly
│   │   ├── bridge.go         # WebView2 native bridge (single-process UI)
│   │   └── middleware.go     # Bearer token auth
│   ├── crypto/               # KDF (Argon2id), AES-256-GCM, HKDF subkeys
│   ├── dpapi/                # Windows DPAPI key protection (Credential Manager)
│   ├── merge/                # merge: classify (three-tier), map (label → field)
│   ├── store/                # SQLite CRUD (fields, attachments, tokens, audit, meta)
│   └── vault/                # business logic (init/unlock/lock, encryption, session, backups, schema)
├── mcp/                      # TypeScript MCP server (src + dist)
├── examples/shopping-demo/   # shopping demo
├── docs/                     # user documentation (usage.md)
├── assets/                   # icons & assets
├── build/                    # build config (winres.json: Windows icon/version/manifest)
├── tools/ocr2text/           # helpers: materials → text → normalize → drive Edge autofill
├── .github/workflows/        # release.yml (push tag → goreleaser)
├── .goreleaser.yml           # multi-platform release config (darwin/linux × amd64/arm64)
├── build.ps1                 # Windows one-click build (icon resources + exe → repo root)
├── 首次使用.ps1              # one-click launcher (Chinese): first run + launch + auto-build
├── FirstRun.ps1              # one-click launcher (English)
├── Makefile                  # build / test / install / clean
├── install.sh                # macOS/Linux one-line install script
├── vercel.json               # one-line install redirect /install.sh
├── go.mod / go.sum           # Go dependencies
└── README.md                 # Chinese documentation (this is README_EN.md, the English mirror)
```

### Runtime Data Directory `~/.omnivault/`

```
~/.omnivault/
├── vault.db          # SQLite (encrypted fields, attachments, audit log, tokens)
├── secret.key        # 128-bit key (mode 0600)
├── .session          # session token (created on unlock)
├── omnivault.pid     # PID of the running server
├── backups/          # versioned encrypted snapshots (keeps 3)
└── ui-data/          # WebView2 profile + page file + localStorage (UI persistence)
```

---

## Build & Development

**Stack**: Go 1.26+ (pure Go, no CGO), `modernc.org/sqlite`, `golang.org/x/crypto` (argon2/hkdf), stdlib `net/http`, `jchv/go-webview2`.

```sh
# Windows
./build.ps1                     # generate icon resources → build OmniVault.exe at the repo root
./build.ps1 -Tests              # build + run all tests
./build.ps1 -SkipIcon           # skip resource embedding, bare exe only

# macOS / Linux
make build                      # build to bin/omnivault
make test                       # go test -v -race ./...
make install                    # install to /usr/local/bin
make clean                      # clean build artifacts

# Per-layer tests
go test -race ./internal/crypto/   # crypto layer
go test -race ./internal/store/    # store layer
go test -race ./internal/vault/    # vault core
go test -race ./internal/api/      # HTTP API
```

> After editing `internal/api/ui/onboarding.html`, rebuild for the change to take effect (the page is also rewritten to `ui-data/` on every launch, overwriting older versions).

---

## Release

- **CI**: `.github/workflows/release.yml` — on push of a `v*` tag, it runs `go test -race ./...`, then [goreleaser](https://goreleaser.com) builds and uploads the multi-platform artifacts (darwin/linux × amd64/arm64) and generates `checksums.txt`.
- **Config**: `.goreleaser.yml` (archives LICENSE and README).
- **Release flow**: you can also do it manually:
  1. Confirm you are on `main` with a clean working tree and everything pushed.
  2. Check the latest tag and the changes since it: `git log <last-tag>..HEAD --oneline`.
  3. Choose a version (patch/minor/major) and confirm with the user.
  4. Run `go test -race ./...` until it passes.
  5. Tag and push: `git tag -a v0.x.y -m "Release v0.x.y" && git push origin v0.x.y`.
  6. Verify the goreleaser build on the GitHub Actions page; artifacts appear on the Releases page.

---

## Testing

```sh
make test    # race detector enabled (go test -v -race ./...)
```

Tests use temp directories (`t.TempDir()`); no cleanup needed. All tests run with `-race`.

---

## FAQ

- **A new person shows someone else's data?** It won't. Person ID assignment scans the existing `pN_` prefixes in the vault, so a new dossier is always empty and never collides with old data; orphan dossiers (data in the vault but local settings lost) are auto-restored as tabs.
- **The person list is gone after restarting?** Make sure you're on the new version (which loads the UI via `file://`). The old version used `SetHtml`, where `localStorage` is unavailable and the list resets on restart; the new version writes the page to `~/.omnivault/ui-data/onboarding.html` and navigates via `file://`, so `localStorage` truly persists.
- **An external agent can't connect after enabling the local server?** Confirm the toggle is on (listening on `127.0.0.1:7200`) and that a valid `VAULT_TOKEN` (service token) is set, or `omnivault unlock` has produced a session.
- **Forgot the vault password?** If a recovery security question was set, answer it to reset; otherwise both key and password are required and cannot be recovered (that's the cost of the zero-knowledge design).
- **`vault: server not running`**: run `omnivault unlock` first, or enable "Local server" in the UI.

---

## Protocol

This project is a **reference implementation** of the **[Personal Context Protocol](https://github.com/54wu/personal-context-protocol)** — an open protocol for AI agents to access personal context. See the [protocol specification](https://github.com/54wu/personal-context-protocol/blob/main/specification.md).

---

## Credits

This project is developed on top of **[Personal Vault](https://github.com/54wu/personal-vault)** and extends it with a rebrand (OmniVault · 万象档案袋), a rebuilt single-process WebView2 native UI, and additional features (versioned automatic backups & rollback, attachments, service tokens, material merge, multi-person dossiers, etc.). Our sincere thanks to the original project and its author.

Development was also assisted by **Trae Work** (AI coding environment) — UI prototyping, debugging and documentation. Our thanks.

---

## Contributing

All kinds of contributions are welcome — bug fixes, docs, tests, features. Before you start, please read [CONTRIBUTING.md](./CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md).

**Quick start**

1. Fork this repository and branch off `main` as `feat/xxx` or `fix/xxx`.
2. You'll need **Go 1.26+**. After cloning, `go build ./...` should compile.
3. Keep code `gofmt`-clean, add tests for new logic, and make sure `make test` (`go test -race ./...`) is green.
4. Use semantic commit messages (`feat:` / `fix:` / `docs:` / `test:` …) and open a Pull Request against `main`.

**Issues & discussion**

- Questions, ideas & discussion: [Discussions](https://github.com/54wu/omnivault/discussions)
- Bug reports & feature requests: [Issues](https://github.com/54wu/omnivault/issues)
- If you've found a **security vulnerability, do not post it publicly** — use the private channel below (see [Security](#security)).

**Release cadence**

Releases are cut by maintainers: after merging to `main`, tag `vX.Y.Z` (see [Release](#release)); CI (goreleaser) automatically produces installers for every platform.

---

## Security

OmniVault is designed around **field-level encryption + zero-knowledge + the key never leaves your machine**:

- Per-field **AES-256-GCM**; the password is stretched with **Argon2id** and expanded into per-field subkeys via **HKDF**.
- The vault password is never written to disk; the key lives only in memory while unlocked.
- `secret.key` is stored separately from `vault.db` — keep a separate, encrypted backup.
- The local server listens on `127.0.0.1` by default; tokens are scoped and expire.

A full walkthrough of the crypto model lives in [Encryption (security model)](#encryption-security-model) and [Cross-device sync (cloud sync + key separation)](#cross-device-sync).

**Reporting a vulnerability**: please **do not expose details in public Issues/discussions**. Use GitHub's **private vulnerability reporting**:

[Private vulnerability reporting](https://github.com/54wu/omnivault/security/advisories)

We triage reports promptly, keep them confidential, and — with your consent — credit you in the acknowledgements once a fix ships. The full policy is in [SECURITY.md](./SECURITY.md).

---

## License

[MIT](./LICENSE)
