package storage

import (
	"os"
	"path/filepath"
	"testing"

	"reso-grab/internal/model"
)

// oldFixture 覆盖旧版数据库的全部四种异构形态，
// 且每个分类保留多条记录以验证顺序保持。
const oldFixture = `dova:
  Author One:
    '100':
      name: Song A
      url: 'https://dova-s.jp/EN/bgm/play100.html'
      loop:
        - true
      tracks: 1
    '200':
      name: Song B
      url: 'https://dova-s.jp/EN/bgm/play200.html'
      loop:
        - false
        - true
      tracks: 2
  Author Two:
    '300':
      name: Song C
      url: 'https://dova-s.jp/EN/bgm/play300.html'
      tracks: 1
pixabay:
  - name: Pix One
    author: PA
    site: 'https://pixabay.com/1'
    download_link: 'https://cdn.pixabay.com/1.mp3'
  - name: Pix Two
    author: PB
    site: 'https://pixabay.com/2'
    download_link: 'https://cdn.pixabay.com/2.mp3'
incompetech:
  - name: Inc One
    site: 'https://incompetech.com/1'
    download_link: 'https://incompetech.com/1.mp3'
freemusicarchive:
  - name: FMA One
    author: FA
    site: 'https://freemusicarchive.org/1'
    download_link: 'https://files.freemusicarchive.org/1.mp3'
`

// writeOldFixture 在临时目录写入旧格式数据文件并返回其路径。
func writeOldFixture(t *testing.T) string {
	t.Helper()
	oldPath := filepath.Join(t.TempDir(), "db-old.yml")
	if err := os.WriteFile(oldPath, []byte(oldFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	return oldPath
}

func TestMigrateFile(t *testing.T) {
	oldPath := writeOldFixture(t)
	store := NewYAMLStore(filepath.Join(t.TempDir(), "db.yml"))

	n, err := MigrateFile(oldPath, store)
	if err != nil {
		t.Fatal(err)
	}
	if n != 7 {
		t.Fatalf("迁移条目数 = %d, 期望 7", n)
	}

	db, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}

	// 按来源分组，便于逐类断言。
	bySource := map[model.Source][]model.AudioResource{}
	for _, r := range db.Resources {
		bySource[r.Source] = append(bySource[r.Source], r)
	}

	t.Run("dova 拍平且保留 tracks 与 loop", func(t *testing.T) {
		list := bySource[model.Dova]
		if len(list) != 3 {
			t.Fatalf("dova 条目数 = %d, 期望 3", len(list))
		}
		first := list[0]
		if first.Name != "Song A" || first.Author != "Author One" {
			t.Errorf("首条 = %+v", first)
		}
		if first.URL != "https://dova-s.jp/EN/bgm/play100.html" {
			t.Errorf("URL 未保留: %q", first.URL)
		}
		if first.Tracks != 1 || len(first.Loop) != 1 || !first.Loop[0] {
			t.Errorf("tracks/loop 未保留: %+v", first)
		}
		// 多轨条目。
		second := list[1]
		if second.Tracks != 2 || len(second.Loop) != 2 || second.Loop[0] || !second.Loop[1] {
			t.Errorf("多轨条目异常: %+v", second)
		}
		// loop 缺失时补全。
		third := list[2]
		if third.Tracks != 1 || len(third.Loop) != 1 || third.Loop[0] {
			t.Errorf("loop 应补全为 [false]: %+v", third)
		}
	})

	t.Run("pixabay site 映射为 url", func(t *testing.T) {
		list := bySource[model.Pixabay]
		if len(list) != 2 {
			t.Fatalf("pixabay 条目数 = %d, 期望 2", len(list))
		}
		if list[0].URL != "https://pixabay.com/1" || list[0].DownloadLink != "https://cdn.pixabay.com/1.mp3" {
			t.Errorf("字段映射异常: %+v", list[0])
		}
	})

	t.Run("incompetech 填充固定作者", func(t *testing.T) {
		list := bySource[model.Incompetech]
		if len(list) != 1 {
			t.Fatalf("incompetech 条目数 = %d, 期望 1", len(list))
		}
		if list[0].Author != "Kevin MacLeod" {
			t.Errorf("作者应为 Kevin MacLeod: %+v", list[0])
		}
	})

	t.Run("freemusicarchive 正常迁移", func(t *testing.T) {
		list := bySource[model.FreemusicArchive]
		if len(list) != 1 {
			t.Fatalf("freemusicarchive 条目数 = %d, 期望 1", len(list))
		}
		if list[0].Author != "FA" || list[0].URL != "https://freemusicarchive.org/1" {
			t.Errorf("字段异常: %+v", list[0])
		}
	})
}

// 迁移必须保留旧文件中的条目顺序（实现遍历 YAML 节点而非 map）。
func TestMigrateFilePreservesOrder(t *testing.T) {
	oldPath := writeOldFixture(t)
	store := NewYAMLStore(filepath.Join(t.TempDir(), "db.yml"))

	if _, err := MigrateFile(oldPath, store); err != nil {
		t.Fatal(err)
	}
	db, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"Song A", "Song B", "Song C", "Pix One", "Pix Two", "Inc One", "FMA One"}
	if len(db.Resources) != len(want) {
		t.Fatalf("条目数 = %d, 期望 %d", len(db.Resources), len(want))
	}
	for i, name := range want {
		if db.Resources[i].Name != name {
			t.Errorf("第 %d 条 = %q, 期望 %q", i, db.Resources[i].Name, name)
		}
	}
}

func TestMigrateFileRefusesOverwrite(t *testing.T) {
	oldPath := writeOldFixture(t)
	target := filepath.Join(t.TempDir(), "db.yml")
	if err := os.WriteFile(target, []byte("existing: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := MigrateFile(oldPath, NewYAMLStore(target)); err == nil {
		t.Error("目标文件已存在时应拒绝迁移")
	}
}

func TestMigrateFileMissingOldFile(t *testing.T) {
	store := NewYAMLStore(filepath.Join(t.TempDir(), "db.yml"))
	if _, err := MigrateFile(filepath.Join(t.TempDir(), "nope.yml"), store); err == nil {
		t.Error("旧文件不存在时应报错")
	}
}
