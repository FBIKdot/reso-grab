package cmd

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reso-grab/internal/model"
	"reso-grab/internal/storage"
)

// memStore 是内存版 Store，用于隔离文件系统。
type memStore struct {
	db   *model.DB
	path string
}

func newMemStore() *memStore {
	return &memStore{db: &model.DB{}, path: "mem://db.yml"}
}

func (m *memStore) Load() (*model.DB, error) { return m.db, nil }
func (m *memStore) Save(db *model.DB) error  { m.db = db; return nil }
func (m *memStore) Path() string             { return m.path }

// recDownloader 记录下载调用，可预设错误。
type recDownloader struct {
	calls []string // 记录 url
	err   error    // 非空时所有下载返回该错误
}

func (r *recDownloader) Download(ctx context.Context, url, dest string) error {
	r.calls = append(r.calls, url)
	return r.err
}

// withStdin 临时替换录入流程的输入源。
func withStdin(t *testing.T, input string) {
	t.Helper()
	old := stdin
	stdin = bufio.NewReader(strings.NewReader(input))
	t.Cleanup(func() { stdin = old })
}

func TestSyncSkipsNonSyncableAndEmptyLinks(t *testing.T) {
	store := newMemStore()
	store.db = &model.DB{Resources: []model.AudioResource{
		// dova 不可同步，应跳过。
		{Source: model.Dova, Name: "D", Author: "A", URL: "https://dova-s.jp/EN/bgm/play1.html"},
		// 无下载链接，应跳过。
		{Source: model.Standalone, Name: "NoLink", Author: "A", URL: "https://x.com"},
		// 正常下载。
		{Source: model.Pixabay, Name: "P", Author: "A", DownloadLink: "https://cdn.pixabay.com/1.mp3"},
		{Source: model.Incompetech, Name: "I", Author: "Kevin MacLeod", DownloadLink: "https://incompetech.com/1.mp3"},
	}}
	dl := &recDownloader{}
	app := NewApp(store, dl, t.TempDir())

	if err := app.sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(dl.calls) != 2 {
		t.Fatalf("下载次数 = %d, 期望 2: %v", len(dl.calls), dl.calls)
	}
	// 下载是并发的，只校验集合不校验顺序。
	got := map[string]bool{}
	for _, u := range dl.calls {
		got[u] = true
	}
	if !got["https://cdn.pixabay.com/1.mp3"] || !got["https://incompetech.com/1.mp3"] {
		t.Errorf("下载对象异常: %v", dl.calls)
	}
}

func TestSyncCreatesSourceDirs(t *testing.T) {
	store := newMemStore()
	store.db = &model.DB{Resources: []model.AudioResource{
		{Source: model.Pixabay, Name: "P", Author: "A", DownloadLink: "https://cdn.pixabay.com/1.mp3"},
	}}
	audioDir := filepath.Join(t.TempDir(), "audio")
	app := NewApp(store, &recDownloader{}, audioDir)

	if err := app.sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, src := range model.SourceOrder {
		info := model.Sources[src]
		if !info.Syncable {
			continue
		}
		dir := filepath.Join(audioDir, info.Dir)
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			t.Errorf("可同步来源目录应被创建: %s", dir)
		}
	}
}

func TestSyncNoTasks(t *testing.T) {
	store := newMemStore() // 空库
	app := NewApp(store, &recDownloader{}, t.TempDir())
	if err := app.sync(context.Background()); err != nil {
		t.Fatalf("空库同步不应报错: %v", err)
	}
}

func TestSyncCountsFailures(t *testing.T) {
	store := newMemStore()
	store.db = &model.DB{Resources: []model.AudioResource{
		{Source: model.Pixabay, Name: "P", Author: "A", DownloadLink: "https://cdn.pixabay.com/1.mp3"},
	}}
	app := NewApp(store, &recDownloader{err: errors.New("boom")}, t.TempDir())

	err := app.sync(context.Background())
	if err == nil || !strings.Contains(err.Error(), "1 个文件下载失败") {
		t.Errorf("应汇总失败数, 得到: %v", err)
	}
}

func TestAddOneGenericSource(t *testing.T) {
	store := newMemStore()
	app := NewApp(store, &recDownloader{}, t.TempDir())
	withStdin(t, "My Song\nAlice\nhttps://x.com/page\nhttps://x.com/file.mp3\n")

	if err := app.addOne(model.Pixabay); err != nil {
		t.Fatal(err)
	}
	if len(store.db.Resources) != 1 {
		t.Fatalf("应有 1 条记录, 得到 %d", len(store.db.Resources))
	}
	r := store.db.Resources[0]
	if r.Source != model.Pixabay || r.Name != "My Song" || r.Author != "Alice" ||
		r.URL != "https://x.com/page" || r.DownloadLink != "https://x.com/file.mp3" {
		t.Errorf("记录异常: %+v", r)
	}
}

func TestAddOneIncompetechSkipsAuthor(t *testing.T) {
	store := newMemStore()
	app := NewApp(store, &recDownloader{}, t.TempDir())
	// incompetech 不问作者：输入依次为名称、页面链接、下载链接。
	withStdin(t, "Hitman\nhttps://incompetech.com/p\nhttps://incompetech.com/f.mp3\n")

	if err := app.addOne(model.Incompetech); err != nil {
		t.Fatal(err)
	}
	r := store.db.Resources[0]
	if r.Author != "Kevin MacLeod" {
		t.Errorf("作者应为固定值 Kevin MacLeod: %+v", r)
	}
}

func TestAddOneDovaFlow(t *testing.T) {
	store := newMemStore()
	app := NewApp(store, &recDownloader{}, t.TempDir())
	withStdin(t, "Song X composed by Author Y\n12345\n2\ny,n\n")

	if err := app.addOne(model.Dova); err != nil {
		t.Fatal(err)
	}
	r := store.db.Resources[0]
	if r.Source != model.Dova || r.Name != "Song X" || r.Author != "Author Y" {
		t.Errorf("基本字段异常: %+v", r)
	}
	if r.URL != "https://dova-s.jp/EN/bgm/play12345.html" {
		t.Errorf("URL 应由 id 构造: %q", r.URL)
	}
	if r.Tracks != 2 || len(r.Loop) != 2 || !r.Loop[0] || r.Loop[1] {
		t.Errorf("tracks/loop 异常: %+v", r)
	}
}

func TestAddOneDovaBadFormat(t *testing.T) {
	app := NewApp(newMemStore(), &recDownloader{}, t.TempDir())
	// 不含 " composed by " 分隔符的输入应报格式错误。
	withStdin(t, "没有分隔符的名称\n")

	err := app.addOne(model.Dova)
	if err == nil || !strings.Contains(err.Error(), "composed by") {
		t.Errorf("格式错误应提示正确格式, 得到: %v", err)
	}
}

func TestAddOneDovaBadTracks(t *testing.T) {
	app := NewApp(newMemStore(), &recDownloader{}, t.TempDir())
	withStdin(t, "S composed by A\n1\nabc\n")

	if err := app.addOne(model.Dova); err == nil {
		t.Error("非法轨道数应报错")
	}
}

func TestAddOneCancelOnEmptyInput(t *testing.T) {
	store := newMemStore()
	app := NewApp(store, &recDownloader{}, t.TempDir())
	withStdin(t, "\n") // 首个必填项留空即取消

	if err := app.addOne(model.Pixabay); err != nil {
		t.Fatal(err)
	}
	if len(store.db.Resources) != 0 {
		t.Error("取消后不应写入记录")
	}
}

func TestAddLoopStopsOnEmpty(t *testing.T) {
	store := newMemStore()
	app := NewApp(store, &recDownloader{}, t.TempDir())
	// 第一条完整录入（含下载链接），第二轮名称留空结束循环。
	// 注意下载链接允许留空，因此必须用名称留空来触发取消。
	withStdin(t, "S1\nA\nhttps://x.com/1\nhttps://x.com/f.mp3\n\n")

	if err := app.addLoop(model.Standalone); err != nil {
		t.Fatal(err)
	}
	if len(store.db.Resources) != 1 {
		t.Errorf("应录入 1 条后结束, 得到 %d", len(store.db.Resources))
	}
}

// 规范化发生在 YAMLStore.Save 内，因此这里使用真实的文件 Store。
func TestFormatNormalizes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.yml")
	store := storage.NewYAMLStore(path)
	// 预写一条未规范化的 dova 记录（缺 loop）。
	db := &model.DB{Resources: []model.AudioResource{
		{Source: model.Dova, Name: "S", Author: "A", URL: "https://dova-s.jp/EN/bgm/play1.html", Tracks: 2},
	}}
	if err := store.Save(db); err != nil {
		t.Fatal(err)
	}

	app := NewApp(store, &recDownloader{}, t.TempDir())
	if err := app.format(); err != nil {
		t.Fatal(err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	r := got.Resources[0]
	if r.Tracks != 2 || len(r.Loop) != 2 {
		t.Errorf("format 应补全 loop: %+v", r)
	}
}

func TestAppMigrateMissingOldFile(t *testing.T) {
	app := NewApp(newMemStore(), &recDownloader{}, t.TempDir())
	if err := app.migrate(filepath.Join(t.TempDir(), "nope.yml")); err == nil {
		t.Error("旧文件不存在时应报错")
	}
}

func TestAppMigrateSuccess(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "db-old.yml")
	old := "pixabay:\n  - name: P\n    author: A\n    site: 'https://x.com'\n    download_link: 'https://x.com/f.mp3'\n"
	if err := os.WriteFile(oldPath, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	store := newMemStore()
	app := NewApp(store, &recDownloader{}, dir)
	if err := app.migrate(oldPath); err != nil {
		t.Fatal(err)
	}
	if len(store.db.Resources) != 1 || store.db.Resources[0].Name != "P" {
		t.Errorf("迁移结果异常: %+v", store.db.Resources)
	}
}

// 确保 memStore 满足 storage.Store 接口（编译期检查）。
var _ storage.Store = (*memStore)(nil)
