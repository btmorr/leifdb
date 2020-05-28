package db

type Database struct {
	underlying map[string]string
}

func (d *Database) Set(key string, value string) {
	d.underlying[key] = value
}

func (d *Database) Delete(key string) {
	delete(d.underlying, key)
}
