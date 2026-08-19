package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/54wu/omnivault/internal/vault"
)

func cmdCreateServiceToken() {
	if len(os.Args) < 3 {
		fatal("usage: omnivault create-service-token <consumer> [--scope categories] [--ttl duration]")
	}

	consumer := os.Args[2]
	scope := "*"
	ttl := 8760 * time.Hour // 1 year

	for i := 3; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--scope":
			if i+1 < len(os.Args) {
				scope = os.Args[i+1]
				i++
			}
		case "--ttl":
			if i+1 < len(os.Args) {
				d, err := time.ParseDuration(os.Args[i+1])
				if err != nil {
					fatal("invalid ttl duration: %v", err)
				}
				ttl = d
				i++
			}
		}
	}

	runVault(func(v *vault.Vault) error {
		token, err := v.CreateServiceToken(consumer, scope, ttl)
		if err != nil {
			return err
		}
		v.ScheduleBackup()
		fmt.Printf("Service token created for %q\n", consumer)
		fmt.Printf("Token:   %s\n", token)
		fmt.Printf("Scope:   %s\n", scope)
		fmt.Printf("Expires: %s\n", time.Now().Add(ttl).UTC().Format(time.RFC3339))
		fmt.Println("\nSave this token — it cannot be displayed again.")
		return nil
	})
}

func cmdListServiceTokens() {
	runVault(func(v *vault.Vault) error {
		tokens, err := v.ListServiceTokens()
		if err != nil {
			return err
		}
		if len(tokens) == 0 {
			fmt.Println("No service tokens.")
			return nil
		}

		type tokenInfo struct {
			TokenPrefix string `json:"token_prefix"`
			Consumer    string `json:"consumer"`
			Scope       string `json:"scope"`
			ExpiresAt   string `json:"expires_at"`
			CreatedAt   string `json:"created_at"`
		}
		result := make([]tokenInfo, 0, len(tokens))
		for _, t := range tokens {
			hashPrefix := t.TokenStr
			if len(hashPrefix) > 8 {
				hashPrefix = hashPrefix[:8] + "..."
			}
			result = append(result, tokenInfo{
				TokenPrefix: hashPrefix,
				Consumer:    t.Consumer,
				Scope:       t.Scope,
				ExpiresAt:   t.ExpiresAt.UTC().Format(time.RFC3339),
				CreatedAt:   t.CreatedAt.UTC().Format(time.RFC3339),
			})
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	})
}

func cmdRevokeServiceToken() {
	if len(os.Args) < 3 {
		fatal("usage: omnivault revoke-service-token <prefix>")
	}
	token := os.Args[2]

	runVault(func(v *vault.Vault) error {
		n, err := v.RevokeServiceToken(token)
		if err != nil {
			return err
		}
		v.ScheduleBackup()
		fmt.Printf("Revoked %d token(s).\n", n)
		return nil
	})
}