package main

import (
	"fmt"
	"os"
	"time"

	"github.com/54wu/omnivault/internal/vault"
)

// cmdRollback lists available versioned backups or restores one. Usage:
//
//	omnivault rollback                 # list backup snapshots
//	omnivault rollback <name>          # restore a snapshot by name/timestamp
func cmdRollback() {
	if portHasVault() {
		fatal("万象档案袋服务正在运行 — 请先运行 'omnivault lock' 再回档")
	}

	v, err := vault.Open(vaultDir())
	if err != nil {
		fatal("打开档案库失败: %v", err)
	}
	defer v.Close()

	backups, err := v.Backups()
	if err != nil {
		fatal("读取备份失败: %v", err)
	}

	if len(os.Args) < 3 {
		if len(backups) == 0 {
			fmt.Println("暂无备份。写入字段后会自动生成版本化备份（保留最近 3 份）。")
			return
		}
		printBackups(backups)
		return
	}

	name := os.Args[2]
	if !confirmRollback() {
		fmt.Println("已取消。")
		return
	}
	if err := v.Rollback(name); err != nil {
		fatal("回档失败: %v", err)
	}
	fmt.Printf("已从备份 %s 回档。运行 'omnivault unlock' 并使用原档案密码解锁。\n", name)
}

func printBackups(backups []vault.BackupInfo) {
	fmt.Println("可用备份（新→旧，保留最近 3 份）:")
	for i, b := range backups {
		size := float64(b.Size) / 1024
		fmt.Printf("  %d. %s  %s  (%.1f KB)\n", i+1, b.Name, b.Created.Format(time.RFC3339), size)
	}
	fmt.Println()
	fmt.Println("回档示例: omnivault rollback " + backups[0].Name)
}

func confirmRollback() bool {
	fmt.Print("回档将覆盖当前档案库，是否继续？(y/N): ")
	var line string
	fmt.Scanln(&line)
	return line == "y" || line == "Y"
}