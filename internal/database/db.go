package database

import iradix "github.com/hashicorp/go-immutable-radix"

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
