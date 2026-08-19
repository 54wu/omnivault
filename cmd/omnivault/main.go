package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		// Double-clicked (no command): run as an app — open the vault and UI
		// in a single process (no background daemon, no network).
		cmdApp()
		return
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "unlock":
		cmdUnlock()
	case "lock":
		cmdLock()
	case "serve":
		cmdServe()
	case "status":
		cmdStatus()
	case "schema":
		cmdSchema()
	case "set":
		cmdSet()
	case "merge":
		cmdMerge()
	case "get":
		cmdGet()
	case "list":
		cmdList()
	case "delete":
		cmdDelete()
	case "set-sensitivity":
		cmdSetSensitivity()
	case "export":
		cmdExport()
	case "audit":
		cmdAudit()
	case "create-service-token":
		cmdCreateServiceToken()
	case "list-service-tokens":
		cmdListServiceTokens()
	case "revoke-service-token":
		cmdRevokeServiceToken()
	case "onboard":
		cmdOnboard()
	case "backup":
		cmdBackup()
	case "restore":
		cmdRestore()
	case "rollback":
		cmdRollback()
	case "set-security-question":
		cmdSetSecurityQuestion()
	case "change-password":
		cmdChangePassword()
	case "key":
		cmdKey()
	case "ui":
		cmdUI()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`omnivault — Personal Context Protocol vault

Usage: omnivault <command> [args]

Commands:
  onboard                          Create vault, unlock, and populate common fields
  init                             Create a new vault
  ui                               Open the vault UI in a native window (single-process)
  status                           Show vault status
  schema                           Show recommended field names (--json for raw JSON)
  set <id> <value>                 Set a field (e.g., identity.full_name "Cool Cucumber")
  merge <material.json>            Merge a material file into a person's dossier (three-tier)
  get <id>                         Get a field value
  list [category]                  List fields
  delete <id>                      Delete a field
  set-sensitivity <id> <tier>      Set field sensitivity (public|standard|sensitive|critical)
  export                           Export all decrypted fields as JSON
  audit                            Show access audit log
  backup <dest>                    Copy encrypted vault.db to a sync folder (key stays local)
  restore <src>                    Copy a synced vault.db back into the vault
  rollback [name]                  List versioned backups, or restore one
  set-security-question            Set recovery security question (asked after 5 wrong passwords)
  change-password                  Change profile password (re-encrypts all fields)
  create-service-token <consumer>  Create a long-lived service token
  list-service-tokens              List active service tokens
  revoke-service-token <prefix>    Revoke a service token by prefix

Daemon (for MCP / HTTP consumers only):
  unlock                           Start the vault serving daemon in the background
  lock                             Stop the serving daemon
  serve                            Run the serving daemon in the foreground`)
}
