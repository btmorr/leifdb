// +build unit

package database

import (
	"testing"
)

func TestDatabase(t *testing.T) {
	d := NewDatabase()

	k := "test"
	r1 := d.Get(k)
	if r1 != "" {
		t.Errorf("Read empty key should return empty string, got: %s\n", r1)
	}

	v1 := "something"
	d.Set(k, v1)
	r2 := d.Get(k)
	if r2 != v1 {
		t.Errorf("Read \"%s\" should return \"%s\", got: %s\n", k, v1, r2)
	}

	d.Delete(k)
	r3 := d.Get(k)
	if r3 != "" {
		t.Errorf("Read deleted key should return empty string, got: %s\n", r3)
	}
}

func TestClone(t *testing.T) {
	d0 := NewDatabase()
	d0.Set("1", "one")
	d0.Set("2", "two")
	d0.Set("3", "three")

	d1 := Clone(d0)

	checkKey := "4"
	checkValue := "four"
	d0.Set(checkKey, checkValue)

	baseCheck := d0.Get(checkKey)
	if baseCheck != checkValue {
		t.Errorf(
			"Base instance should have been updated with %s=%s, got %s=%s\n",
			checkKey, checkValue, checkKey, baseCheck)
	}

	branchCheck := d1.Get(checkValue)
	if branchCheck != "" {
		t.Errorf(
			"Branch instance should not have been updated, got %s=%s\n",
			checkKey, branchCheck)
	}
}

func TestSnapshotRoundtrip(t *testing.T) {
	d0 := NewDatabase()
	keys := []string{"1", "2", "3"}
	d0.Set("1", "one")
	d0.Set("2", "two")
	d0.Set("3", "three")

	snapshot, err := BuildSnapshot(d0, Metadata{2, 1})
	if err != nil {
		t.Errorf("Error in BuildSnapshot: %v\n", err)
	}
	d1, err := InstallSnapshot(snapshot)
	if err != nil {
		t.Errorf("Error in InstallSnapshot: %v\n", err)
	}

	for _, key := range keys {
		v0 := d0.Get(key)
		v1 := d1.Get(key)
		if v0 != v1 {
			t.Errorf("Source value for key %s is %s but destination is %s\n", key, v0, v1)
		}
	}
}
