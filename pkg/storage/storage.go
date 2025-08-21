package storage

import (
	"context"
	"io"
)

// Storage is a minimal object-storage interface (S3-like).
// Implementations must store objects by bucket and key.
type Storage interface {
	// Put stores object data (legacy: single part) without metadata.
	Put(ctx context.Context, bucket, key string, r io.Reader) error
	// PutWithMetadata stores object data and writes the metadata file.
	PutWithMetadata(ctx context.Context, bucket, key string, r io.Reader, meta Metadata) error
	// Get returns a ReadCloser that yields the concatenated parts of the object.
	Get(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	// PutMetadata writes or updates only the metadata for an existing or new object.
	PutMetadata(ctx context.Context, bucket, key string, meta Metadata) error
	// GetMetadata reads the object's metadata (from data.meta).
	GetMetadata(ctx context.Context, bucket, key string) (Metadata, error)
	// Delete removes the object (directory and all parts/meta).
	Delete(ctx context.Context, bucket, key string) error
	// List returns object keys (directories) under the bucket that match prefix.
	List(ctx context.Context, bucket, prefix string) ([]string, error)
}
