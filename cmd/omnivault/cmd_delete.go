package main

import (
	"fmt"
	"os"

	"github.com/54wu/omnivault/internal/vault"
)

func cmdDelete() {
	if len(os.Args) < 3 {
		fatal("usage: omnivault delete <id>")
	}
	id := os.Args[2]

	runVault(func(v *vault.Vault) error {
		if err := v.Delete(id); err != nil {
			return err
		}
		v.ScheduleBackup()
		fmt.Printf("Deleted %s\n", id)
		return nil
	})
}