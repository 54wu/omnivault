package main

import (
	"fmt"
	"os"

	"github.com/54wu/omnivault/internal/vault"
)

func cmdSetSensitivity() {
	if len(os.Args) < 4 {
		fatal("usage: omnivault set-sensitivity <id> <tier>\n  tiers: public, standard, sensitive, critical")
	}
	id := os.Args[2]
	tier := os.Args[3]

	valid := map[string]bool{"public": true, "standard": true, "sensitive": true, "critical": true}
	if !valid[tier] {
		fatal("invalid tier %q (must be public, standard, sensitive, or critical)", tier)
	}

	runVault(func(v *vault.Vault) error {
		if err := v.SetSensitivity(id, tier); err != nil {
			return err
		}
		v.ScheduleBackup()
		fmt.Printf("Set %s sensitivity to %s\n", id, tier)
		return nil
	})
}