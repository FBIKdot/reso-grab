package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"reso-grab/internal/model"
)

// stdin 是录入流程的输入源，测试时可替换。
var stdin = bufio.NewReader(os.Stdin)

// prompt 打印提示并读取一行输入，返回去除首尾空白的结果。
func prompt(label string) (string, error) {
	fmt.Print(label)
	line, err := stdin.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// addOne 执行一次录入流程：按来源提问，落库保存。
// 录入过程中任意必填项为空输入即取消本次录入。
func (a *App) addOne(src model.Source) error {
	r, ok, err := a.promptResource(src)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("输入为空，已取消")
		return nil
	}
	if err := a.addResource(r); err != nil {
		return err
	}
	return nil
}

// addLoop 循环录入，直到用户输入空值为止。
func (a *App) addLoop(src model.Source) error {
	fmt.Printf("循环录入 [%s]，任意必填项留空即结束\n", src)
	for {
		r, ok, err := a.promptResource(src)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("输入为空，结束循环录入")
			return nil
		}
		if err := a.addResource(r); err != nil {
			fmt.Println(err)
		}
		fmt.Println()
	}
}

// addResource 落库并保存。
func (a *App) addResource(r model.AudioResource) error {
	db, err := a.store.Load()
	if err != nil {
		return err
	}
	if err := db.Add(r); err != nil {
		return err
	}
	if err := a.store.Save(db); err != nil {
		return err
	}
	fmt.Printf("已添加: %s - %s [%s]\n", r.Name, r.Author, r.Source)
	return nil
}

// promptResource 按来源执行对应的提问序列。
// 返回的 ok 为 false 表示用户输入空值、取消本次录入。
func (a *App) promptResource(src model.Source) (model.AudioResource, bool, error) {
	info := model.Sources[src]
	r := model.AudioResource{Source: src}

	// 作者：来源级固定时直接赋值，否则提问。
	if info.FixedAuthor != "" {
		r.Author = info.FixedAuthor
	}

	if src == model.Dova {
		return a.promptDova(r)
	}

	name, err := prompt("名称> ")
	if err != nil || name == "" {
		return r, false, err
	}
	r.Name = name

	if r.Author == "" {
		r.Author, err = prompt("作者> ")
		if err != nil || r.Author == "" {
			return r, false, err
		}
	}

	r.URL, err = prompt("页面链接> ")
	if err != nil || r.URL == "" {
		return r, false, err
	}

	// 下载链接允许留空：留空的条目仅记录、不参与同步。
	r.DownloadLink, err = prompt("下载链接（可留空）> ")
	if err != nil {
		return r, false, err
	}
	return r, true, nil
}

// promptDova 是 dova 特有的录入流程。
//
// dova 不提供可供程序化抓取的信息（违反其 terms of use），因此全部手动录入：
// 一行式输入 "名称 composed by 作者"，再输入 dova 的曲目 id 构造页面链接，
// 最后录入轨道数与逐轨循环标记。
func (a *App) promptDova(r model.AudioResource) (model.AudioResource, bool, error) {
	line, err := prompt("[名称] composed by [作者]> ")
	if err != nil || line == "" {
		return r, false, err
	}
	parts := strings.SplitN(line, " composed by ", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return r, false, fmt.Errorf("格式应为: [名称] composed by [作者]")
	}
	r.Name = strings.TrimSpace(parts[0])
	r.Author = strings.TrimSpace(parts[1])

	id, err := prompt("dova 曲目 id> ")
	if err != nil || id == "" {
		return r, false, err
	}
	r.URL = fmt.Sprintf("https://dova-s.jp/EN/bgm/play%s.html", id)

	tracksStr, err := prompt("轨道数（默认 1）> ")
	if err != nil {
		return r, false, err
	}
	r.Tracks = 1
	if tracksStr != "" {
		n, convErr := strconv.Atoi(tracksStr)
		if convErr != nil || n < 1 {
			return r, false, fmt.Errorf("轨道数应为正整数: %q", tracksStr)
		}
		r.Tracks = n
	}

	hint := `逐轨是否循环，"y" 表示循环，逗号分隔（如 y,n）`
	loopStr, err := prompt(hint + "> ")
	if err != nil {
		return r, false, err
	}
	r.Loop = make([]bool, 0, r.Tracks)
	if loopStr != "" {
		for _, s := range strings.Split(loopStr, ",") {
			r.Loop = append(r.Loop, strings.TrimSpace(s) == "y")
		}
	}
	r.Normalize()
	return r, true, nil
}
