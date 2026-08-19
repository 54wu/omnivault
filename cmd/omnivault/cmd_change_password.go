package main

import (
	"fmt"

	"github.com/54wu/omnivault/internal/vault"
)

// cmdChangePassword changes the profile password. Every field is re-encrypted
// with a new vault key derived from the new password. In the single-process
// model the vault is unlocked directly with the old password, so it is only
// entered once.
func cmdChangePassword() {
	oldPw, err := promptPassword("旧密码: ")
	if err != nil {
		fatal("读取密码失败: %v", err)
	}
	newPw, err := promptPassword("新密码: ")
	if err != nil {
		fatal("读取密码失败: %v", err)
	}
	if len(newPw) < 8 {
		fatal("新密码至少需要 8 个字符")
	}
	confirm, err := promptPassword("再次输入新密码: ")
	if err != nil {
		fatal("读取密码失败: %v", err)
	}
	if newPw != confirm {
		fatal("两次输入的新密码不一致")
	}

	v, err := vault.Open(vaultDir())
	if err != nil {
		fatal("打开档案库失败: %v", err)
	}
	sk, err := readSecretKey()
	if err != nil {
		v.Close()
		fatal("%v", err)
	}
	if _, err := v.Unlock(oldPw, sk); err != nil {
		v.Close()
		fatal("解锁失败: %v", err)
	}
	defer func() {
		v.Lock()
		v.Close()
	}()

	if _, err := v.ChangePassword(oldPw, newPw); err != nil {
		fatal("修改密码失败: %v", err)
	}
	v.ScheduleBackup()
	fmt.Println("密码已修改，全部字段已用新密码重新加密。")
}