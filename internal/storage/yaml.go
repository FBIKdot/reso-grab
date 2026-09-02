// Package storage 提供数据库的持久化实现。
package storage

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"reso-grab/internal/model"
)

// Store 定义数据库持久化接口，便于测试时替换为内存实现。
type Store interface {
	// Load 读取数据库；文件不存在时返回空数据库。
	Load() (*model.DB, error)
	// Save 规范化并写入数据库。
	Save(db *model.DB) error
	// Path 返回底层数据文件路径。
	Path() string
}

// fileLayout 是数据库文件的存储布局：顶层键为来源分类，值为该来源的条目列表。
// 字段声明顺序决定分类在文件中的书写顺序。
type fileLayout struct {
	Dova             []model.AudioResource `yaml:"dova,omitempty"`
	Pixabay          []model.AudioResource `yaml:"pixabay,omitempty"`
	Incompetech      []model.AudioResource `yaml:"incompetech,omitempty"`
	FreemusicArchive []model.AudioResource `yaml:"freemusicarchive,omitempty"`
	Standalone       []model.AudioResource `yaml:"standalone,omitempty"`
}

// YAMLStore 是基于单个 YAML 文件的 Store 实现。
type YAMLStore struct {
	path string
}

// NewYAMLStore 创建以 path 为数据文件的 YAMLStore。
func NewYAMLStore(path string) *YAMLStore {
	return &YAMLStore{path: path}
}

// Path 返回数据文件路径。
func (s *YAMLStore) Path() string { return s.path }

// Load 读取并解析数据文件，按分类键还原每条记录的来源。
func (s *YAMLStore) Load() (*model.DB, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return &model.DB{}, nil
	}
	if err != nil {
		return nil, err
	}

	var layout fileLayout
	if err := yaml.Unmarshal(data, &layout); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", s.path, err)
	}

	db := &model.DB{}
	appendAll := func(src model.Source, list []model.AudioResource) {
		for _, r := range list {
			r.Source = src
			r.Normalize()
			db.Resources = append(db.Resources, r)
		}
	}
	appendAll(model.Dova, layout.Dova)
	appendAll(model.Pixabay, layout.Pixabay)
	appendAll(model.Incompetech, layout.Incompetech)
	appendAll(model.FreemusicArchive, layout.FreemusicArchive)
	appendAll(model.Standalone, layout.Standalone)
	return db, nil
}

// Save 将数据库按来源分类写回文件。
// yaml.v3 对不含空格的长标量（如 URL）不会折行，与旧版
// lineWidth: Infinity 的诉求一致。
func (s *YAMLStore) Save(db *model.DB) error {
	var layout fileLayout
	for _, r := range db.Resources {
		r.Normalize()
		switch r.Source {
		case model.Dova:
			layout.Dova = append(layout.Dova, r)
		case model.Pixabay:
			layout.Pixabay = append(layout.Pixabay, r)
		case model.Incompetech:
			layout.Incompetech = append(layout.Incompetech, r)
		case model.FreemusicArchive:
			layout.FreemusicArchive = append(layout.FreemusicArchive, r)
		case model.Standalone:
			layout.Standalone = append(layout.Standalone, r)
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(layout)
	if cerr := enc.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, buf.Bytes(), 0o644)
}
