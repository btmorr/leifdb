package configuration

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/btmorr/leifdb/internal/util"
	"github.com/rs/zerolog"
)

var (
	// ErrInvalidMultiConfig indicates a configuration mode of "multi" but no
	// other nodes included in the configuration
	ErrInvalidMultiConfig = errors.New(
		"Multi-node configuration must include more than one node")

	// ErrSelfNotInConfig indicates a configuration that does not include this
	// node--double-check the config file and make sure that one entry matches
	// the preferred outbound IP address and chosen RaftPort for this server
	ErrSelfNotInConfig = errors.New(
		"This node must be included in the configration")
)

// GetOutboundIP returns ip of preferred interface this machine
func GetOutboundIP() net.IP {
	// UDP dial is able to succeed even if the target is not available--
	// 8.8.8.8 is Google's public DNS--doesn't matter what the IP is, as
	// long as it's in a public subnet
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		panic(err)
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

// Single is a cluster of one node, which only trivally involves Raft functions
// Multi is a cluster of more than one node, requiring full Raft coordination
const (
	SingleNode ClusterMode = "single"
	MultiNode              = "multi"
)

// A ServerConfig contains the configuation values needed for other parts of
// the server (see `BuildConfig`)
type ServerConfig struct {
	Host       string
	DataDir    string
	RaftPort   string
	RaftAddr   string
	ClientPort string
	ClientAddr string
	Mode       ClusterMode
	NodeIds    []string
}

type ClusterConfig struct {
	Mode    ClusterMode
	NodeIds []string
}

func resolveHostPort(addr string) ([]string, string) {
	// is addr's host portion an IP or a DNS name?
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Printf("Error splitting %s\n", addr)
		panic(err)
	}
	var ips []string
	ip := net.ParseIP(host)
	if ip == nil {
		// this means host is not an IP (expect it is a DNS name)
		ips, err = net.LookupHost(host)
		if err != nil {
			fmt.Printf("Error looking up %s\n", host)
			panic(err)
		}
	} else {
		ips = []string{string(ip)}
	}
	return ips, port
}

// selectForeignNodes returns all members that are not the same as raftAddr.
// raftAddr may be an IP address or a DNS name. Same goes for each entry in
// members. Resolve all of them to all their possible IPs, and check that at
// least one of the IP:Port addresses for raftAddr is in the list of IP:Port
// addresses of all members.
func selectForeignNodes(raftAddr string, members []string) []string {
	foreignNodes := []string{}
	ips, port := resolveHostPort(raftAddr)

	candidates := []string{}
	for _, ip := range ips {
		candidates = append(candidates, net.JoinHostPort(ip, port))
	}

	// for each member of the cluster, find all possible IP addresses and check
	// against all possible IPs for the current node
	for _, member := range members {
		match := false
		memberIps, memberPort := resolveHostPort(member)

		for _, ip := range memberIps {
			option := net.JoinHostPort(ip, memberPort)
			for _, candidate := range candidates {
				if candidate == option {
					match = true
				}
			}
		}
		// If none of the IPs for the current node match any IP for the member,
		// add the member to the foreign node list
		if !match {
			foreignNodes = append(foreignNodes, member)
		}
	}

	return foreignNodes
}

func buildClusterConfig(dataDir string, raftAddr string) *ClusterConfig {
	// Application defaults to single-node operation
	var members []string
	var mode ClusterMode
	modeEnv := os.Getenv("LEIFDB_MODE")

	if modeEnv == "multi" {
		mode = MultiNode
		members = strings.Split(os.Getenv("LEIFDB_MEMBER_NODES"), ",")
		fmt.Printf("Multi-node cluster, members: %v\n", members)
	} else {
		mode = SingleNode
		members = []string{raftAddr}
		fmt.Printf("Single-node cluster, address: %v\n", members)
	}

	foreignNodes := selectForeignNodes(raftAddr, members)
	if len(foreignNodes) == len(members) {
		fmt.Printf(
			"Error: %s not found in cluster membership %v\n", raftAddr, members)
		panic(ErrSelfNotInConfig)
	}

	return &ClusterConfig{
		Mode:    mode,
		NodeIds: foreignNodes}
}

// BuildConfig performs all operations needed to parse configuration options
// from environment variables, precompute other static configration values from
// those options, and perform tasks that ensure that the configration is
// locally valid (such as checking that the IP and RaftPort for this machine
// are included in the cluster configuration)
func BuildServerConfig() *ServerConfig {
	dataDir := os.Getenv("LEIFDB_DATA_DIR")
	raftPort := os.Getenv("LEIFDB_RAFT_PORT")
	if raftPort == "" {
		raftPort = "16990"
	}
	if _, err := strconv.Atoi(raftPort); err != nil {
		panic(err)
	}

	clientPort := os.Getenv("LEIFDB_HTTP_PORT")
	if clientPort == "" {
		clientPort = "8080"
	}
	if _, err := strconv.Atoi(clientPort); err != nil {
		panic(err)
	}

	host := os.Getenv("LEIFDB_HOST")
	if host == "" {
		host = GetOutboundIP().String()
	}

	raftAddr := net.JoinHostPort(host, raftPort)
	clientAddr := net.JoinHostPort(host, clientPort)

	if dataDir == "" {
		hash := fnv.New32()
		hash.Write([]byte(raftAddr))
		hashString := fmt.Sprintf("%x", hash.Sum(nil))

		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, ".leifdb", hashString)
	}
	err2 := util.EnsureDirectory(dataDir)
	if err2 != nil {
		panic(err2)
	}

	ccfg := buildClusterConfig(dataDir, raftAddr)

	return &ServerConfig{
		Host:       host,
		DataDir:    dataDir,
		RaftPort:   raftPort,
		RaftAddr:   raftAddr,
		ClientPort: clientPort,
		ClientAddr: clientAddr,
		Mode:       ccfg.Mode,
		NodeIds:    ccfg.NodeIds}
}

// GetLogLevel fetches the log level set at the env var: LEIF_LOG_LEVEL
func GetLogLevel() zerolog.Level {
	logLevel, ok := os.LookupEnv("LEIF_LOG_LEVEL")

	if !ok {
		return zerolog.InfoLevel
	}

	switch logLevel {
	case "panic":
		return zerolog.PanicLevel
	case "fatal":
		return zerolog.FatalLevel
	case "error":
		return zerolog.ErrorLevel
	case "warn":
		return zerolog.WarnLevel
	case "info":
		return zerolog.InfoLevel
	case "debug":
		return zerolog.DebugLevel
	case "trace":
		return zerolog.TraceLevel
	default:
		return zerolog.InfoLevel
	}
}
