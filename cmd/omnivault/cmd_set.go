package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/54wu/omnivault/internal/vault"
)

func cmdSet() {
	if len(os.Args) < 4 {
		fatal("usage: omnivault set <id> <value>\n  example: omnivault set identity.full_name \"Cool Cucumber\"")
	}
	id := os.Args[2]
	value := strings.Join(os.Args[3:], " ")

	if !strings.Contains(id, ".") {
		fatal("field ID must be category.name (e.g., identity.full_name)")
	}

	runVault(func(v *vault.Vault) error {
		if err := v.Set(id, value, vault.DefaultSensitivity(id)); err != nil {
			return err
		}
		v.ScheduleBackup()
		fmt.Printf("Set %s\n", id)
		if suggestion := vault.SuggestCanonical(id); suggestion != nil {
			fmt.Fprintf(os.Stderr, "Hint: the recommended field is %s (%s)\n",
				suggestion.Canonical, suggestion.Description)
		}
		return nil
	})
}