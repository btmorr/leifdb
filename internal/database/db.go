package database

import (
	"encoding/json"

	iradix "github.com/hashicorp/go-immutable-radix"
)

// A Database is a key-value store
type Database struct {
	underlying *iradix.Tree
}

// Get retrieves the value for a key (empty string if key does not exist)
func (d *Database) Get(key string) string {
	r, _ := d.underlying.Get([]byte(key))
	if r == nil {
		return ""
	}
	return r.(string)
}

// Set assigns a value to a key
func (d *Database) Set(key string, value string) {
	d.underlying, _, _ = d.underlying.Insert([]byte(key), value)
}

// Delete removes a key and value from the store
func (d *Database) Delete(key string) {
	d.underlying, _, _ = d.underlying.Delete([]byte(key))
}

// NewDatabase returns an initialized Database
func NewDatabase() *Database {
	return &Database{
		underlying: iradix.New(),
	}
}

// Clone makes a new instance of a database from an existing one
func Clone(db *Database) *Database {
	return &Database{
		underlying: db.underlying,
	}
}

type pair struct {
	K string
	V string
}

// BuildSnapshot serializes the database state into a JSON array of objects
// with keys K and V and the key and value for each entry as respective values
func BuildSnapshot(db *Database) ([]byte, error) {
	accumulator := []pair{}
	db.underlying.Root().Walk(func(key []byte, value interface{}) bool {
		accumulator = append(accumulator, pair{K: string(key), V: value.(string)})
		return false
	})
	return json.Marshal(accumulator)
}

// InstallSnapshot deserializes a JSON string (following the schema created by
// BuildSnapshot) and returns a populated Database
func InstallSnapshot(data []byte) (*Database, error) {
	var pairs []pair
	db := NewDatabase()

	if err := json.Unmarshal(data, &pairs); err != nil {
		return nil, err
	}
	for _, p := range pairs {
		db.Set(p.K, p.V)
	}
	return db, nil
}
