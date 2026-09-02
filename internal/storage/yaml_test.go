package storage

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"reso-grab/internal/model"
)

// sampleDB 构造覆盖全部五个来源的示例数据库。
func sampleDB() *model.DB {
	return &model.DB{Resources: []model.AudioResource{
		{Source: model.Dova, Name: "Dova Song", Author: "DA", URL: "https://dova-s.jp/EN/bgm/play1.html", Tracks: 2, Loop: []bool{true, false}},
		{Source: model.Pixabay, Name: "Pix Song", Author: "PA", URL: "https://pixabay.com/p", DownloadLink: "https://cdn.pixabay.com/d.mp3"},
		{Source: model.Incompetech, Name: "Inc Song", Author: "Kevin MacLeod", URL: "https://incompetech.com/i", DownloadLink: "https://incompetech.com/d.mp3"},
		{Source: model.FreemusicArchive, Name: "FMA Song", Author: "FA", URL: "https://freemusicarchive.org/f", DownloadLink: "https://files.freemusicarchive.org/d.mp3"},
		{Source: model.Standalone, Name: "Solo Song", Author: "SA", URL: "https://example.com/s"},
	}}
}

func TestYAMLStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.yml")
	store := NewYAMLStore(path)

	if err := store.Save(sampleDB()); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}

	want := sampleDB()
	if len(got.Resources) != len(want.Resources) {
		t.Fatalf("条目数 = %d, 期望 %d", len(got.Resources), len(want.Resources))
	}
	for i := range want.Resources {
		g, w := got.Resources[i], want.Resources[i]
		// AudioResource 含切片字段，不能用 == 比较。
		if !reflect.DeepEqual(g, w) {
			t.Errorf("条目 %d 不一致:\n得到 %+v\n期望 %+v", i, g, w)
		}
	}
}

func TestYAMLStoreLoadMissingFile(t *testing.T) {
	store := NewYAMLStore(filepath.Join(t.TempDir(), "not-exist.yml"))
	db, err := store.Load()
	if err != nil {
		t.Fatalf("文件不存在不应报错: %v", err)
	}
	if len(db.Resources) != 0 {
		t.Errorf("期望空数据库, 得到 %d 条", len(db.Resources))
	}
}

// 长 URL 不应被折行，与旧版 lineWidth: Infinity 的诉求一致。
func TestYAMLStoreNoLineWrap(t *testing.T) {
	longURL := "https://files.freemusicarchive.org/storage-freemusicarchive-org/tracks/" +
		strings.Repeat("x", 120) + ".mp3?download=1&name=very-long-name.mp3"
	path := filepath.Join(t.TempDir(), "db.yml")
	store := NewYAMLStore(path)

	db := &model.DB{Resources: []model.AudioResource{
		{Source: model.Pixabay, Name: "S", Author: "A", URL: longURL},
	}}
	if err := store.Save(db); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "url: "+longURL) {
		t.Error("长 URL 应完整出现在单行中")
	}
}

// 分类键应按 SourceOrder 的规范顺序书写。
func TestYAMLStoreSectionOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.yml")
	store := NewYAMLStore(path)
	if err := store.Save(sampleDB()); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	last := -1
	for _, src := range model.SourceOrder {
		idx := strings.Index(text, string(src)+":")
		if idx == -1 {
			t.Fatalf("缺少分类键 %q", src)
		}
		if idx < last {
			t.Errorf("分类 %q 的顺序不正确", src)
		}
		last = idx
	}
}

// 空分类不应写入文件（omitempty）。
func TestYAMLStoreOmitEmptySections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.yml")
	store := NewYAMLStore(path)

	db := &model.DB{Resources: []model.AudioResource{
		{Source: model.Pixabay, Name: "S", Author: "A"},
	}}
	if err := store.Save(db); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "dova:") {
		t.Error("空分类 dova 不应出现在文件中")
	}
}
