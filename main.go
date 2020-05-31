// Practice implementation of the Raft distributed-consensus algorithm
// See: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf

package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/raftserver"
	"github.com/btmorr/leifdb/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Data types for [un]marshalling JSON

// A WriteBody is a request body template for write route
type WriteBody struct {
	Value string `json:"value"`
}

// A HealthResponse is a response body template for the health route [note: the
//health endpoint takes a GET request, so there is no corresponding Body type]
type HealthResponse struct {
	Status string `json:"status"`
}

// buildRouter hooks endpoints for Node/Database ops
func buildRouter(n *node.Node) *gin.Engine {
	// Distilled structure of how this is hooking the database:
	// https://play.golang.org/p/c_wk9rQdJx8

	// Handler for database reads (GET /db/:key)
	handleRead := func(c *gin.Context) {
		key := c.Param("key")
		value := n.Store.Get(key)

		status := http.StatusOK
		c.String(status, value)
	}

	// Handler for database writes (POST /db/:key)
	handleWrite := func(c *gin.Context) {
		key := c.Param("key")

		var body WriteBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// todo: replace this with log-append once commit logic is in place
		n.Store.Set(key, body.Value)

		status := http.StatusOK
		c.String(status, "Ok")
	}

	// Handler for database deletes (DELETE /db/:key)
	handleDelete := func(c *gin.Context) {
		key := c.Param("key")
		// todo: replace this with log-append once commit logic is in place
		n.Store.Delete(key)

		status := http.StatusOK
		c.String(status, "Ok")
	}

	// Handler for the health endpoint--not required for Raft, but useful for
	// infrastructure monitoring, such as determining when a node is available
	// in blue-green deploy
	handleHealth := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "Ok"})
	}

	router := gin.Default()

	router.GET("/health", handleHealth)
	router.GET("/db/:key", handleRead)
	router.POST("/db/:key", handleWrite)
	router.DELETE("/db/:key", handleDelete)

	return router
}

// GetOutboundIP returns ip of preferred interface this machine
func GetOutboundIP() net.IP {
	// UDP dial is able to succeed even if the target is not available--
	// 8.8.8.8 is Google's public DNS--doesn't matter what the IP is, as
	// long as it's in a public subnet
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// get the address of the outbound interface chosen for the request
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var dataDir = *flag.String("data", "", "Path to directory for data storage")
	var raftPort = *flag.Int("raftport", 16990, "Port number for Raft gRPC service interface")
	var clientPort = *flag.Int("httpport", 8080, "Port number for database HTTP service interface")
	flag.Parse()

	raftPortString := fmt.Sprintf(":%d", raftPort)
	clientPortString := fmt.Sprintf(":%d", clientPort)
	ip := GetOutboundIP()
	raftAddr := fmt.Sprintf("%s%s", ip, raftPortString)
	clientAddr := fmt.Sprintf("%s%s", ip, clientPortString)
	log.Println("Cluster interface: " + raftAddr)
	log.Println("Client interface:  " + clientAddr)

	if dataDir == "" {
		hash := fnv.New32()
		hash.Write([]byte(raftAddr))
		hashString := fmt.Sprintf("%x", hash.Sum(nil))

		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, ".leifdb", hashString)
	}

	// Applicaiton defaults to single-node operation. To configure, copy
	// "config/default_config.toml" to "<data directory>/config.toml"
	// and then edit. By default, looks for "$HOME/.leifdb/config.toml"
	// and falls back to single-node configuration if not found.
	viper.SetConfigName("config")
	viper.AddConfigPath(dataDir)

	otherNodes := make([]string, 0, 0)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			log.Fatalln("Error parsing config file:", err)
		}
		log.Println("No config file found -- defaulting to single-node configuration")
	} else {
		// Config file found and successfully parsed
		if viper.GetString("configuration.mode") == "multi" {
			fmt.Println("Multi-node configuration. Members:")
			svs := viper.GetStringSlice("configuration.members")
			for _, s := range svs {
				subv := viper.Sub(s)
				addr := subv.GetString("host")
				port := subv.GetInt("port")
				fmt.Printf("%s at %s:%d\n", s, addr, port)
				otherNodes = append(otherNodes, fmt.Sprintf("%s:%d", addr, port))
			}
		} else {
			fmt.Println("Single-node configuration")
		}
	}

	log.Println("Data dir: ", dataDir)
	err2 := util.EnsureDirectory(dataDir)
	if err2 != nil {
		panic(err2)
	}

	store := db.NewDatabase()
	config := node.NewNodeConfig(dataDir, raftAddr)
	n, err := node.NewNode(config, store)
	if err != nil {
		log.Fatal("Failed to initialize node with error:", err)
	}

	for _, nodeId := range otherNodes {
		n.AddForeignNode(nodeId)
	}

	log.Println("Election timeout: ", n.ElectionTimeout.String())

	raftserver.StartRaftServer(raftPortString, n)
	router := buildRouter(n)
	router.Run()
}
