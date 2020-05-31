// +build ignore

// Creating an object that can be used to call methods from another object
// Used in this application for building an HTTP rounter that can call
// handler methods on the Node or the Database, without having to include
// the http framework in the node or database packages

// There's probably a nicer way to do this with interfaces, but this does
// the job for now

package functionhooks

import "fmt"

type thing struct {
	value map[string]string
}

func newThing() *thing {
	return &thing{value: make(map[string]string)}
}

type other struct {
	Set func(string, string)
	Get func(string) string
}

func builder(t *thing) *other {
	setter := func(k string, v string) {
		t.value[k] = v
	}

	getter := func(k string) string {
		return t.value[k]
	}

	return &other{
		Get: getter,
		Set: setter}
}

func f() {

	t := newThing()
	o := builder(t)

	o.Set("greeting", "Hello World!")
	fmt.Println(o.Get("greeting"))
}

func main() {
	f()
}
