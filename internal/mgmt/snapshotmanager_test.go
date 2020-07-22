// +build unit

package mgmt

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/util"
)

// checkMock is used to skip membership checks during test, so that a Node will
// respond to RPC calls without creating a full multi-node configuration
func checkMock(addr string, known map[string]*node.ForeignNode) bool {
	return true
}

func setupTestDir(t *testing.T) string {
	testDir, err := util.CreateTmpDir(".tmp-leifdb")
	if err != nil {
		log.Fatalln("Error creating test dir:", err)
	}
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})
	return testDir
}

// setupServer configures a Database and a Node, mocks cluster membership check,
// and creates a test directory that is cleaned up after each test
func setupServer(t *testing.T) *node.Node {
	addr := "localhost:16990"
	clientAddr := "localhost:8080"

	testDir := setupTestDir(t)
	store := db.NewDatabase()

	config := node.NewNodeConfig(testDir, addr, clientAddr, make([]string, 0, 0))
	n, _ := node.NewNode(config, store)
	n.CheckForeignNode = checkMock
	return n
}

func TestFindExisting(t *testing.T) {
	testDir := setupTestDir(t)

	// create dummy snapshot files
	for _, n := range []string{"000013", "000014"} {
		fileName := filepath.Join(testDir, prefix+n)
		_, err := os.Stat(fileName)
		if os.IsNotExist(err) {
			file, err := os.Create(fileName)
			if err != nil {
				log.Fatal(err)
			}
			file.Close()
		}
	}

	snapshots, nextIndex := findExistingSnapshots(testDir)

	if len(snapshots) != 2 {
		t.Errorf("Expected 2 snapshots found, got %d\n", len(snapshots))
	}
	if nextIndex != 15 {
		t.Errorf("Expected next index of 15, got %d\n", nextIndex)
	}
}

func TestCloneAndSerialize(t *testing.T) {
	n := setupServer(t)
	n.Store.Set("ice", "cream")
	n.Store.Set("straw", "bale")
	var snapshot []byte
	var err error
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		snapshot, err = cloneAndSerialize(n)
		wg.Done()
	}()
	// simulate raft write
	n.Lock()
	n.Store.Set("straw", "berry")
	n.Unlock()

	wg.Wait()
	fmt.Println("=-------> snapshot <-------=")
	fmt.Println(string(snapshot))
	reconstituted, err := db.InstallSnapshot(snapshot)
	if err != nil {
		t.Errorf("Error installing snapshot: %v\n", err)
	}

	ice := reconstituted.Get("ice")
	if ice != "cream" {
		t.Errorf("Expected ice cream but got ice %s\n", ice)
	}

	straw := reconstituted.Get("straw")
	if straw != "bale" {
		t.Errorf("Expected straw bale but got straw %s\n", straw)
	}
}
