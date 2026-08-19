# Vault MCP Server

MCP server that exposes your encrypted personal vault to AI agents.

## Install

```sh
npm install omnivault-mcp
```

Or run directly with npx:

```sh
npx -y omnivault-mcp@latest
```

## Register with Claude Code

```sh
claude mcp add vault -- npx -y omnivault-mcp@latest
```

## Tools

| Tool | Description |
|------|-------------|
| `vault_status` | Check if the vault is running and unlocked |
| `vault_get` | Get a single decrypted field by ID |
| `vault_list` | List all field metadata (no values) |
| `vault_context` | Get all decrypted fields grouped by category |
| `vault_set` | Set an encrypted field value in the vault |
| `vault_merge_plan` | Three-tier merge dry-run: classify a material source against a person's dossier into a decision plan (no writes) |
| `vault_merge_apply` | Apply a merge decision plan (only fill/replace/add actions; auto backup + audit) |

## Merge workflow

`vault_merge_plan` + `vault_merge_apply` implement a three-tier, human-in-the-loop
merge for adding external material (e.g. a filled application form) into a person's
dossier:

1. **Ownership** — the material's `person_hint` is cross-checked against existing
   persons (name + id_number + phone + email weighting). When ambiguous, the plan
   returns candidates for the user to pick (`person` param forces a target).
2. **Classify** — each material item is mapped to a vault field and tagged:
   - `auto` → identical value (no-op) or safe fill of a blank low-sensitivity field
   - `batch` → conflict on a low-sensitivity field (needs user decision)
   - `manual` → sensitive/critical field or list-merge (needs user decision)
3. **Decide & apply** — the user sets `action` (keep/replace/fill/add/skip) on the
   batch/manual items, then `vault_merge_apply` writes only those actions. Writes
   trigger an automatic `BackupNow()` snapshot and are audited.

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

## Authentication

The MCP server authenticates with the vault automatically:

1. `VAULT_TOKEN` env var — use with service tokens for always-on agents
2. `~/.omnivault/.session` — session token from `omnivault unlock`

Token is resolved on each request, so the server survives vault lock/unlock cycles.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VAULT_ADDR` | `http://127.0.0.1:7200` | Vault server address |
| `VAULT_DIR` | `~/.omnivault` | Vault directory (for session file) |
| `VAULT_TOKEN` | — | Service token (overrides session file) |

## Error Messages

| Error | Meaning |
|-------|---------|
| `vault: server not running` | Run `omnivault unlock` to start the server |
| `vault: session expired` | Run `omnivault unlock` to refresh the session |
| `vault: vault is locked` | Run `omnivault unlock` to decrypt |
| `vault: not found` | Field ID doesn't exist |
| `vault: not configured` | No token available — run `omnivault unlock` or set `VAULT_TOKEN` |

## Demo

See [examples/shopping-demo](../examples/shopping-demo/) for a demo using Vault MCP + a mock shop.
