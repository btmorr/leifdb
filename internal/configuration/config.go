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

	// ErrInvalidNSnapshots indicates that a number less than 1 was specified
	// for retaining snapshots
	ErrInvalidNSnapshots = errors.New(
		"Number of snapshots to retain must be greater than 0")
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

// Single is a cluster of one node, which only trivially involves Raft
// Multi is a cluster of more than one node, requiring full Raft coordination
const (
	SingleNode ClusterMode = "single"
	MultiNode              = "multi"
)

// A ServerConfig contains the configuration values needed for other parts of
// the server (see `BuildConfig`)
type ServerConfig struct {
	Host              string
	DataDir           string
	SnapshotThreshold int
	RetainNSnapshots  int
	RaftPort          string
	RaftAddr          string
	ClientPort        string
	ClientAddr        string
	Mode              ClusterMode
	NodeIds           []string
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

func getEnvDefault(key string, fallback func() string) string {
	res := os.Getenv(key)
	if res == "" {
		res = fallback()
	}
	return res
}

func verifyInt(s string) {
	if _, err := strconv.Atoi(s); err != nil {
		panic(err)
	}
}

// BuildConfig performs all operations needed to parse configuration options
// from environment variables, pre-compute other static configuration values
// from those options, and perform tasks that ensure that the configuration is
// locally valid (such as checking that the IP and RaftPort for this machine
// are included in the cluster configuration)
func BuildServerConfig() *ServerConfig {
	raftPort := getEnvDefault(
		"LEIFDB_RAFT_PORT", func() string { return "16990" })
	verifyInt(raftPort)

	clientPort := getEnvDefault(
		"LEIFDB_HTTP_PORT", func() string { return "8080" })
	verifyInt(clientPort)

	host := getEnvDefault(
		"LEIFDB_HOST", func() string { return GetOutboundIP().String() })

	raftAddr := net.JoinHostPort(host, raftPort)
	clientAddr := net.JoinHostPort(host, clientPort)

	dataDir := getEnvDefault("LEIFDB_DATA_DIR", func() string {
		hash := fnv.New32()
		hash.Write([]byte(raftAddr))
		hashString := fmt.Sprintf("%x", hash.Sum(nil))

		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, ".leifdb", hashString)
	})

	err2 := util.EnsureDirectory(dataDir)
	if err2 != nil {
		panic(err2)
	}

	ccfg := buildClusterConfig(dataDir, raftAddr)

	// by default, snapshot when log is 1Gb
	snapshotThreshold := getEnvDefault(
		"LEIFDB_SNAPSHOT_THRESHOLD", func() string { return string(1024*1024*1024) })
	verifyInt(snapshotThreshold)
	snapshotBytes, _ := strconv.Atoi(snapshotThreshold)

	retain := getEnvDefault(
		"LEIFDB_RETAIN_N_SNAPSHOTS", func() string { return "1" })
	verifyInt(retain)
	retainNSnapshots, _ := strconv.Atoi(retain)
	if retainNSnapshots < 1 {
		panic(ErrInvalidNSnapshots)
	}

	return &ServerConfig{
		Host:       host,
		DataDir:    dataDir,
		SnapshotThreshold: snapshotBytes,
		RetainNSnapshots: retainNSnapshots,
		RaftPort:   raftPort,
		RaftAddr:   raftAddr,
		ClientPort: clientPort,
		ClientAddr: clientAddr,
		Mode:       ccfg.Mode,
		NodeIds:    ccfg.NodeIds}
}

// GetLogLevel fetches the log level set at the env var: LEIF_LOG_LEVEL
func GetLogLevel() zerolog.Level {
	logLevel, ok := os.LookupEnv("LEIFDB_LOG_LEVEL")

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
