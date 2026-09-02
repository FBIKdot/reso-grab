package storage

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"reso-grab/internal/model"
)

// 旧版数据库的条目结构（仅迁移时使用）。
type (
	// oldDovaEntry 是旧版 dova 嵌套映射 author -> id -> 条目 中的条目。
	oldDovaEntry struct {
		Name   string `yaml:"name"`
		URL    string `yaml:"url"`
		Tracks int    `yaml:"tracks"`
		Loop   []bool `yaml:"loop"`
	}
	// oldDefaultEntry 是旧版 pixabay / freemusicarchive 的条目。
	oldDefaultEntry struct {
		Name         string `yaml:"name"`
		Author       string `yaml:"author"`
		Site         string `yaml:"site"`
		DownloadLink string `yaml:"download_link"`
	}
	// oldIncompetechEntry 是旧版 incompetech 的条目，作者固定故无 author 字段。
	oldIncompetechEntry struct {
		Name         string `yaml:"name"`
		Site         string `yaml:"site"`
		DownloadLink string `yaml:"download_link"`
	}
)

// MigrateFile 将 oldPath 处的旧版数据库转换为新格式并通过 store 写入。
// 目标文件已存在时返回错误，避免覆盖现有数据；迁移是一次性的。
//
// 旧格式有四种异构形态：
//   - dova: author -> id -> 条目 的嵌套映射；
//   - pixabay / freemusicarchive: 含 author 与 download_link 的条目列表；
//   - incompetech: 无 author 的条目列表（作者固定为 Kevin MacLeod）。
//
// 为保留旧文件中的条目顺序，直接遍历 YAML 节点而不是反序列化为 map
// （Go 的 map 遍历顺序是随机的）。
func MigrateFile(oldPath string, store Store) (int, error) {
	if fileExists(store.Path()) {
		return 0, fmt.Errorf("%s 已存在，拒绝覆盖", store.Path())
	}
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return 0, fmt.Errorf("读取旧数据失败: %w", err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return 0, fmt.Errorf("解析 %s 失败: %w", oldPath, err)
	}
	db := &model.DB{}
	if len(root.Content) > 0 {
		top := root.Content[0] // 文档根映射节点
		for i := 0; i+1 < len(top.Content); i += 2 {
			key, val := top.Content[i].Value, top.Content[i+1]
			switch key {
			case string(model.Dova):
				migrateDova(val, db)
			case string(model.Pixabay):
				migratePixabayOrFMA(val, db, model.Pixabay)
			case string(model.FreemusicArchive):
				migratePixabayOrFMA(val, db, model.FreemusicArchive)
			case string(model.Incompetech):
				migrateIncompetech(val, db)
			}
		}
	}

	for i := range db.Resources {
		db.Resources[i].Normalize()
	}
	if err := store.Save(db); err != nil {
		return 0, err
	}
	return len(db.Resources), nil
}

// migrateDova 把 author -> id -> 条目 的嵌套映射拍平为统一条目，
// URL 直接沿用旧数据中已存储的页面链接。
func migrateDova(n *yaml.Node, db *model.DB) {
	for i := 0; i+1 < len(n.Content); i += 2 {
		author, ids := n.Content[i].Value, n.Content[i+1]
		for j := 0; j+1 < len(ids.Content); j += 2 {
			var e oldDovaEntry
			if err := ids.Content[j+1].Decode(&e); err != nil {
				continue
			}
			db.Resources = append(db.Resources, model.AudioResource{
				Source: model.Dova,
				Name:   e.Name,
				Author: author,
				URL:    e.URL,
				Tracks: e.Tracks,
				Loop:   e.Loop,
			})
		}
	}
}

// migratePixabayOrFMA 迁移结构相同的 pixabay / freemusicarchive 条目列表，
// 旧字段 site 对应新字段 url。
func migratePixabayOrFMA(n *yaml.Node, db *model.DB, src model.Source) {
	for _, item := range n.Content {
		var e oldDefaultEntry
		if err := item.Decode(&e); err != nil {
			continue
		}
		db.Resources = append(db.Resources, model.AudioResource{
			Source:       src,
			Name:         e.Name,
			Author:       e.Author,
			URL:          e.Site,
			DownloadLink: e.DownloadLink,
		})
	}
}

// migrateIncompetech 迁移 incompetech 条目列表，作者取来源级固定值。
func migrateIncompetech(n *yaml.Node, db *model.DB) {
	for _, item := range n.Content {
		var e oldIncompetechEntry
		if err := item.Decode(&e); err != nil {
			continue
		}
		db.Resources = append(db.Resources, model.AudioResource{
			Source:       model.Incompetech,
			Name:         e.Name,
			Author:       model.Sources[model.Incompetech].FixedAuthor,
			URL:          e.Site,
			DownloadLink: e.DownloadLink,
		})
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
