package main

import (
	"fmt"

	"github.com/54wu/omnivault/internal/vault"
)

func cmdInit() {
	dir := vaultDir()

	pw, err := promptPassword("档案密码: ")
	if err != nil {
		fatal("读取密码失败: %v", err)
	}
	if len(pw) < 8 {
		fatal("密码至少需要 8 个字符")
	}

	confirm, err := promptPassword("确认密码: ")
	if err != nil {
		fatal("读取确认密码失败: %v", err)
	}
	if pw != confirm {
		fatal("两次输入的密码不一致")
	}

	sk, err := vault.Init(dir, pw)
	if err != nil {
		fatal("%v", err)
	}

	keyPath, err := saveSecretKey(sk)
	if err != nil {
		fatal("保存密钥失败: %v", err)
	}

	fmt.Println("万象档案袋初始化成功。")
	fmt.Println()
	fmt.Println("你的密钥（请妥善保存到安全的地方）：")
	fmt.Printf("  %s\n", sk)
	fmt.Println()
	fmt.Printf("密钥已保存到: %s\n", keyPath)
	fmt.Printf("万象档案袋数据库: %s/vault.db\n", dir)
	fmt.Println()
	fmt.Println("下一步：运行 'omnivault unlock' 开始使用万象档案袋。")
}
