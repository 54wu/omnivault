package main

import (
	"fmt"

	"github.com/54wu/omnivault/internal/vault"
)

func cmdStatus() {
	v, err := vault.Open(vaultDir())
	if err != nil {
		fmt.Println("万象档案袋已锁定（无法打开档案库）。")
		return
	}
	defer v.Close()

	status, err := v.Status()
	if err != nil {
		fmt.Println("万象档案袋已锁定（无法读取状态）。")
		return
	}

	if !status.Initialized {
		fmt.Println("万象档案袋尚未初始化，请先运行 'omnivault init'。")
		return
	}

	if status.Locked {
		fmt.Println("状态:  已锁定")
	} else {
		fmt.Println("状态:  已解锁")
	}
	fmt.Printf("字段数: %d\n", status.FieldCount)
	if len(status.Categories) > 0 {
		fmt.Println("门类:")
		for cat, count := range status.Categories {
			fmt.Printf("  %-20s %d 个字段\n", cat, count)
		}
	}
}