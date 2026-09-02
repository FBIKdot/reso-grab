// Package cmd 实现命令行交互层。
//
// cobra 命令树与 REPL 共用同一批 handler（add/sync/fmt/migrate），
// 保证两种入口的行为完全一致。
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"

	"reso-grab/internal/downloader"
	"reso-grab/internal/model"
	"reso-grab/internal/storage"
)

// Version 是程序版本号，构建时可用 -ldflags 覆盖。
var Version = "2.0.0"

// App 持有全部依赖，是 CLI 与 REPL 的公共入口。
type App struct {
	store      storage.Store
	downloader downloader.Downloader
	audioDir   string
}

// NewApp 通过依赖注入组装 App。
func NewApp(store storage.Store, dl downloader.Downloader, audioDir string) *App {
	return &App{store: store, downloader: dl, audioDir: audioDir}
}

// Execute 是程序入口：无参数时进入 REPL，否则执行 cobra 子命令。
func (a *App) Execute() error {
	if len(os.Args) <= 1 {
		return a.RunREPL()
	}
	return a.newRootCmd().Execute()
}

// newRootCmd 构建命令树。
func (a *App) newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "reso-grab",
		Short:         "音频资源记录与同步工具",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		// 无子命令时进入 REPL，与直接运行程序的行为一致。
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.RunREPL()
		},
	}

	addCmd := &cobra.Command{
		Use:   "add [source]",
		Short: "录入一条音频资源（默认 " + string(model.DefaultSource) + "）",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, err := sourceArg(args)
			if err != nil {
				return err
			}
			return a.addOne(src)
		},
	}
	addCmd.ValidArgs = model.SourceNames()

	addLoopCmd := &cobra.Command{
		Use:   "add-loop [source]",
		Short: "循环录入音频资源，指定来源后锁定该来源（默认 " + string(model.DefaultSource) + "）",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, err := sourceArg(args)
			if err != nil {
				return err
			}
			return a.addLoop(src)
		},
	}
	addLoopCmd.ValidArgs = model.SourceNames()

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "下载所有可同步来源的资源到本地",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.sync(cmd.Context())
		},
	}

	fmtCmd := &cobra.Command{
		Use:     "fmt",
		Aliases: []string{"format"},
		Short:   "规范化并重新保存数据库文件",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.format()
		},
	}

	migrateCmd := &cobra.Command{
		Use:   "migrate [旧数据文件]",
		Short: "将旧版数据库一次性迁移为新格式（默认读取 db-old.yml）",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldPath := "db-old.yml"
			if len(args) == 1 {
				oldPath = args[0]
			}
			return a.migrate(oldPath)
		},
	}

	root.AddCommand(addCmd, addLoopCmd, syncCmd, fmtCmd, migrateCmd)
	return root
}

// sourceArg 解析 add / add-loop 的来源参数，缺省时返回默认来源。
func sourceArg(args []string) (model.Source, error) {
	if len(args) == 0 {
		return model.DefaultSource, nil
	}
	return model.ParseSource(args[0])
}

// format 重新加载并保存数据库，触发字段规范化（如补全 loop 数组）。
func (a *App) format() error {
	db, err := a.store.Load()
	if err != nil {
		return err
	}
	if err := a.store.Save(db); err != nil {
		return err
	}
	fmt.Println("格式化完成")
	return nil
}

// migrate 执行旧数据一次性迁移。
func (a *App) migrate(oldPath string) error {
	if _, err := os.Stat(oldPath); err != nil {
		return fmt.Errorf("旧数据文件 %s 不存在", oldPath)
	}
	n, err := storage.MigrateFile(oldPath, a.store)
	if err != nil {
		return err
	}
	fmt.Printf("迁移完成：共 %d 条记录，已写入 %s\n", n, a.store.Path())
	return nil
}

// syncTask 是一个待下载任务。
type syncTask struct {
	url  string
	dest string
}

// sync 下载所有可同步来源的资源。
// 跳过两类条目：来源不支持同步（如 dova，仅记录），以及没有下载链接的条目。
// 各条目并发下载，单个失败不中断整体流程。
func (a *App) sync(ctx context.Context) error {
	db, err := a.store.Load()
	if err != nil {
		return err
	}

	var tasks []syncTask
	for _, r := range db.Resources {
		info, ok := model.Sources[r.Source]
		if !ok || !info.Syncable || r.DownloadLink == "" {
			continue
		}
		dest := filepath.Join(a.audioDir, info.Dir, r.FileName(1))
		tasks = append(tasks, syncTask{url: r.DownloadLink, dest: dest})
	}
	if len(tasks) == 0 {
		fmt.Println("没有可同步的资源")
		return nil
	}

	// 预创建各来源子目录。
	for _, src := range model.SourceOrder {
		if info, ok := model.Sources[src]; ok && info.Syncable {
			if err := os.MkdirAll(filepath.Join(a.audioDir, info.Dir), 0o755); err != nil {
				return err
			}
		}
	}

	fmt.Printf("开始下载 %d 个文件...\n", len(tasks))
	var wg sync.WaitGroup
	failed := 0
	var mu sync.Mutex
	for _, t := range tasks {
		wg.Add(1)
		go func(t syncTask) {
			defer wg.Done()
			err := a.downloader.Download(ctx, t.url, t.dest)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case errors.Is(err, downloader.ErrFileExists):
				fmt.Printf("已存在，跳过: %s\n", filepath.Base(t.dest))
			case err != nil:
				failed++
				fmt.Printf("失败: %s (%v)\n", t.url, err)
			default:
				fmt.Printf("完成: %s\n", filepath.Base(t.dest))
			}
		}(t)
	}
	wg.Wait()

	if failed > 0 {
		return fmt.Errorf("同步完成，%d 个文件下载失败", failed)
	}
	fmt.Println("同步完成！")
	return nil
}
