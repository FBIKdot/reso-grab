// reso-grab 是一个音频资源记录与同步工具。
//
// 程序在此完成依赖注入组装：存储、下载器注入 App，
// 无参数运行时进入 REPL，带参数时执行对应子命令。
package main

import (
	"fmt"
	"os"

	"reso-grab/internal/cmd"
	"reso-grab/internal/downloader"
	"reso-grab/internal/storage"
)

func main() {
	app := cmd.NewApp(
		storage.NewYAMLStore("db.yml"),
		downloader.NewHTTP(),
		"audio",
	)
	if err := app.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}
