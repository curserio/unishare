package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/curserio/unishare/internal/model"
)

type FileStore struct {
	mu       sync.Mutex
	dataDir  string
	maxBytes int64
}

func NewFileStore(dataDir string, maxBytes int64) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Join(dataDir, "users"), 0o755); err != nil {
		return nil, err
	}
	store := &FileStore{
		dataDir:  dataDir,
		maxBytes: maxBytes,
	}
	return store, nil
}

func (s *FileStore) List(userID string) ([]model.Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.loadLocked(userID)
	if err != nil {
		return nil, err
	}
	items = append([]model.Item(nil), items...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (s *FileStore) Add(userID, title, text, rawURL string, uploads []*multipart.FileHeader) (model.Item, error) {
	item := model.Item{
		ID:        randomID(),
		Title:     strings.TrimSpace(title),
		Text:      strings.TrimSpace(text),
		URL:       strings.TrimSpace(rawURL),
		CreatedAt: time.Now().UTC(),
	}
	if item.URL != "" {
		parsed, err := url.ParseRequestURI(item.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			item.Text = strings.TrimSpace(strings.Join([]string{item.Text, item.URL}, "\n"))
			item.URL = ""
		}
	}

	itemDir := filepath.Join(s.userDir(userID), "files", item.ID)
	if len(uploads) > 0 {
		if err := os.MkdirAll(itemDir, 0o755); err != nil {
			return model.Item{}, err
		}
	}

	for _, header := range uploads {
		if header == nil || header.Filename == "" {
			continue
		}
		if header.Size > s.maxBytes {
			return model.Item{}, fmt.Errorf("%s is larger than max upload size", header.Filename)
		}
		file, err := header.Open()
		if err != nil {
			return model.Item{}, err
		}
		defer file.Close()

		fileID := randomID()
		target, err := os.OpenFile(filepath.Join(itemDir, fileID), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return model.Item{}, err
		}
		size, err := io.Copy(target, io.LimitReader(file, s.maxBytes+1))
		closeErr := target.Close()
		if err != nil {
			return model.Item{}, err
		}
		if closeErr != nil {
			return model.Item{}, closeErr
		}
		if size > s.maxBytes {
			return model.Item{}, fmt.Errorf("%s is larger than max upload size", header.Filename)
		}
		item.Files = append(item.Files, model.StoredFile{
			ID:          fileID,
			Name:        filepath.Base(header.Filename),
			ContentType: header.Header.Get("Content-Type"),
			Size:        size,
		})
	}

	if item.Title == "" && item.Text == "" && item.URL == "" && len(item.Files) == 0 {
		return model.Item{}, errors.New("empty item")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked(userID)
	if err != nil {
		return model.Item{}, err
	}
	items = append(items, item)
	if err := s.saveLocked(userID, items); err != nil {
		return model.Item{}, err
	}
	return item, nil
}

func (s *FileStore) Delete(userID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.loadLocked(userID)
	if err != nil {
		return err
	}
	next := items[:0]
	found := false
	for _, item := range items {
		if item.ID == id {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return os.ErrNotExist
	}
	if err := s.saveLocked(userID, next); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(s.userDir(userID), "files", id))
}

func (s *FileStore) File(userID, itemID, fileID string) (model.StoredFile, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked(userID)
	if err != nil {
		return model.StoredFile{}, "", false
	}
	for _, item := range items {
		if item.ID != itemID {
			continue
		}
		for _, file := range item.Files {
			if file.ID == fileID {
				return file, filepath.Join(s.userDir(userID), "files", itemID, fileID), true
			}
		}
	}
	return model.StoredFile{}, "", false
}

func (s *FileStore) loadLocked(userID string) ([]model.Item, error) {
	file, err := os.Open(s.indexPath(userID))
	if errors.Is(err, os.ErrNotExist) {
		return []model.Item{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var items []model.Item
	if err := json.NewDecoder(file).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *FileStore) saveLocked(userID string, items []model.Item) error {
	if err := os.MkdirAll(filepath.Join(s.userDir(userID), "files"), 0o755); err != nil {
		return err
	}
	index := s.indexPath(userID)
	tmp := index + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(items); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, index)
}

func (s *FileStore) userDir(userID string) string {
	return filepath.Join(s.dataDir, "users", userID)
}

func (s *FileStore) indexPath(userID string) string {
	return filepath.Join(s.userDir(userID), "items.json")
}

func randomID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf[:])
}
