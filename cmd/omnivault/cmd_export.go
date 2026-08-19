package main

import (
	"encoding/json"
	"os"

	"github.com/54wu/omnivault/internal/vault"
)

func cmdExport() {
	runVault(func(v *vault.Vault) error {
		ctx, err := v.GetContext()
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(ctx); err != nil {
			return err
		}
		return nil
	})
}