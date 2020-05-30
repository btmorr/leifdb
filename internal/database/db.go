package database

// A Database is a key-value store
type Database struct {
	underlying map[string]string
}

// Get retreives the value for a key (empty string if key does not exist)
func (d *Database) Get(key string) string {
	r := d.underlying[key]
	return r
}

// Set assigns a value to a key
func (d *Database) Set(key string, value string) {
	d.underlying[key] = value
}

// Delete removes a key and value from the store
func (d *Database) Delete(key string) {
	delete(d.underlying, key)
}

// NewDatabase returns an initialized Database
func NewDatabase() *Database {
	return &Database{
		underlying: make(map[string]string)}
}
