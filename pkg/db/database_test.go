package db

import (
	"testing"
)

func TestNew(t *testing.T) {
	name := "testdb"
	path := "/tmp/testdb"

	db := New(name, path)

	if db.Name() != name {
		t.Errorf("Expected name %s, got %s", name, db.Name())
	}

	if db.Path() != path {
		t.Errorf("Expected path %s, got %s", path, db.Path())
	}
}

func TestDatabaseOperations(t *testing.T) {
	db := New("test", "/tmp/test")

	// Test Open (should not return error for now)
	err := db.Open()
	if err != nil {
		t.Errorf("Open() returned error: %v", err)
	}

	// Test Close (should not return error for now)
	err = db.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
