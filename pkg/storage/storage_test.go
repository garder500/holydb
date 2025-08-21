package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorage_PutGetListDelete(t *testing.T) {
	dir := t.TempDir()
	s := &LocalStorage{Root: dir}
	ctx := context.Background()

	bucket := "testbucket"
	key := "path/to/object.txt"
	data := []byte("hello world")

	// Put
	if err := s.Put(ctx, bucket, key, bytes.NewReader(data)); err != nil {
		t.Fatalf("Put error: %v", err)
	}

	// Get
	rc, err := s.Get(ctx, bucket, key)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch: got %q want %q", string(got), string(data))
	}

	// List
	list, err := s.List(ctx, bucket, "path/")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(list) != 1 || list[0] != "path/to/object.txt" {
		t.Fatalf("List returned unexpected: %+v", list)
	}

	// Test metadata
	meta, err := s.GetMetadata(ctx, bucket, key)
	if err != nil {
		t.Fatalf("GetMetadata error: %v", err)
	}
	if meta["created_by"] != "LocalStorage" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}

	// Delete
	if err := s.Delete(ctx, bucket, key); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	// ensure directory removed
	if _, err := os.Stat(filepath.Join(dir, bucket, key)); !os.IsNotExist(err) {
		t.Fatalf("expected object dir to be removed, stat err: %v", err)
	}
}
