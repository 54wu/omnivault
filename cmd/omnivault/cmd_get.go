package main

import (
	"fmt"
	"os"

	"github.com/54wu/omnivault/internal/vault"
)

func cmdGet() {
	if len(os.Args) < 3 {
		fatal("usage: omnivault get <id>\n  example: omnivault get identity.full_name")
	}
	id := os.Args[2]

	runVault(func(v *vault.Vault) error {
		field, err := v.Get(id)
		if err != nil {
			return err
		}
		if field == nil {
			return fmt.Errorf("field not found: %s", id)
		}
		fmt.Println(field.Value)
		return nil
	})
}