package downloader

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadSuccess(t *testing.T) {
	const body = "hello audio bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.mp3")
	if err := NewHTTP().Download(context.Background(), srv.URL, dest); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != body {
		t.Errorf("文件内容 = %q, 期望 %q", data, body)
	}
	// 成功后不应残留 .part 临时文件。
	if _, err := os.Stat(dest + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Error(".part 临时文件应已被清理")
	}
}

func TestDownloadSkipsExisting(t *testing.T) {
	requested := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = true
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "exists.mp3")
	if err := os.WriteFile(dest, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := NewHTTP().Download(context.Background(), srv.URL, dest)
	if !errors.Is(err, ErrFileExists) {
		t.Fatalf("期望 ErrFileExists, 得到 %v", err)
	}
	if requested {
		t.Error("文件已存在时不应发起请求")
	}
}

func TestDownloadNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.mp3")
	err := NewHTTP().Download(context.Background(), srv.URL, dest)
	if err == nil {
		t.Fatal("非 200 状态应报错")
	}
	// 失败时不应留下目标文件或临时文件。
	if _, statErr := os.Stat(dest); !errors.Is(statErr, os.ErrNotExist) {
		t.Error("失败时不应留下目标文件")
	}
	if _, statErr := os.Stat(dest + ".part"); !errors.Is(statErr, os.ErrNotExist) {
		t.Error("失败时不应残留 .part 文件")
	}
}

// 服务端声明长度后提前断开，模拟下载中断。
func TestDownloadInterruptedCleansPartial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.Write([]byte("short"))
		// 直接关闭连接，使客户端读取时得到意外 EOF。
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Skip("响应不支持 Hijack")
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.mp3")
	err := NewHTTP().Download(context.Background(), srv.URL, dest)
	if err == nil {
		t.Fatal("下载中断应报错")
	}
	if _, statErr := os.Stat(dest); !errors.Is(statErr, os.ErrNotExist) {
		t.Error("中断时不应留下残缺目标文件")
	}
	if _, statErr := os.Stat(dest + ".part"); !errors.Is(statErr, os.ErrNotExist) {
		t.Error("中断时 .part 临时文件应被清理")
	}
}
