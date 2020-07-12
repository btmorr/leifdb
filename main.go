package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/btmorr/leifdb/internal/configuration"
	"github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/mgmt"
	"github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/raftserver"
	"github.com/gin-gonic/gin"
	cors "github.com/rs/cors/wrapper/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"

	_ "github.com/btmorr/leifdb/docs" // Generated by Swag CLI
)

var (
	ErrInvalidTimeouts = errors.New("appendInterval must be shorter than minimum election window")
	// LeifDBVersion is a flag the indicates the version of the current build
	LeifDBVersion = "Version not defined"
)

// @title LeifDb Client API
// @version 0.1
// @description A distributed K-V store using the Raft protocol
// @license.name MIT
// @license.url https://github.com/btmorr/leifdb/blob/edge/LICENSE

// Controller wraps routes for HTTP interface
type Controller struct {
	Node *node.Node
}

// NewController returns a Controller
func NewController(n *node.Node) *Controller {
	return &Controller{Node: n}
}

// HealthResponse is a response body template for the health route [note: this
// endpoint takes a GET request, so there is no corresponding Request type]
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
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
	c.JSON(http.StatusOK, HealthResponse{Status: "OK", Version: LeifDBVersion})
}

// ReadResponse is a response body template for the data read route [note: this
// endpoint takes a GET request, so there is no corresponding Request type]
type ReadResponse struct {
	Value string `json:"value"`
}

// Handler for database reads
// @Summary Return value from database by key
// @ID db-read
// @Accept */*
// @Produce application/json
// @Param key path string true "Key"
// @Success 200 {object} ReadResponse
// @Router /db/{key} [get]
func (ctl *Controller) handleRead(c *gin.Context) {
	key := c.Param("key")
	value := ctl.Node.Store.Get(key)

	c.JSON(http.StatusOK, ReadResponse{Value: value})
}

// WriteRequest is a request body template for the write route
type WriteRequest struct {
	Value string `json:"value"`
}

// WriteResponse is a response body template for the write route
type WriteResponse struct {
	Status string `json:"status"`
}

// Handler for database writes
// @Summary Write value to database by key
// @ID db-write
// @Accept application/json
// @Produce application/json
// @Param key path string true "Key"
// @Param body body WriteRequest true "Value"
// @Success 200 {object} WriteResponse
// @Failure 307 {string} string "Temporary Redirect"
// @Header 307 {string} Location "Redirect address of the current leader"
// @Failure 400 {string} string "Error message"
// @Router /db/{key} [put]
func (ctl *Controller) handleWrite(c *gin.Context) {
	key := c.Param("key")
	var body WriteRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	// Short circuit, if we are not the leader right now, we return
	// a redirect to the current presumptive leader
	if ctl.Node.State != mgmt.Leader {
		// We could be in a state where we don't have a leader elected yet to
		// redirect to, at this point this server can't do much
		if ctl.Node.RedirectLeader() == "" {
			c.String(http.StatusInternalServerError, node.ErrNotLeaderRecv.Error())
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("http://%s/db/%s", ctl.Node.RedirectLeader(), key))
		return
	}

	if err := ctl.Node.Set(key, body.Value); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, WriteResponse{Status: "Ok"})
}

// DeleteResponse is a response body template for the write route
type DeleteResponse struct {
	Status string `json:"status"`
}

// Handler for database deletes
// @Summary Delete item from database by key
// @ID db-delete
// @Accept */*
// @Produce application/json
// @Param key path string true "Key"
// @Success 200 {object} DeleteResponse
// @Failure 307 {string} string "Temporary Redirect"
// @Header 307 {string} Location "Redirect address of current leader"
// @Router /db/{key} [delete]
func (ctl *Controller) handleDelete(c *gin.Context) {
	// todo: add redirect if not leader, use "Location:" header
	key := c.Param("key")

	// Short circuit, if we are not the leader right now, we return
	// a redirect to the current presumptive leader
	if ctl.Node.State != mgmt.Leader {
		// We could be in a state where we don't have a leader elected yet to
		// redirect to, at this point this server can't do much
		if ctl.Node.RedirectLeader() == "" {
			c.String(http.StatusInternalServerError, node.ErrNotLeaderRecv.Error())
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("http://%s/db/%s", ctl.Node.RedirectLeader(), key))
		return
	}

	if err := ctl.Node.Delete(key); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, DeleteResponse{Status: "Ok"})
}

// buildRouter hooks endpoints for Node/Database ops
func buildRouter(n *node.Node) *gin.Engine {
	// Distilled structure of how this is hooking the database:
	// https://play.golang.org/p/c_wk9rQdJx8
	ctl := NewController(n)

	router := gin.Default()
	router.Use(cors.AllowAll())

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
	config := node.NewNodeConfig(cfg.DataDir, cfg.RaftAddr, cfg.ClientAddr, cfg.NodeIds)
	n, err := node.NewNode(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize node")
	}

	// todo: make these configurable
	upperBound := 1000
	lowerBound := upperBound / 2

	// Select random election timeout (in interval specified above), and set
	// static interval for sending append requests
	ms := (rand.Int() % lowerBound) + (upperBound - lowerBound)
	electionTimeout := time.Duration(ms) * time.Millisecond
	minimumTimeout := time.Duration(lowerBound) * time.Millisecond
	appendInterval := time.Duration(14) * time.Millisecond
	log.Info().Msgf("Election timeout: %s", electionTimeout.String())
	if minimumTimeout < appendInterval {
		// in practice, append interval should be 10x-100x shorter
		panic(ErrInvalidTimeouts)
	}

	// Reference to StateManager is not kept--all coordination is done via
	// either channels or callback hooks
	mgmt.NewStateManager(
		n.Reset,         // Node -> StateManager: reset election timer
		electionTimeout, // Time to wait for election when Follower
		n.DoElection,    // Call when election timer expires
		minimumTimeout,  // After successful election, window to bar elections
		func() {
			n.AllowVote = true
		},
		appendInterval, // Period for doing append job when Leader
		func() {
			if n.State == mgmt.Leader {
				n.SendAppend(0, n.Term)
			}
		}) // Call when append ticker cycles

	raftPortString := fmt.Sprintf(":%s", cfg.RaftPort)
	clientPortString := fmt.Sprintf(":%s", cfg.ClientPort)
	lis, err := net.Listen("tcp", raftPortString)
	if err != nil {
		log.Fatal().Err(err).Msg("Cluster interface failed to bind")
	}
	raftserver.StartRaftServer(lis, n)
	router := buildRouter(n)
	router.Run(clientPortString)
}
