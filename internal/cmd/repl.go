package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// RunREPL 运行交互式命令循环。
// 命令与子命令模式一致：add [source] / add-loop [source] / sync / fmt / migrate / exit。
func (a *App) RunREPL() error {
	fmt.Printf("reso-grab v%s\n", Version)
	fmt.Println("命令: add [source] / add-loop [source] / sync / fmt / migrate / exit")
	for {
		line, err := prompt("> ")
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				return nil
			}
			return err
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		var cmdErr error
		switch fields[0] {
		case "exit", "quit":
			fmt.Println("再见")
			return nil
		case "add":
			cmdErr = a.runAdd(fields[1:], false)
		case "add-loop":
			cmdErr = a.runAdd(fields[1:], true)
		case "sync":
			cmdErr = a.sync(context.Background())
		case "fmt", "format":
			cmdErr = a.format()
		case "migrate":
			oldPath := "db-old.yml"
			if len(fields) > 1 {
				oldPath = fields[1]
			}
			cmdErr = a.migrate(oldPath)
		default:
			fmt.Printf("未知命令: %s\n", fields[0])
		}
		if cmdErr != nil {
			fmt.Println("错误:", cmdErr)
		}
		fmt.Println()
	}
}

// runAdd 处理 REPL 中的 add / add-loop 命令。
// loop 为 true 时循环录入并锁定来源。
func (a *App) runAdd(args []string, loop bool) error {
	src, err := sourceArg(args)
	if err != nil {
		return err
	}
	if loop {
		return a.addLoop(src)
	}
	return a.addOne(src)
}
