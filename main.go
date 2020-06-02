// Practice implementation of the Raft distributed-consensus algorithm
// See: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf

package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/btmorr/leifdb/internal/configuration"
	"github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/raftserver"
	"github.com/gin-gonic/gin"
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

func main() {
	cfg := configuration.BuildServerConfig()
	fmt.Printf("Configuration:\n%+v\n", cfg)

	store := database.NewDatabase()
	config := node.NewNodeConfig(cfg.DataDir, cfg.RaftAddr)
	n, err := node.NewNode(config, store)
	if err != nil {
		log.Fatal("Failed to initialize node with error:", err)
	}

	for _, nodeId := range cfg.ClusterCfg.NodeIds {
		n.AddForeignNode(nodeId)
	}

	log.Println("Election timeout: ", n.ElectionTimeout.String())

	raftPortString := fmt.Sprintf(":%d", cfg.RaftPort)
	raftserver.StartRaftServer(raftPortString, n)
	router := buildRouter(n)
	router.Run()
}
