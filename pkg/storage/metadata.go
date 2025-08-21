package storage

import "encoding/json"

// Metadata represents arbitrary key/value metadata stored alongside an object.
type Metadata map[string]string

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
