// Package downloader 提供文件下载能力。
package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ErrFileExists 表示目标文件已存在、跳过下载。
var ErrFileExists = errors.New("文件已存在")

// Downloader 定义文件下载接口，测试时可替换为 mock 实现。
type Downloader interface {
	// Download 下载 url 并保存为 dest；dest 已存在时返回 ErrFileExists。
	Download(ctx context.Context, url, dest string) error
}

// HTTP 是通用的 HTTP 下载实现。
type HTTP struct {
	client *http.Client
}

// NewHTTP 创建默认 HTTP 下载器。
// 超时设为 10 分钟，兼顾大体积音频文件与异常连接的及时失败。
func NewHTTP() *HTTP {
	return &HTTP{client: &http.Client{Timeout: 10 * time.Minute}}
}

// userAgent 模拟浏览器 UA，避免被部分 CDN 拒绝。
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) reso-grab"

// Download 实现 Downloader 接口。
// 先写入 .part 临时文件再改名，下载中断时不会留下残缺的目标文件。
func (h *HTTP) Download(ctx context.Context, url, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return ErrFileExists
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("意外的 HTTP 状态 %s", resp.Status)
	}

	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}
