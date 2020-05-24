package fileutils

import "io/ioutil"

// Write writes payload to file (overwrites file if it exists)
func Write(filename string, payload string) error {
	p := []byte(payload)
	return ioutil.WriteFile(filename, p, 0644)
}

// Read reads file contents back as a string
func Read(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	return string(data), err
}
