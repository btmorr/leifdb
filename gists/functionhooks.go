// Creating an object that can be used to call methods from another object
// Used in this application for building an HTTP rounter that can call
// handler methods on the Node or the Database, without having to include
// the http framework in the node or database packages

package main

import "fmt"

type Thing struct {
	value map[string]string
}

func NewThing() *Thing {
	return &Thing{value: make(map[string]string)}
}

type Other struct {
	Set func(string, string)
	Get func(string) string
}

func builder(t *Thing) *Other {
	setter := func(k string, v string) {
		t.value[k] = v
	}

	getter := func(k string) string {
		return t.value[k]
	}

	return &Other{
		Get: getter,
		Set: setter}
}

func f() {

	t := NewThing()
	o := builder(t)

	o.Set("greeting", "Hello World!")
	fmt.Println(o.Get("greeting"))
}

func main() {
	f()
}
