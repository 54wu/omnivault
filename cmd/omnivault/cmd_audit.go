package main

import (
	"fmt"

	"github.com/54wu/omnivault/internal/vault"
)

func cmdAudit() {
	runVault(func(v *vault.Vault) error {
		entries, err := v.AuditLog(20)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("No audit entries.")
			return nil
		}
		for _, e := range entries {
			purpose := ""
			if e.Purpose != "" {
				purpose = fmt.Sprintf(" (%s)", e.Purpose)
			}
			fmt.Printf("%-20s %-10s %-8s %s%s\n",
				e.CreatedAt.Format("2006-01-02 15:04:05"),
				e.Consumer, e.Action, e.Scope, purpose)
		}
		return nil
	})
}