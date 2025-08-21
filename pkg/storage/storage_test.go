package storage

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestLocalStorage_MultipartStatsReconstruct(t *testing.T) {
	dir := t.TempDir()
	s := &LocalStorage{Root: dir}
	ctx := context.Background()
	bucket := "mbucket"
	key := "folder/file.bin"

	// Start multipart
	uploadID, err := s.StartMultipart(ctx, bucket, key)
	if err != nil {
		t.Fatalf("StartMultipart: %v", err)
	}
	// upload 2 parts
	if err := s.UploadPart(ctx, bucket, key, uploadID, 1, bytes.NewReader([]byte("AAAA"))); err != nil {
		t.Fatalf("UploadPart1: %v", err)
	}
	if err := s.UploadPart(ctx, bucket, key, uploadID, 2, bytes.NewReader([]byte("BBBB"))); err != nil {
		t.Fatalf("UploadPart2: %v", err)
	}
	meta := Metadata{"filename": "file.bin", "tag": "x"}
	if err := s.CompleteMultipart(ctx, bucket, key, uploadID, meta); err != nil {
		t.Fatalf("CompleteMultipart: %v", err)
	}

	// Stats
	if err := s.PutBucketMetadata(ctx, bucket, BucketMetadata{CapacityBytes: 0}); err != nil {
		t.Fatalf("PutBucketMetadata: %v", err)
	}
	st, err := s.Stats(ctx, bucket)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if st.ObjectCount != 1 {
		t.Fatalf("expected 1 object, got %d", st.ObjectCount)
	}

	// Reconstruct
	out := filepath.Join(dir, "reconstructed.bin")
	if err := s.Reconstruct(ctx, bucket, key, out, []string{"filename"}); err != nil {
		t.Fatalf("Reconstruct: %v", err)
	}
	// read header
	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("open out: %v", err)
	}
	var lnBuf [8]byte
	if _, err := io.ReadFull(f, lnBuf[:]); err != nil {
		t.Fatalf("read header len: %v", err)
	}
	var ln uint64
	for i := 0; i < 8; i++ {
		ln = (ln << 8) | uint64(lnBuf[i])
	}
	h := make([]byte, ln)
	if _, err := io.ReadFull(f, h); err != nil {
		t.Fatalf("read header: %v", err)
	}
	var hdr map[string]string
	if err := json.Unmarshal(h, &hdr); err != nil {
		t.Fatalf("unmarshal header: %v", err)
	}
	if hdr["filename"] != "file.bin" {
		t.Fatalf("unexpected header filename: %v", hdr)
	}
	f.Close()
}
