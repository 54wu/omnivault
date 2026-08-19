package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/54wu/omnivault/internal/merge"
	"github.com/54wu/omnivault/internal/vault"
)

func cmdMerge() {
	materialPath, personFlag, planFile, applyFile := "", "", "", ""
	auto := false

	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--person":
			if i+1 < len(os.Args) {
				personFlag = os.Args[i+1]
				i++
			}
		case "--plan":
			if i+1 < len(os.Args) {
				planFile = os.Args[i+1]
				i++
			}
		case "--apply":
			if i+1 < len(os.Args) {
				applyFile = os.Args[i+1]
				i++
			}
		case "--auto":
			auto = true
		case "--dry-run":
			// default behaviour
		default:
			materialPath = arg
		}
	}

	// Apply mode: read a decision plan and write the accepted actions.
	if applyFile != "" {
		runVault(func(v *vault.Vault) error {
			plan, err := readMergePlan(applyFile)
			if err != nil {
				return err
			}
			applied, kept, skipped := applyMergeActions(v, plan.Items, false)
			fmt.Printf("已应用 %d 项，保留 %d 项，跳过 %d 项。\n", applied, kept, skipped)
			return nil
		})
		return
	}

	if materialPath == "" {
		fatal("usage: omnivault merge <material.json> [--person <id>] [--auto] [--plan <out.json>] | [--apply <plan.json>]")
	}

	src, err := readMergeSource(materialPath)
	if err != nil {
		fatal("读取材料失败: %v", err)
	}

	runVault(func(v *vault.Vault) error {
		plan := merge.Plan{}

		// Determine target person.
		personID := personFlag
		if personID == "" {
			persons, err := merge.DiscoverPersons(v)
			if err != nil {
				return err
			}
			own := merge.VerifyOwnership(src.PersonHint, persons)
			if !own.Matched {
				plan.Ownership = own
				out := map[string]any{
					"ownership": own,
					"candidates": persons,
					"note": "材料归属无法自动确认。请用 --person <id> 指定归属人，或先新建档案。",
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(out)
				return nil
			}
			personID = own.PersonID
			plan.Ownership = own
		} else {
			plan.Ownership = merge.Ownership{PersonID: personID, Matched: true}
		}

		// Classify every material item.
		unmatched := []string{}
		for _, it := range src.Items {
			item, err := merge.ClassifyItem(v, personID, it.Label, it.Value)
			if err != nil {
				return err
			}
			if item == nil {
				unmatched = append(unmatched, it.Label)
				continue
			}
			plan.Items = append(plan.Items, *item)
		}

		if len(unmatched) > 0 {
			fmt.Fprintf(os.Stderr, "提示：以下条目未映射到档案字段，已忽略：%s\n", strings.Join(unmatched, "、"))
		}

		if planFile != "" {
			if err := writeMergePlan(planFile, plan); err != nil {
				return err
			}
			fmt.Printf("决策计划已写入 %s\n", planFile)
		}

		if auto {
			applied, kept, skipped := applyMergeActions(v, plan.Items, true)
			fmt.Printf("自动采纳 %d 项，保留待决 %d 项，跳过 %d 项。\n", applied, kept, skipped)

			remaining := merge.Plan{Ownership: plan.Ownership}
			for _, it := range plan.Items {
				if it.Tier == "auto" {
					continue
				}
				remaining.Items = append(remaining.Items, it)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(remaining)
			return nil
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(plan)
		return nil
	})
}

// applyMergeActions writes fields whose action is a write (fill/replace/add).
// When onlyAuto is true, only auto-tier items are considered. Returns counts.
func applyMergeActions(v *vault.Vault, items []merge.Item, onlyAuto bool) (applied, kept, skipped int) {
	for _, it := range items {
		if onlyAuto && it.Tier != "auto" {
			kept++
			continue
		}
		switch it.Action {
		case "fill", "replace", "add":
			if strings.TrimSpace(it.Incoming) == "" {
				skipped++
				continue
			}
			if err := v.Set(it.FieldID, it.Incoming, it.Sensitivity); err != nil {
				fmt.Fprintf(os.Stderr, "写入失败 %s: %v\n", it.FieldID, err)
				continue
			}
			applied++
		case "keep", "skip":
			skipped++
		default:
			kept++
		}
	}
	if applied > 0 {
		v.BackupNow()
	}
	return
}

func readMergeSource(path string) (*merge.Source, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var src merge.Source
	if err := json.Unmarshal(data, &src); err != nil {
		return nil, err
	}
	if src.PersonHint.Name == "" && src.PersonHint.IDNumber == "" && src.PersonHint.Phone == "" && len(src.Items) == 0 {
		return nil, fmt.Errorf("材料文件为空或格式不正确")
	}
	return &src, nil
}

func readMergePlan(path string) (*merge.Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var plan merge.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func writeMergePlan(path string, plan merge.Plan) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}