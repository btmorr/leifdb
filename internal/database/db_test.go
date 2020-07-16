// +build unit

package database

import (
	"bufio"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/btmorr/leifdb/internal/util"
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

func BenchmarkDatabasePersist(b *testing.B) {
	// testing file write in relation to snapshotting--this doesn't test
	// or prove anything about the live system currently, but leaving it
	// here for now until there is an actual snapshot behavior to
	// benchmark
	testDir, err := util.CreateTmpDir(".tmp-leifdb")
	if err != nil {
		log.Fatalln("Error creating test dir:", err)
	}
	b.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	d := NewDatabase()
	for i := 0; i < 1000000; i++ {
		s := string(i)
		// repeat of 10 yields a db file around 44Mb
		// repeat of 1000 yields a file around 3.7Gb
		d.Set(s, strings.Repeat(s, 10))
	}
	// b.ResetTimer()
	f, err := os.Create(testDir + "/data")
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(f)
	for k, v := range d.underlying {
		w.WriteString(k + ": " + v + "\n")
	}
	w.Flush()
}
