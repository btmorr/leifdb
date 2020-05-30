// Practice implementation of the Raft distributed-consensus algorithm
// See: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf

package main

import (
	"errors"
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
	"github.com/gin-gonic/gin"
)

// A HealthResponse is a response body template for the health route [note: the
//health endpoint takes a GET request, so there is no corresponding Body type]
type HealthResponse struct {
	Status string `json:"status"`
}

// Handler for the health endpoint--not required for Raft, but useful for infrastructure
// monitoring, such as determining when a node is available in blue-green deploy
func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "Ok"})
}

// Data types for [un]marshalling JSON

// A WriteBody is a request body template for write route
type WriteBody struct {
	Value string `json:"value"`
}

// buildRouter hooks endpoints for Node/Database ops
func buildRouter(n *node.Node) *gin.Engine {
	// Distilled structure of how this is hooking the database:
	// https://play.golang.org/p/c_wk9rQdJx8
	handleRead := func(c *gin.Context) {
		key := c.Param("key")
		value := n.Store.Get(key)

		status := http.StatusOK
		c.String(status, value)
	}

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

	handleDelete := func(c *gin.Context) {
		key := c.Param("key")
		// todo: replace this with log-append once commit logic is in place
		n.Store.Delete(key)

		status := http.StatusOK
		c.String(status, "Ok")
	}

	router := gin.Default()

	router.GET("/health", handleHealth)
	router.POST("/vote", n.HandleVote)
	router.POST("/append", n.HandleAppend)
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

// EnsureDirectory creates the directory if it does not exist (fail if path
// exists and is not a directory)
func EnsureDirectory(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		var fileMode os.FileMode
		fileMode = os.ModeDir | 0775
		mdErr := os.MkdirAll(path, fileMode)
		return mdErr
	}
	if err != nil {
		return err
	}
	file, _ := os.Stat(path)
	if !file.IsDir() {
		return errors.New(path + " is not a directory")
	}
	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// todo: make ports configurable
	raftPort := "16990"
	clientPort := "8080"
	ip := GetOutboundIP()
	raftAddr := fmt.Sprintf("%s:%s", ip, raftPort)
	clientAddr := fmt.Sprintf("%s:%s", ip, clientPort)
	log.Println("Cluster interface: " + raftAddr)
	log.Println("Client interface:  " + clientAddr)

	_, err := net.Listen("tcp", fmt.Sprintf(":"+raftPort))
	if err != nil {
		log.Fatal("Cluster interface failed to bind:", err)
	}
	// todo: stand up grpc server (using variable elided just above)

	hash := fnv.New32()
	hash.Write([]byte(raftAddr))
	hashString := fmt.Sprintf("%x", hash.Sum(nil))

	// todo: make this configurable
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".leifdb", hashString)
	log.Println("Data dir: ", dataDir)
	err2 := EnsureDirectory(dataDir)
	if err2 != nil {
		panic(err2)
	}

	store := db.NewDatabase()

	config := node.NewNodeConfig(dataDir, raftAddr)

	n, err := node.NewNode(config, store)
	if err != nil {
		log.Fatal("Failed to initialize node with error:", err)
	}
	log.Println("Election timeout: ", n.ElectionTimeout.String())

	router := buildRouter(n)
	router.Run()
}
