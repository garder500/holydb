package storage

import (
	"encoding/json"
)

// Metadata represents arbitrary key/value metadata stored alongside an object.
// Stored as JSON for now but the file is written as binary (not human-focused).
type Metadata map[string]string

// BucketMetadata stores per-bucket settings.
type BucketMetadata struct {
	CapacityBytes int64 `json:"capacity_bytes"` // max capacity for bucket (0 = unlimited)
	RetentionDays int   `json:"retention_days"` // retention in days (0 = unlimited)
}

// Stats returns quick analytics for a bucket
type Stats struct {
	ObjectCount   int64 `json:"object_count"`
	UsedBytes     int64 `json:"used_bytes"`
	CapacityBytes int64 `json:"capacity_bytes"` // copied from bucket metadata
}

// MarshalJSON ensures deterministic JSON for metadata.
func (m Metadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string(m))
}

// UnmarshalJSON for Metadata
func (m *Metadata) UnmarshalJSON(b []byte) error {
	var tmp map[string]string
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*m = tmp
	return nil
}
