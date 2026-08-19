# Usage

## Quick Start

```sh
make build && make install

omnivault onboard               # Create vault, unlock, populate common fields (interactive)
omnivault set identity.full_name "Cool Cucumber"
omnivault get identity.full_name
omnivault list
omnivault lock                  # Stop server, zero keys from memory
```

Or step by step:

```sh
omnivault init           # Create vault at ~/.omnivault/, sets profile password + secret key
omnivault unlock         # Start server at localhost:7200, prompts for password
```

## Secret Key: Locating & Backing Up

Your `secret.key` is a 128-bit file. You need **both** the profile password and this file to unlock the vault — losing the key means the vault data cannot be recovered. It is deliberately **not** stored in the vault/project folder.

**Where it lives.** By default the vault key is created at:

```
~/.omnivault/secret.key
```

but on Windows the first-run script remembers whichever path you choose. To see the current remembered path, run:

```sh
omnivault key where
```

**Locate it.** When you launch via `首次使用.ps1` / `FirstRun.ps1`, the script resolves the key path in this order: environment variable `OVAULT_KEY_PATH` → the remembered (DPAPI) path → a prompt where you type the path. If no path is remembered, the script asks you for it.

**Back it up now (recommended steps).**
1. Copy `secret.key` off your machine — e.g. a USB drive or a private encrypted folder — **separate from both the app and the data**.
2. Keep at least one backup in a different physical location, so a lost or broken device doesn't cost you the vault.
3. If you only copied the project to a new computer, remember the new folder will *not* contain `secret.key`; copy the key file alongside it.
4. After moving to a new machine, point the launcher at the backed-up key (the script will prompt you for its path), then run:

```sh
omnivault key remember <absolute-path-of-secret.key>
```

> Keep `secret.key` such that only you can read it. Do not put it inside a git repo, the project folder, or your documents library.

## Fields

Fields use dot notation: `category.field_name`.

```sh
omnivault set identity.full_name "Cool Cucumber"
omnivault set identity.date_of_birth "1995-06-15"
omnivault set addresses.home_city "San Francisco"
omnivault set employment.employer "Acme Corp"
omnivault set financial.filing_status "single"
omnivault set preferences.timezone "America/Los_Angeles"
omnivault get identity.full_name
omnivault list                      # All fields
omnivault list identity             # One category
omnivault delete identity.date_of_birth
omnivault export                    # All fields as JSON
```

You can use any category and field name. Run `omnivault schema` to see recommended field names and their default sensitivity tiers.

## Sensitivity Tiers

Each field has a sensitivity tier that controls how it's shared with consumers.

| Tier | Examples | Behavior |
|------|----------|----------|
| `public` | Name, timezone, language | Auto-shared with authorized consumers |
| `standard` | Address, employer, education | Shared on request, logged |
| `sensitive` | DOB, phone, tax status | Requires explicit approval |
| `critical` | SSN, card number, card expiry | Requires approval + verification |

```sh
omnivault set-sensitivity financial.ssn critical
omnivault set-sensitivity preferences.timezone public
```

Default is `standard` for new fields. The recommended schema provides sensible defaults — use `omnivault schema` to see them.

## Service Tokens

Service tokens let applications authenticate with the vault using long-lived credentials. They follow the 1Password service account pattern.

```sh
omnivault create-service-token myapp --scope "*" --ttl 8760h
omnivault create-service-token tax-agent --scope "identity.*,financial.*" --ttl 1h
omnivault list-service-tokens
omnivault revoke-service-token abc123    # Revoke by token prefix
```

Service tokens keep the vault alive. Each authenticated request resets the 30-minute auto-lock timer, so the vault stays unlocked as long as a consumer is active.

## HTTP API

The vault runs at `http://127.0.0.1:7200`. All protected endpoints require `Authorization: Bearer <token>`.

### Public

```
GET  /vault/status                       # { initialized, locked, field_count, categories }
GET  /vault/schema                       # Recommended field names and sensitivity tiers
POST /vault/unlock                       # { password, secret_key } → { token }
```

### Fields

```
GET    /vault/fields                     # List all field metadata (no values)
GET    /vault/fields/{id}                # Get field with decrypted value
PUT    /vault/fields/{id}                # { value, sensitivity? } — upsert
DELETE /vault/fields/{id}                # Delete field
GET    /vault/fields/category/{name}     # All fields in category with values
```

### Context

```
GET /vault/context                       # Full decrypted dump grouped by category
```

This is what consumers call. Returns:

```json
{
  "categories": {
    "identity": [
      { "id": "identity.full_name", "category": "identity", "field_name": "full_name", "value": "Cool Cucumber", "sensitivity": "standard" }
    ],
    "preferences": [...]
  }
}
```

### Sensitivity

```
PUT /vault/sensitivity/{id}              # { tier } — update sensitivity
```

### Service Tokens

```
POST   /vault/tokens/service             # { consumer, scope, ttl } → { token, expires_at }
GET    /vault/tokens/service             # List active tokens (values truncated)
DELETE /vault/tokens/service/{prefix}    # Revoke by prefix
```

### Session

```
POST /vault/lock                         # Lock vault, zero keys
```

### Audit

```
GET /vault/audit?limit=50                # Recent access log
```

## Security Model

```
Profile Password + Secret Key (128-bit)
  → Argon2id KDF (64MB, 3 iterations)
  → Vault Key (256-bit, in-memory only)
  → HKDF per category → Category Subkeys
  → AES-256-GCM per field (12-byte random nonce)
```

- Profile password is never stored
- Secret key lives at `~/.omnivault/secret.key` (mode 0600), never transmitted
- Vault key exists only in memory while unlocked, zeroed on lock
- Auto-lock after 30 minutes of inactivity
- Every access logged to `vault_access_log`

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `VAULT_DIR` | `~/.omnivault` | Vault directory |
| `VAULT_ADDR` | `http://127.0.0.1:7200` | Server address for CLI |
| `VAULT_PORT` | `7200` | Server listen port |

## File Layout

```
~/.omnivault/
├── vault.db       # SQLite database (encrypted fields)
├── secret.key     # 128-bit secret key (mode 0600)
├── .session       # Session token (created on unlock)
└── omnivault.pid     # PID of running server
```
