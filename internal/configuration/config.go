package configuration

import (
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/btmorr/leifdb/internal/util"
	"github.com/spf13/viper"
)

var (
	// ErrInvalidMultiConfig indicates a configuration mode of "multi" but no other
	// nodes included in the configuration
	ErrInvalidMultiConfig = errors.New("Multi-node configuration must include more than one node")

	// ErrSelfNotInConfig indicates a configuration that does not include this node--
	// double-check the config file and make sure that one of the entries matches the
	// preferred outbound IP address and chosen RaftPort for this server
	ErrSelfNotInConfig = errors.New("This node must be included in the configration")
)

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

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ClusterMode is one of "single" or "multi" for describing distribution mode
type ClusterMode string

// Single is a configuration of one node, which only trivally involves Raft functions
// Multi is a configuration of more than one nodes, requiring full Raft coordination
const (
	SingleNode ClusterMode = "single"
	MultiNode              = "multi"
)

// A ServerConfig contains the configuation values needed for other parts of the
// server (see `BuildConfig`)
type ServerConfig struct {
	IpAddr     net.IP
	DataDir    string
	RaftPort   int
	RaftAddr   string
	CleintPort int
	ClientAddr string
	ClusterCfg *ClusterConfig
}

type ClusterConfig struct {
	Mode    ClusterMode
	NodeIds []string
}

func buildClusterConfig(dataDir string, raftAddr string) *ClusterConfig {
	// Application defaults to single-node operation. To configure, copy
	// "config/default_config.toml" to "<data directory>/config.toml"
	// and then edit. By default, looks for "$HOME/.leifdb/config.toml"
	// and falls back to single-node configuration if not found.
	viper.SetConfigName("config")
	viper.AddConfigPath(dataDir)

	mode := SingleNode
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
			mode = MultiNode
			// fmt.Println("Multi-node configuration. Members:")

			svs := viper.GetStringSlice("configuration.members")
			if len(svs) < 2 {
				panic(ErrInvalidMultiConfig)
			}

			selfInConfig := false
			for _, s := range svs {
				subv := viper.Sub(s)
				addr := subv.GetString("host")
				port := subv.GetInt("port")
				// fmt.Printf("%s at %s:%d\n", s, addr, port)
				nodeId := fmt.Sprintf("%s:%d", addr, port)
				if nodeId != raftAddr {
					// Don't include self in `otherNodes`
					otherNodes = append(otherNodes, nodeId)
				} else {
					selfInConfig = true
				}
				if !selfInConfig {
					panic(ErrSelfNotInConfig)
				}
			}
		} else {
			// fmt.Println("Single-node configuration")
		}
	}
	return &ClusterConfig{
		Mode:    mode,
		NodeIds: otherNodes}
}

// BuildConfig performs all operations needed to parse configuration options, whether
// commandline flags, config file parsing, or boot-time environment variable checks,
// precompute other static configration values from those options, and perform tasks
// that ensure that the configration is locally valid (such as checking that the IP
// and RaftPort for this machine are included in the cluster configuration)
func BuildServerConfig() *ServerConfig {
	var dataDir = *flag.String("data", "", "Path to directory for data storage")
	var raftPort = *flag.Int("raftport", 16990, "Port number for Raft gRPC service interface")
	var clientPort = *flag.Int("httpport", 8080, "Port number for database HTTP service interface")
	flag.Parse()

	ip := GetOutboundIP()

	raftAddr := fmt.Sprintf("%s:%d", ip, raftPort)
	clientAddr := fmt.Sprintf("%s:%d", ip, clientPort)
	// log.Println("Cluster interface: " + raftAddr)
	// log.Println("Client interface:  " + clientAddr)

	if dataDir == "" {
		hash := fnv.New32()
		hash.Write([]byte(raftAddr))
		hashString := fmt.Sprintf("%x", hash.Sum(nil))

		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, ".leifdb", hashString)
	}
	// log.Println("Data dir: ", dataDir)
	err2 := util.EnsureDirectory(dataDir)
	if err2 != nil {
		panic(err2)
	}

	ccfg := buildClusterConfig(dataDir, raftAddr)

	return &ServerConfig{
		IpAddr:     ip,
		DataDir:    dataDir,
		RaftPort:   raftPort,
		RaftAddr:   raftAddr,
		CleintPort: clientPort,
		ClientAddr: clientAddr,
		ClusterCfg: ccfg}
}
