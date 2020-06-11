package testutil

import (
	"testing"

	"github.com/btmorr/leifdb/internal/raft"
)

// CompareLogs traverses a pair of LogStores to check for equality by value
func CompareLogs(
	t *testing.T, testName string, got *raft.LogStore, expected *raft.LogStore) {
	// Note: reflect.DeepEqual failed to return true for `LogRecord`s with
	// identical contents, so have to do this instead... (DeepEqual probably
	// can't reliably traverse objects with arrays of pointers to objects)

	lengthG := len(got.Entries)
	lengthE := len(expected.Entries)
	if lengthG != lengthE {
		t.Errorf(
			"[%s] Expected %d log entries roundtrip but got %d\n",
			testName,
			lengthE,
			lengthG)
	} else {
		for idx, entry := range got.Entries {
			if entry.Term != expected.Entries[idx].Term {
				t.Errorf(
					"[%s] Expected term %d but got %d\n",
					testName,
					expected.Entries[idx].Term,
					entry.Term)
			}
			if entry.Key != expected.Entries[idx].Key {
				t.Errorf(
					"[%s] Expected key %s but got %s\n",
					testName,
					expected.Entries[idx].Key,
					entry.Key)
			}
			if entry.Value != expected.Entries[idx].Value {
				t.Errorf(
					"[%s] Expected value %s but got %s\n",
					testName,
					expected.Entries[idx].Value,
					entry.Value)
			}
		}
	}
}
