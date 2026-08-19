package merge

import (
	"sort"
	"strings"

	"github.com/54wu/omnivault/internal/vault"
)

// Person is a discovered person in the vault (identified by its id prefix).
type Person struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IDNumber string `json:"id_number"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
}

// Signal is one identity signal compared between material and a person.
type Signal struct {
	Key   string `json:"key"`
	A     string `json:"a"`
	B     string `json:"b"`
	Match bool   `json:"match"`
}

// Ownership is the outcome of person-ownership verification.
type Ownership struct {
	PersonID string   `json:"person_id"`
	Matched  bool     `json:"matched"`
	Score    float64  `json:"score"`
	Signals  []Signal `json:"signals"`
}

// Item is a single field-level merge decision.
type Item struct {
	FieldID     string `json:"field_id"`
	Label       string `json:"label"`
	Current     string `json:"current"`
	Incoming    string `json:"incoming"`
	Sensitivity string `json:"sensitivity"`
	Tier        string `json:"tier"`        // auto | batch | manual
	Cardinality int    `json:"cardinality,omitempty"`
	Action      string `json:"action"`      // keep | replace | fill | add | skip
}

// Plan is the full decision plan produced by dry-run and consumed by apply.
type Plan struct {
	Ownership Ownership `json:"ownership"`
	Items     []Item    `json:"items"`
}

// Source is the material file format consumed by merge.
type Source struct {
	Source     string    `json:"source"`
	PersonHint Person    `json:"person_hint"`
	Items      []RawItem `json:"items"`
}

// RawItem is one labelled material entry.
type RawItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// DiscoverPersons scans the vault for existing person prefixes and the key
// identity fields of each, used for ownership verification.
func DiscoverPersons(v *vault.Vault) ([]Person, error) {
	ctx, err := v.GetContext()
	if err != nil {
		return nil, err
	}

	byID := map[string]*Person{}
	for catKey, fields := range ctx.Categories {
		prefix := personPrefix(catKey)
		if prefix == "" {
			continue
		}
		p := byID[prefix]
		if p == nil {
			p = &Person{ID: prefix}
			byID[prefix] = p
		}
		for _, f := range fields {
			switch f.FieldName {
			case "name":
				p.Name = f.Value
			case "id_number":
				p.IDNumber = f.Value
			case "phone":
				p.Phone = f.Value
			case "email":
				p.Email = f.Value
			}
		}
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	persons := make([]Person, 0, len(ids))
	for _, id := range ids {
		persons = append(persons, *byID[id])
	}
	return persons, nil
}

// personPrefix extracts the person id ("p1") from a full category key
// ("p1_identity"). Returns "" if the category has no person prefix.
func personPrefix(category string) string {
	first := strings.SplitN(category, "_", 2)[0]
	if first == "" {
		return ""
	}
	return first
}

// VerifyOwnership scores the material hint against each known person and picks
// the best match. Weights: name 0.25, id_number 0.35, phone 0.25, email 0.15.
func VerifyOwnership(hint Person, persons []Person) Ownership {
	best := Ownership{Signals: []Signal{}}
	for _, p := range persons {
		signals := []Signal{
			{Key: "name", A: p.Name, B: hint.Name, Match: strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(hint.Name)) && p.Name != ""},
			{Key: "id_number", A: p.IDNumber, B: hint.IDNumber, Match: p.IDNumber != "" && p.IDNumber == hint.IDNumber},
			{Key: "phone", A: p.Phone, B: hint.Phone, Match: p.Phone != "" && p.Phone == hint.Phone},
			{Key: "email", A: p.Email, B: hint.Email, Match: strings.EqualFold(strings.TrimSpace(p.Email), strings.TrimSpace(hint.Email)) && p.Email != ""},
		}
		score := 0.0
		for _, s := range signals {
			if !s.Match {
				continue
			}
			switch s.Key {
			case "name":
				score += 0.25
			case "id_number":
				score += 0.35
			case "phone":
				score += 0.25
			case "email":
				score += 0.15
			}
		}
		if score > best.Score {
			best = Ownership{PersonID: p.ID, Score: score, Signals: signals}
		}
	}
	strong := false
	nameOK := false
	for _, s := range best.Signals {
		if (s.Key == "id_number" || s.Key == "phone") && s.Match {
			strong = true
		}
		if s.Key == "name" && s.Match {
			nameOK = true
		}
	}
	best.Matched = best.Score >= 0.6 || (nameOK && strong)
	return best
}

// ClassifyItem turns one material entry into an Item, comparing against the
// current vault value and applying the three-tier policy.
func ClassifyItem(v *vault.Vault, personPrefix, label, value string) (*Item, error) {
	mf := ResolveField(label)
	if mf == nil {
		return nil, nil // no mapping -> skip
	}
	if strings.TrimSpace(value) == "" {
		return nil, nil // empty material value -> nothing to merge
	}

	fullID := personPrefix + "_" + mf.FieldID
	item := &Item{
		FieldID:     fullID,
		Label:       mf.Label,
		Incoming:    value,
		Sensitivity: mf.Sensitivity,
		Cardinality: mf.Cardinality,
	}

	cur, err := v.Get(fullID)
	if err != nil {
		return nil, err
	}
	var a string
	if cur != nil {
		a = cur.Value
	}
	item.Current = a

	weight := SensitivityWeight(mf.Sensitivity)
	switch {
	case a == value:
		item.Tier = "auto"
		item.Action = "skip"
	case a == "":
		if weight <= 1 {
			item.Tier = "auto"
			item.Action = "fill"
		} else {
			item.Tier = "manual"
			item.Action = ""
		}
	default:
		if weight <= 1 {
			item.Tier = "batch"
			item.Action = ""
		} else {
			item.Tier = "manual"
			item.Action = ""
		}
	}
	return item, nil
}