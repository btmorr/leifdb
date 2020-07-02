// +build unit

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/mgmt"
	"github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/btmorr/leifdb/internal/util"
	"github.com/gin-gonic/gin"
)

// setupServer configures a Database, Node, and router for test, and creates
// a test directory that is automatically cleaned up after each test
func setupServer(t *testing.T) (*gin.Engine, *node.Node) {
	addr := "localhost:16990"
	clientAddr := "localhost:8080"

	testDir, err := util.CreateTmpDir(".tmp-leifdb")
	if err != nil {
		log.Fatalln("Error creating test dir:", err)
	}
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	store := db.NewDatabase()

	config := node.NewNodeConfig(testDir, addr, clientAddr, make([]string, 0, 0))
	n, _ := node.NewNode(config, store)
	router := buildRouter(n)
	n.State = mgmt.Leader
	return router, n
}

func TestHealthRoute(t *testing.T) {
	router, _ := setupServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Non-200 health status")
	}
}

func TestReadAfterWrite(t *testing.T) {
	router, _ := setupServer(t)

	v := "testy"
	body1 := WriteRequest{Value: v}
	b1, _ := json.Marshal(body1)
	br1 := bytes.NewReader(b1)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/db/stuff", br1)
	router.ServeHTTP(w, req)

	fmt.Printf("Response: %+v\n", w)

	if w.Code != http.StatusOK {
		t.Error("Non-200 health status in PUT:", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/db/stuff", nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Error("Non-200 health status in GET:", w2.Code)
	}

	raw, _ := ioutil.ReadAll(w2.Body)
	var data ReadResponse
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Error(err.Error())
	}
	if data.Value != v {
		t.Errorf("Incorrect response: %+v", data)
	}
}

func TestWriteRedirect(t *testing.T) {
	router, node := setupServer(t)

	// Change our node so we become a follower
	node.State = mgmt.Follower
	node.SetTerm(node.Term+1, &raft.Node{
		Id:         "localhost:16991",
		ClientAddr: "localhost:8081",
	})

	v := "testy"
	body1 := WriteRequest{Value: v}
	b1, _ := json.Marshal(body1)
	br1 := bytes.NewReader(b1)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/db/stuff", br1)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected a temporary (307) redirect but got: %d\n", w.Code)
	}

	location := w.Header().Get("Location")
	expected := "http://localhost:8081/db/stuff"
	if location != expected {
		t.Errorf("Expected to be redirected to %s but got %s\n", expected, location)
	}
}

func TestDelete(t *testing.T) {
	router, _ := setupServer(t)

	v := "testy"
	body1 := WriteRequest{Value: v}
	b1, _ := json.Marshal(body1)
	br1 := bytes.NewReader(b1)

	uri := "/db/stuff"

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", uri, br1)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Non-200 health status in PUT:", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", uri, nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Error("Non-200 health status in GET:", w2.Code)
	}

	raw, _ := ioutil.ReadAll(w2.Body)
	var data ReadResponse
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Error(err.Error())
	}
	if data.Value != v {
		t.Errorf("Incorrect response: %+v", data)
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

	raw4, _ := ioutil.ReadAll(w4.Body)
	var data4 ReadResponse
	if err := json.Unmarshal(raw4, &data4); err != nil {
		t.Errorf("%s while processing %s", err.Error(), string(raw4))
	}
	if data4.Value != "" {
		t.Errorf("Incorrect response: %+v", data4)
	}
}

func TestDeleteRedirect(t *testing.T) {
	router, node := setupServer(t)

	// Change our node so we become a follower
	node.State = mgmt.Follower
	node.SetTerm(node.Term+1, &raft.Node{
		Id:         "localhost:16991",
		ClientAddr: "localhost:8081",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/db/stuff", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected a temporary (307) redirect but got: %d\n", w.Code)
	}

	location := w.Header().Get("Location")
	expected := "http://localhost:8081/db/stuff"
	if location != expected {
		t.Errorf("Expected to be redirected to %s but got %s\n", expected, location)
	}
}
