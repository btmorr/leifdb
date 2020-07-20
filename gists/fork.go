// +build ignore

package main

import (
  "fmt"
  "time"

	iradix "github.com/hashicorp/go-immutable-radix"
)

func checkTree(tree *iradix.Tree) {
  time.Sleep(time.Second)
  check, _ := tree.Get([]byte("something"))
  fmt.Println("Medial:", check)
}

func main() {
  tree := iradix.New()
  tree, _, _ = tree.Insert([]byte("this"), "that")
  tree, _, _ = tree.Insert([]byte("something"), "else")
  tree, _, _ = tree.Insert([]byte("what?"), "I didn't say anything...")
  go checkTree(tree)
  tree, _, _ = tree.Insert([]byte("something"), "unspeakable")
  time.Sleep(time.Second * 2)
  res, _ := tree.Get([]byte("something"))
  fmt.Println("Finial:", res)
}
