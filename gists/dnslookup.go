//go:build ignore

// Test

package main

import (
	"fmt"
	"net"
)

func main() {
	testIp := "172.217.12.174" // ip of google.com from ping
	names, err := net.LookupAddr(testIp)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Names for %s:\n", testIp)
	for i, name := range names {
		fmt.Printf("%d: %s\n", i, name)
	}
	if len(names) > 0 {
		reverse, err := net.LookupHost(names[0])
		if err != nil {
			panic(err)
		}
		fmt.Printf("IP address for %s: %s\n", names[0], reverse)
	}
	fmt.Println("done")
}
