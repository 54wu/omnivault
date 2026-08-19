package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/54wu/omnivault/internal/merge"
	"github.com/54wu/omnivault/internal/vault"
)

// POST /vault/merge/plan — dry-run: classify a material source into a plan.
func (s *Server) handleMergePlan(w http.ResponseWriter, r *http.Request) {
	var src merge.Source
	if err := json.NewDecoder(r.Body).Decode(&src); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "无效的 JSON")
		return
	}

	plan := merge.Plan{}

	// Determine target person.
	personID := r.URL.Query().Get("person")
	if personID == "" {
		persons, err := merge.DiscoverPersons(s.vault)
		if err != nil {
			handleVaultError(w, err)
			return
		}
		own := merge.VerifyOwnership(src.PersonHint, persons)
		if !own.Matched {
			writeJSON(w, http.StatusOK, map[string]any{
				"ownership": own,
				"candidates": persons,
				"note": "材料归属无法自动确认。请指定 person 参数，或先新建档案。",
			})
			return
		}
		personID = own.PersonID
		plan.Ownership = own
	} else {
		plan.Ownership = merge.Ownership{PersonID: personID, Matched: true}
	}

	// Scope guard: plan only touches fields the token allows.
	scope := scopeFromRequest(r)
	unmatched := []string{}
	for _, it := range src.Items {
		item, err := merge.ClassifyItem(s.vault, personID, it.Label, it.Value)
		if err != nil {
			handleVaultError(w, err)
			return
		}
		if item == nil {
			unmatched = append(unmatched, it.Label)
			continue
		}
		if !vault.ScopeAllows(scope, item.FieldID) {
			unmatched = append(unmatched, it.Label)
			continue
		}
		plan.Items = append(plan.Items, *item)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"plan":      plan,
		"unmatched": unmatched,
	})
}

// POST /vault/merge/apply — apply a decision plan.
func (s *Server) handleMergeApply(w http.ResponseWriter, r *http.Request) {
	var plan merge.Plan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "无效的 JSON")
		return
	}

	scope := scopeFromRequest(r)
	applied, kept, skipped := 0, 0, 0
	for _, it := range plan.Items {
		if !vault.ScopeAllows(scope, it.FieldID) {
			kept++
			continue
		}
		switch it.Action {
		case "fill", "replace", "add":
			if strings.TrimSpace(it.Incoming) == "" {
				skipped++
				continue
			}
			if err := s.vault.Set(it.FieldID, it.Incoming, it.Sensitivity); err != nil {
				handleVaultError(w, err)
				return
			}
			applied++
		case "keep", "skip":
			skipped++
		default:
			kept++
		}
	}
	if applied > 0 {
		s.vault.BackupNow()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"applied": applied,
		"kept":    kept,
		"skipped": skipped,
	})
}