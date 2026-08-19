package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// cmdKey manages where the secret key (secret.key) lives. The key is meant to
// be stored OUTSIDE the vault folder at an arbitrary path; these subcommands
// locate, remember, forget, and relocate that path.
func cmdKey() {
	if len(os.Args) < 3 {
		fatal("用法: omnivault key <where|remember|forget|relocate>")
	}
	switch os.Args[2] {
	case "where":
		cmdKeyWhere()
	case "remember":
		cmdKeyRemember()
	case "forget":
		cmdKeyForget()
	case "relocate":
		cmdKeyRelocate()
	default:
		fatal("未知 key 子命令: %s", os.Args[2])
	}
}

// key where — print the resolved key path.
func cmdKeyWhere() {
	p, err := resolveKeyPath()
	if err != nil {
		fatal("%v", err)
	}
	fmt.Println(p)
}

// key remember <path> — persist the key path in the OS credential store.
func cmdKeyRemember() {
	if len(os.Args) < 4 {
		fatal("用法: omnivault key remember <路径>")
	}
	p := os.Args[3]
	abs, err := filepath.Abs(p)
	if err != nil {
		fatal("无法解析路径: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		fatal("路径不存在: %s", abs)
	}
	if err := rememberKeyPath(abs); err != nil {
		fatal("记住密钥路径失败: %v", err)
	}
	fmt.Printf("已记住密钥路径: %s\n", abs)
}

// key forget — clear the remembered key path.
func cmdKeyForget() {
	if err := forgetKeyPath(); err != nil {
		fatal("清除失败: %v", err)
	}
	fmt.Println("已清除记住的密钥路径。")
}

// key relocate --to <path> — move the legacy vaultDir/secret.key out to an
// external path, remember it, and delete the old copy (migration helper).
func cmdKeyRelocate() {
	to := ""
	for i := 1; i+1 < len(os.Args); i++ {
		if os.Args[i] == "--to" {
			to = os.Args[i+1]
		}
	}
	if to == "" {
		fatal("用法: omnivault key relocate --to <外部路径>")
	}
	abs, err := filepath.Abs(to)
	if err != nil {
		fatal("无法解析目标路径: %v", err)
	}
	legacy := secretKeyPath()
	if _, err := os.Stat(legacy); err != nil {
		fatal("%s 不存在，无需迁移", legacy)
	}
	if err := writeSecretKey(abs, string(mustReadFile(legacy))); err != nil {
		fatal("迁移密钥失败: %v", err)
	}
	if err := rememberKeyPath(abs); err != nil {
		fmt.Fprintf(os.Stderr, "警告: 记住路径失败: %v\n", err)
	}
	if err := os.Remove(legacy); err != nil {
		fmt.Fprintf(os.Stderr, "警告: 删除旧密钥失败（请手动删除 %s）: %v\n", legacy, err)
	} else {
		fmt.Printf("已迁移密钥到: %s（旧副本已删除，路径已记住）\n", abs)
	}
	fmt.Println("建议将密钥再备份到另一物理位置（USB/密码管理器）。")
}

func mustReadFile(p string) []byte {
	data, err := os.ReadFile(p)
	if err != nil {
		fatal("%v", err)
	}
	return data
}