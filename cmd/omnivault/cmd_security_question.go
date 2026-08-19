package main

import (
	"fmt"

	"github.com/54wu/omnivault/internal/vault"
)

// cmdSetSecurityQuestion sets (or updates) the recovery security question.
// After 5 consecutive wrong passwords the vault demands this answer to continue
// unlocking. In the single-process model the command unlocks the vault itself.
func cmdSetSecurityQuestion() {
	fmt.Print("密保问题（例如：我的第一所学校叫什么？）: ")
	var q string
	if _, err := fmt.Scanln(&q); err != nil || q == "" {
		fatal("问题不能为空")
	}

	a, err := promptPassword("密保答案: ")
	if err != nil {
		fatal("读取答案失败: %v", err)
	}
	confirm, err := promptPassword("再次输入答案: ")
	if err != nil {
		fatal("读取答案失败: %v", err)
	}
	if a != confirm {
		fatal("两次输入的答案不一致")
	}

	runVault(func(v *vault.Vault) error {
		if err := v.SetSecurityQuestion(q, a); err != nil {
			return err
		}
		v.ScheduleBackup()
		return nil
	})
	fmt.Println("密保问题已设置。连续输错 5 次密码后，需回答该问题才能继续解锁。")
}