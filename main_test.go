// Unit tests on non-rpc functionality

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/util"
	"github.com/gin-gonic/gin"
)

// setupServer configures a Database, Node, and router for test, and creates
// a test directory that is automatically cleaned up after each test
func setupServer(t *testing.T) (*gin.Engine, *node.Node) {
	addr := "localhost:8080"

	testDir, err := util.CreateTmpDir(".tmp-leifdb")
	if err != nil {
		log.Fatalln("Error creating test dir:", err)
	}
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	store := db.NewDatabase()

	config := node.NewNodeConfig(testDir, addr)
	node, _ := node.NewNode(config, store)
	router := buildRouter(node)
	return router, node
}

func TestHealthRoute(t *testing.T) {
	log.Println("~~~ TestHealthRoute")
	router, _ := setupServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Non-200 health status")
	}
}

func TestReadAfterWrite(t *testing.T) {
	log.Println("~~~ TestReadAfterWrite")
	router, _ := setupServer(t)

	v := "testy"
	body1 := WriteBody{
		Value: v}
	b1, _ := json.Marshal(body1)
	br1 := bytes.NewReader(b1)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/db/stuff", br1)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Non-200 health status in POST:", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/db/stuff", nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Error("Non-200 health status in GET:", w2.Code)
	}

	raw, _ := ioutil.ReadAll(w2.Body)
	if string(raw) != v {
		t.Error("Incorrect response:", string(raw))
	}
}

func TestDelete(t *testing.T) {
	log.Println("~~~ TestDelete")
	router, _ := setupServer(t)

	v := "testy"
	body1 := WriteBody{
		Value: v}
	b1, _ := json.Marshal(body1)
	br1 := bytes.NewReader(b1)

	uri := "/db/stuff"

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", uri, br1)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Non-200 health status in POST:", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", uri, nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Error("Non-200 health status in GET:", w2.Code)
	}

	raw, _ := ioutil.ReadAll(w2.Body)
	if string(raw) != v {
		t.Error("Incorrect response:", string(raw))
	}

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("DELETE", uri, nil)
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Error("Non-200 health status in DELETE:", w3.Code)
	}

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", uri, nil)
	router.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Error("Non-200 health status in GET (for key not found):", w4.Code)
	}

	raw2, _ := ioutil.ReadAll(w2.Body)
	if string(raw2) != "" {
		t.Error("Incorrect response:", string(raw))
	}
}
