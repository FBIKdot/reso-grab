// Package model 定义音频资源的核心数据模型。
//
// 所有来源共用 AudioResource 一种结构，来源间的差异通过 Source 与可选字段表达，
// 新增来源只需在 Sources 注册表中追加一条 SourceInfo。
package model

import (
	"fmt"
	"strings"
)

// Source 标识音频资源的来源。
type Source string

const (
	Dova             Source = "dova"
	Pixabay          Source = "pixabay"
	Incompetech      Source = "incompetech"
	FreemusicArchive Source = "freemusicarchive"
	Standalone       Source = "standalone"
)

// DefaultSource 是 add / add-loop 未指定来源时使用的默认来源。
const DefaultSource = Standalone

// SourceInfo 描述一个来源的行为特征。
type SourceInfo struct {
	// Dir 是该来源在 audio/ 下的子目录名。
	Dir string
	// Syncable 表示 sync 是否支持自动下载该来源的资源。
	// dova 的 terms of use 禁止程序化抓取素材，因此仅记录、不参与同步。
	Syncable bool
	// FixedAuthor 非空时表示该来源的作者固定（如 incompetech 全部为
	// Kevin MacLeod），录入时无需再询问作者。
	FixedAuthor string
}

// Sources 是全部来源的注册表。
var Sources = map[Source]SourceInfo{
	Dova:             {Dir: "dova", Syncable: false},
	Pixabay:          {Dir: "pixabay", Syncable: true},
	Incompetech:      {Dir: "incompetech", Syncable: true, FixedAuthor: "Kevin MacLeod"},
	FreemusicArchive: {Dir: "freemusicarchive", Syncable: true},
	Standalone:       {Dir: "standalone", Syncable: true},
}

// SourceOrder 定义来源在数据库文件中的规范顺序。
var SourceOrder = []Source{Dova, Pixabay, Incompetech, FreemusicArchive, Standalone}

// SourceNames 按规范顺序返回所有来源的名称。
func SourceNames() []string {
	names := make([]string, 0, len(SourceOrder))
	for _, s := range SourceOrder {
		names = append(names, string(s))
	}
	return names
}

// ParseSource 忽略大小写地解析来源名，非法名称返回错误并列出可用来源。
func ParseSource(s string) (Source, error) {
	src := Source(strings.ToLower(strings.TrimSpace(s)))
	if _, ok := Sources[src]; ok {
		return src, nil
	}
	return "", fmt.Errorf("未知来源 %q，可用来源: %s", s, strings.Join(SourceNames(), " / "))
}

// AudioResource 是一条统一的音频资源条目。
//
// 各来源字段约定：
//   - URL 为资源页面链接，直接存储（dova 录入时由 id 构造后存入）；
//   - DownloadLink 仅可同步来源需要，为空表示不自动下载；
//   - Tracks / Loop 目前是 dova 独有字段，其余来源保持零值。
type AudioResource struct {
	// Source 记录所属来源。不写入文件：数据库文件中条目按顶层分类键归类，
	// 读取时由分类键还原。
	Source Source `yaml:"-"`
	// Name 是曲目名。
	Name string `yaml:"name"`
	// Author 是作者。
	Author string `yaml:"author,omitempty"`
	// URL 是资源页面链接。
	URL string `yaml:"url,omitempty"`
	// DownloadLink 是下载直链。
	DownloadLink string `yaml:"download_link,omitempty"`
	// Tracks 是轨道数，dova 独有。
	Tracks int `yaml:"tracks,omitempty"`
	// Loop 逐轨标记是否循环，dova 独有。
	Loop []bool `yaml:"loop,omitempty"`
}

// FileName 返回指定轨道号（从 1 开始）的本地文件名。
// 统一约定为 "名称 - 作者.mp3"，第 2 轨起追加 "_2"、"_3" 后缀。
func (r *AudioResource) FileName(track int) string {
	base := r.Name
	if r.Author != "" {
		base += " - " + r.Author
	}
	if track > 1 {
		return fmt.Sprintf("%s_%d.mp3", base, track)
	}
	return base + ".mp3"
}

// Normalize 补全条目的默认值。
// dova 的 tracks 至少为 1，loop 数组不足 tracks 长度时补 false，
// 与旧版 save 时的规范化行为保持一致。
func (r *AudioResource) Normalize() {
	if r.Source != Dova {
		return
	}
	if r.Tracks < 1 {
		r.Tracks = 1
	}
	for len(r.Loop) < r.Tracks {
		r.Loop = append(r.Loop, false)
	}
}

// DB 是音频资源数据库整体。
type DB struct {
	Resources []AudioResource
}

// Add 规范化并追加一条条目，重复时返回错误。
// 判重规则：同一来源下，双方 URL 均非空时比较 URL（dova 存在同名不同 id
// 的曲目，不能仅按名称判重）；否则按名称 + 作者判重。
func (db *DB) Add(r AudioResource) error {
	r.Normalize()
	for _, e := range db.Resources {
		if e.Source != r.Source {
			continue
		}
		var dup bool
		if r.URL != "" && e.URL != "" {
			dup = e.URL == r.URL
		} else {
			dup = e.Name == r.Name && e.Author == r.Author
		}
		if dup {
			return fmt.Errorf("%q - %s 已存在于 %s", r.Name, r.Author, r.Source)
		}
	}
	db.Resources = append(db.Resources, r)
	return nil
}
