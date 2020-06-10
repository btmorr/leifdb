// +build bench

package node

import (
	"log"
	"testing"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/btmorr/leifdb/internal/util"
	"github.com/rs/zerolog"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
}

// checkForeignNodeMock is used to skip membership checks during test, so that a
// Node will perform raft operations without creating a full multi-node config
func checkForeignNodeMock(addr string, known map[string]*ForeignNode) bool {
	return true
}

func setupNodeBench(b *testing.B) *Node {
	addr := "localhost:8080"

	testDir, err := util.CreateTmpDir(".tmp-leifdb")
	if err != nil {
		log.Fatalln("Error creating test dir:", err)
	}
	b.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	store := db.NewDatabase()

	config := NewNodeConfig(testDir, addr, make([]string, 0, 0))
	n, _ := NewNode(config, store)
	n.CheckForeignNode = checkForeignNodeMock
	return n
}

func BenchmarkEmptyAppend(b *testing.B) {
	n := setupNodeBench(b)
	req := &raft.AppendRequest{
		Term:         1,
		LeaderId:     "localhost:8181",
		PrevLogIndex: -1,
		PrevLogTerm:  -1,
		Entries:      make([]*raft.LogRecord, 0, 0),
		LeaderCommit: -1}

	for i := 0; i < b.N; i++ {
		n.HandleAppend(req)
	}
}

func BenchmarkFullAppend(b *testing.B) {
	n := setupNodeBench(b)

	for i := 1; i <= b.N; i++ {
		term := int64(i)
		record := &raft.LogRecord{
			Term:   term,
			Action: raft.LogRecord_SET,
			Key:    "a",
			Value:  "b"}
		req := &raft.AppendRequest{
			Term:         term,
			LeaderId:     "localhost:8181",
			PrevLogIndex: term - 1,
			PrevLogTerm:  term - 1,
			Entries:      []*raft.LogRecord{record},
			LeaderCommit: term - 1}
		n.HandleAppend(req)
	}
}
