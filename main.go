// Practice implementation of the Raft distributed-consensus algorithm
// See: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/btmorr/leifdb/internal/fileutils"
	. "github.com/btmorr/leifdb/internal/database"
	. "github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"github.com/golang/protobuf/proto"
)

// Handler for the health endpoint--not required for Raft, but useful for infrastructure
// monitoring, such as determining when a node is available in blue-green deploy
func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "Ok"})
}

func buildRouter(d *Database) *gin.Engine {
	router := gin.Default()

	router.GET("/health", handleHealth)
	router.GET("/", d.handleRead)
	router.POST("/", d.handleWrite)
	router.DELETE("/", d.handleDelete)

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
	dataPort := "8080"
	raftAddr := fmt.Sprintf("%s:%s", GetOutboundIP(), raftPort)
	log.Println("Cluster interface:", raftAddr)
	lis, err := net.Listen("tcp", fmt.Sprintf(":" + raftPort))
	if err != nil {
		log.Fatal("Cluster interface failed to bind:", err)
	}


	hash := fnv.New32()
	hash.Write([]byte(addr))
	hashString := fmt.Sprintf("%x", hash.Sum(nil))

	// todo: make this configurable
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".leifdb", hashString)
	log.Println("Data dir: ", dataDir)
	err := EnsureDirectory(dataDir)
	if err != nil {
		panic(err)
	}

	config := NewNodeConfig(dataDir, raftAddr)

	node, err := NewNode(config)
	if err != nil {
		log.Println("Failed to initialize node")
		panic(err)
	}
	log.Println("Election timeout: ", node.electionTimeout.String())

	router := buildRouter(node)
	router.Run()
}
