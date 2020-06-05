// Practice implementation of the Raft distributed-consensus algorithm
// See: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf

package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/btmorr/leifdb/internal/configuration"
	"github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/raftserver"
	"github.com/gin-gonic/gin"
	cors "github.com/rs/cors/wrapper/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"

	_ "github.com/btmorr/leifdb/docs" // Generated by Swag CLI
)

// @title LeifDb Client API
// @version 0.1
// @description A distributed K-V store using the Raft protocol
// @license.name MIT
// @license.url https://github.com/btmorr/leifdb/blob/edge/LICENSE

// Data types for [un]marshalling JSON

// A HealthResponse is a response body template for the health route [note: the
//health endpoint takes a GET request, so there is no corresponding Body type]
type HealthResponse struct {
	Status string `json:"status"`
}

// Controller wraps routes for HTTP interface
type Controller struct {
	Node *node.Node
}

// NewController returns a Controller
func NewController(n *node.Node) *Controller {
	return &Controller{Node: n}
}

// Handler for the health endpoint--not required for Raft, but useful for
// infrastructure monitoring, such as determining when a node is available
// in blue-green deploy
// @Summary Return server health status
// @ID http-health
// @Accept */*
// @Produce application/json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (ctl *Controller) handleHealth(c *gin.Context) {
	// todo: add checking on other aspects of the system, remember to
	// include an at-failure directive for swagger
	c.JSON(http.StatusOK, HealthResponse{Status: "Ok"})
}

// Handler for database reads (GET /db/:key)
// @Summary Return value from database by key
// @ID db-read
// @Accept */*
// @Produce text/plain
// @Param key path string true "Key"
// @Success 200 {string} string "Ok"
// @Router /db/{key} [get]
func (ctl *Controller) handleRead(c *gin.Context) {
	key := c.Param("key")
	value := ctl.Node.Store.Get(key)

	status := http.StatusOK
	c.String(status, value)
}

// Handler for database writes (PUT /db/:key?value="<value>")
// @Summary Write value to database by key
// @ID db-write
// @Accept */*
// @Produce text/plain
// @Param key path string true "Key"
// @Param value query string false "Value"
// @Success 200 {string} string "Ok"
// @Failure 307 {string} string "Temporary Redirect"
// @Header 307 {string} Location "localhost:8181"
// @Router /db/{key} [put]
func (ctl *Controller) handleWrite(c *gin.Context) {
	// todo: add redirect if not leader, use "Location:" header
	key := c.Param("key")
	value := c.Request.URL.Query().Get("value")

	if err := ctl.Node.Set(key, value); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	status := http.StatusOK
	c.String(status, "Ok")
}

// Handler for database deletes (DELETE /db/:key)
// @Summary Delete item from database by key
// @ID db-delete
// @Accept */*
// @Produce text/plain
// @Param key path string true "Key"
// @Success 200 {string} string "Ok"
// @Failure 307 {string} string "Temporary Redirect"
// @Header 307 {string} Location "localhost:8181"
// @Router /db/{key} [delete]
func (ctl *Controller) handleDelete(c *gin.Context) {
	// todo: add redirect if not leader, use "Location:" header
	key := c.Param("key")

	if err := ctl.Node.Delete(key); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	status := http.StatusOK
	c.String(status, "Ok")
}

// buildRouter hooks endpoints for Node/Database ops
func buildRouter(n *node.Node) *gin.Engine {
	// Distilled structure of how this is hooking the database:
	// https://play.golang.org/p/c_wk9rQdJx8
	ctl := NewController(n)

	router := gin.Default()
	router.Use(cors.Default())

	router.GET("/health", ctl.handleHealth)

	dbRouter := router.Group("/db")
	{
		dbRouter.GET("/:key", ctl.handleRead)
		dbRouter.PUT("/:key", ctl.handleWrite)
		dbRouter.DELETE("/:key", ctl.handleDelete)
	}
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.StaticFile("/", "./docs/swagger.json")

	return router
}

// For more human-readable logs, uncomment the following function and add the
// associated imports ("github.com/rs/zerolog", and "os")
// todo: make this configurable
func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func main() {

	cfg := configuration.BuildServerConfig()
	fmt.Printf("Configuration:\n%+v\n\n", *cfg)

	store := database.NewDatabase()
	config := node.NewNodeConfig(cfg.DataDir, cfg.RaftAddr, cfg.NodeIds)
	n, err := node.NewNode(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize node")
	}

	log.Info().Msgf("Election timeout: %s", n.ElectionTimeout.String())

	raftPortString := fmt.Sprintf(":%d", cfg.RaftPort)
	clientPortString := fmt.Sprintf(":%d", cfg.ClientPort)
	go raftserver.StartRaftServer(raftPortString, n)
	router := buildRouter(n)
	router.Run(clientPortString)
}
