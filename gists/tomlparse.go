// +build ignore

// Example of how to parse toml using viper

package tomlparse

import (
	"bytes"
	"fmt"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigType("toml")

	var tomlExample = []byte(`
[conf]
members = ["s1", "s2", "s3"]

[conf.s1]
address = "localhost"
port = 12345

[conf.s2]
address = "localhost"
port = 23456

[conf.s3]
address = "localhost"
port = 34567
`)

	viper.ReadConfig(bytes.NewBuffer(tomlExample))

	svs := viper.GetStringSlice("conf.members")
	for _, s := range svs {
		subv := viper.Sub("conf." + s)
		addr := subv.GetString("address")
		port := subv.GetInt("port")
		fmt.Printf("%s at %s:%d\n", s, addr, port)
	}
}
