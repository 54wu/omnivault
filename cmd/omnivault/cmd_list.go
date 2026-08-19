package main

import (
	"fmt"
	"os"

	"github.com/54wu/omnivault/internal/vault"
)

func cmdList() {
	category := ""
	if len(os.Args) >= 3 {
		category = os.Args[2]
	}

	runVault(func(v *vault.Vault) error {
		var fields []vault.FieldInfo
		var err error
		if category != "" {
			fields, err = v.GetByCategory(category)
		} else {
			fields, err = v.List()
		}
		if err != nil {
			return err
		}

		if len(fields) == 0 {
			fmt.Println("No fields found.")
			return nil
		}

		for _, f := range fields {
			sens := ""
			if f.Sensitivity != "" && f.Sensitivity != "standard" {
				sens = fmt.Sprintf(" [%s]", f.Sensitivity)
			}
			if f.Value != "" {
				fmt.Printf("%-35s %s%s\n", f.ID, f.Value, sens)
			} else {
				fmt.Printf("%-35s (v%d)%s\n", f.ID, f.Version, sens)
			}
		}
		return nil
	})
}