# Shopping Demo

A shopping agent that uses your vault to buy a t-shirt — no questions asked.

Two MCP servers connected to one agent. The agent bridges your personal context with a mock shop. One sentence, zero questions.

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Shop MCP   │◄────│  Claude Code │────►│  Vault MCP   │
│  (products)  │     │  (agent)     │     │  (context)   │
└──────────────┘     └──────────────┘     └──────────────┘
```

## Prerequisites

- [OmniVault](../../) built and on your PATH
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed
- Node.js 18+

## Setup

### 1. Build and initialize the vault

```sh
cd ../..
make build && make install
omnivault init
```

Save the secret key somewhere safe.

### 2. Populate your vault

```sh
omnivault unlock
omnivault set identity.first_name "Cool"
omnivault set identity.last_name "Cucumber"
omnivault set identity.full_name "Cool Cucumber"
omnivault set identity.email "cool@example.com"
omnivault set addresses.home_street "123 Main St"
omnivault set addresses.home_city "Seattle"
omnivault set addresses.home_state "WA"
omnivault set addresses.home_zip "98101"
omnivault set addresses.home_country "US"
```

### 3. Install the MCP servers

```sh
# Vault MCP
cd ../../mcp
npm install

# Shop MCP
cd ../examples/shopping-demo
npm install
```

### 4. Connect both MCP servers to Claude Code

From the project root:

```sh
claude mcp add vault -- npx tsx mcp/src/index.ts
claude mcp add shop -- npx tsx examples/shopping-demo/src/index.ts
```

### 5. Run the demo

Open Claude Code in this directory and type:

> Test the shopping demo — order a t-shirt using my vault data.

The agent will:
1. Query your vault for identity and address
2. Browse the shop for products
3. Place an order with your details
4. Return an order confirmation

## What's happening

Without the vault, the agent asks you 6+ questions: name, email, street, city, state, zip. With the vault, it already knows. One sentence in, order confirmation out.

The vault MCP server reads your encrypted personal context and makes it available as structured data. The shop MCP server handles product listing and orders. The agent connects the two.

## Troubleshooting

**"vault: server not running"** — Run `omnivault unlock` to start the vault server.

**"vault: session expired"** — The vault auto-locks after 30 minutes of inactivity. Run `omnivault unlock` again.

**"vault: vault is locked"** — Same as above. Run `omnivault unlock`.

**Empty vault** — Run the `omnivault set` commands from step 2 above.
