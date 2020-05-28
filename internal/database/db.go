package database

type Database struct {
	underlying map[string]string
}

func (d *Database) Get(key string) string {
	r := d.underlying[key]
	return r
}

func (d *Database) Set(key string, value string) {
	d.underlying[key] = value
}

func (d *Database) Delete(key string) {
	delete(d.underlying, key)
}

func NewDatabase() *Database {
	return &Database{
		underlying: make(map[string]string)}
}
