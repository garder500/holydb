package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LocalStorage is a simple filesystem-backed Storage implementation.
type LocalStorage struct {
	Root string // root directory where buckets are stored
}

// ensureBucketDir ensures the bucket directory exists and returns its path.
func (s *LocalStorage) ensureBucketDir(bucket string) (string, error) {
	dir := filepath.Join(s.Root, bucket)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *LocalStorage) Put(ctx context.Context, bucket, key string, r io.Reader) error {
	// default to single part -> store as part.1 inside object dir
	meta := Metadata{"created_by": "LocalStorage"}
	return s.PutWithMetadata(ctx, bucket, key, r, meta)
}

func (s *LocalStorage) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	objDir := filepath.Join(s.Root, bucket, key)
	// list parts named part.N
	files, err := os.ReadDir(objDir)
	if err != nil {
		return nil, err
	}
	var parts []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if strings.HasPrefix(f.Name(), "part.") {
			parts = append(parts, filepath.Join(objDir, f.Name()))
		}
	}
	sort.Strings(parts)
	if len(parts) == 0 {
		return nil, fmt.Errorf("no parts found for object %s/%s", bucket, key)
	}
	// create a reader that concatenates all parts
	readers := make([]io.ReadCloser, 0, len(parts))
	for _, p := range parts {
		rc, err := os.Open(p)
		if err != nil { // close all opened
			for _, r := range readers {
				r.Close()
			}
			return nil, err
		}
		readers = append(readers, rc)
	}
	return &multiReadCloser{readers: readers}, nil
}

type multiReadCloser struct{ readers []io.ReadCloser }

func (m *multiReadCloser) Read(p []byte) (int, error) {
	for len(m.readers) > 0 {
		n, err := m.readers[0].Read(p)
		if err == io.EOF {
			m.readers[0].Close()
			m.readers = m.readers[1:]
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
	return 0, io.EOF
}
func (m *multiReadCloser) Close() error {
	var firstErr error
	for _, r := range m.readers {
		if err := r.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	m.readers = nil
	return firstErr
}

func (s *LocalStorage) PutWithMetadata(ctx context.Context, bucket, key string, r io.Reader, meta Metadata) error {
	dir, err := s.ensureBucketDir(bucket)
	if err != nil {
		return err
	}
	objDir := filepath.Join(dir, key)
	if err := os.MkdirAll(objDir, 0o755); err != nil {
		return err
	}
	// write single part as part.1
	partPath := filepath.Join(objDir, "part.1")
	f, err := os.Create(partPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// write metadata
	if err := s.writeMeta(objDir, meta); err != nil {
		return err
	}
	return nil
}

func (s *LocalStorage) PutMetadata(ctx context.Context, bucket, key string, meta Metadata) error {
	dir, err := s.ensureBucketDir(bucket)
	if err != nil {
		return err
	}
	objDir := filepath.Join(dir, key)
	if err := os.MkdirAll(objDir, 0o755); err != nil {
		return err
	}
	return s.writeMeta(objDir, meta)
}

func (s *LocalStorage) GetMetadata(ctx context.Context, bucket, key string) (Metadata, error) {
	objDir := filepath.Join(s.Root, bucket, key)
	data, err := os.ReadFile(filepath.Join(objDir, "data.meta"))
	if err != nil {
		return nil, err
	}
	var m Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *LocalStorage) writeMeta(objDir string, meta Metadata) error {
	b, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	tmp := filepath.Join(objDir, "data.meta.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(objDir, "data.meta"))
}

func (s *LocalStorage) Delete(ctx context.Context, bucket, key string) error {
	path := filepath.Join(s.Root, bucket, key)
	return os.RemoveAll(path)
}

func (s *LocalStorage) List(ctx context.Context, bucket, prefix string) ([]string, error) {
	base := filepath.Join(s.Root, bucket)
	// Collect object directories (each object is a directory containing parts and data.meta)
	set := map[string]struct{}{}
	// if base does not exist, return empty list
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return nil, nil
	}
	err := filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(base, p)
		if err != nil {
			return err
		}
		dir := filepath.Dir(rel)
		// if file sits directly at base (unlikely), dir == "." -> use file name as key
		if dir == "." {
			dir = rel
		}
		if prefix == "" || strings.HasPrefix(dir, prefix) {
			set[dir] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var out []string
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}
