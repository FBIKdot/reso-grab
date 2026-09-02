package model

import (
	"strings"
	"testing"
)

func TestParseSource(t *testing.T) {
	tests := []struct {
		in      string
		want    Source
		wantErr bool
	}{
		{"dova", Dova, false},
		{"PIXABAY", Pixabay, false},
		{"  Incompetech  ", Incompetech, false},
		{"freemusicarchive", FreemusicArchive, false},
		{"standalone", Standalone, false},
		{"bogus", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		got, err := ParseSource(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseSource(%q) 期望错误，得到 %v", tt.in, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("ParseSource(%q) = %v, %v; 期望 %v", tt.in, got, err, tt.want)
		}
	}
}

// 非法来源的错误信息应列出可用来源，方便用户自助纠正。
func TestParseSourceErrorListsAvailable(t *testing.T) {
	_, err := ParseSource("bogus")
	if err == nil {
		t.Fatal("期望错误")
	}
	for _, name := range SourceNames() {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("错误信息应包含来源 %q: %v", name, err)
		}
	}
}

func TestSourceNames(t *testing.T) {
	names := SourceNames()
	if len(names) != len(SourceOrder) {
		t.Fatalf("SourceNames 长度 = %d, 期望 %d", len(names), len(SourceOrder))
	}
	for i, src := range SourceOrder {
		if names[i] != string(src) {
			t.Errorf("SourceNames()[%d] = %q, 期望 %q", i, names[i], src)
		}
	}
}

func TestFileName(t *testing.T) {
	tests := []struct {
		name   string
		res    AudioResource
		track  int
		expect string
	}{
		{"单轨带作者", AudioResource{Name: "Song", Author: "Alice"}, 1, "Song - Alice.mp3"},
		{"第二轨后缀", AudioResource{Name: "Song", Author: "Alice"}, 2, "Song - Alice_2.mp3"},
		{"第三轨后缀", AudioResource{Name: "Song", Author: "Alice"}, 3, "Song - Alice_3.mp3"},
		{"无作者省略分隔符", AudioResource{Name: "Song"}, 1, "Song.mp3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.res.FileName(tt.track); got != tt.expect {
				t.Errorf("FileName(%d) = %q, 期望 %q", tt.track, got, tt.expect)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	t.Run("dova 补全 tracks 与 loop", func(t *testing.T) {
		r := AudioResource{Source: Dova}
		r.Normalize()
		if r.Tracks != 1 || len(r.Loop) != 1 || r.Loop[0] != false {
			t.Errorf("规范化结果异常: tracks=%d loop=%v", r.Tracks, r.Loop)
		}
	})
	t.Run("dova loop 不足时补 false", func(t *testing.T) {
		r := AudioResource{Source: Dova, Tracks: 3, Loop: []bool{true}}
		r.Normalize()
		want := []bool{true, false, false}
		if len(r.Loop) != len(want) {
			t.Fatalf("loop = %v, 期望 %v", r.Loop, want)
		}
		for i := range want {
			if r.Loop[i] != want[i] {
				t.Errorf("loop[%d] = %v, 期望 %v", i, r.Loop[i], want[i])
			}
		}
	})
	t.Run("非 dova 来源不做处理", func(t *testing.T) {
		r := AudioResource{Source: Pixabay}
		r.Normalize()
		if r.Tracks != 0 || r.Loop != nil {
			t.Errorf("非 dova 来源不应被修改: tracks=%d loop=%v", r.Tracks, r.Loop)
		}
	})
}

func TestDBAdd(t *testing.T) {
	t.Run("正常添加并规范化", func(t *testing.T) {
		db := &DB{}
		err := db.Add(AudioResource{Source: Dova, Name: "S", Author: "A", URL: "https://dova-s.jp/EN/bgm/play1.html"})
		if err != nil {
			t.Fatal(err)
		}
		r := db.Resources[0]
		if r.Tracks != 1 || len(r.Loop) != 1 {
			t.Errorf("添加时应规范化: tracks=%d loop=%v", r.Tracks, r.Loop)
		}
	})

	t.Run("同 URL 判重", func(t *testing.T) {
		db := &DB{}
		_ = db.Add(AudioResource{Source: Pixabay, Name: "S", Author: "A", URL: "https://x.com/1"})
		err := db.Add(AudioResource{Source: Pixabay, Name: "S", Author: "A", URL: "https://x.com/1"})
		if err == nil {
			t.Error("相同 URL 应判重")
		}
	})

	t.Run("同名不同 URL 不误杀（dova 同名不同 id 场景）", func(t *testing.T) {
		db := &DB{}
		_ = db.Add(AudioResource{Source: Dova, Name: "Darklight", Author: "A", URL: "https://dova-s.jp/EN/bgm/play22967.html"})
		err := db.Add(AudioResource{Source: Dova, Name: "Darklight", Author: "A", URL: "https://dova-s.jp/EN/bgm/play22995.html"})
		if err != nil {
			t.Errorf("不同 URL 不应判重: %v", err)
		}
	})

	t.Run("无 URL 时按名称加作者判重", func(t *testing.T) {
		db := &DB{}
		_ = db.Add(AudioResource{Source: Standalone, Name: "S", Author: "A"})
		if err := db.Add(AudioResource{Source: Standalone, Name: "S", Author: "A"}); err == nil {
			t.Error("相同名称与作者应判重")
		}
		if err := db.Add(AudioResource{Source: Standalone, Name: "S", Author: "B"}); err != nil {
			t.Errorf("作者不同不应判重: %v", err)
		}
	})

	t.Run("跨来源不互相判重", func(t *testing.T) {
		db := &DB{}
		_ = db.Add(AudioResource{Source: Pixabay, Name: "S", Author: "A", URL: "https://x.com/1"})
		if err := db.Add(AudioResource{Source: Standalone, Name: "S", Author: "A", URL: "https://x.com/1"}); err != nil {
			t.Errorf("不同来源不应互相判重: %v", err)
		}
	})
}
