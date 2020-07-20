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
