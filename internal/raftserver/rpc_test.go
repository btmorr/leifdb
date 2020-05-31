// Unit tests on rpc functionality

package raftserver

import (
	"context"
	"io/ioutil"
	"log"
	"testing"

	db "github.com/btmorr/leifdb/internal/database"
	. "github.com/btmorr/leifdb/internal/node"
	"github.com/btmorr/leifdb/internal/raft"
	"github.com/btmorr/leifdb/internal/util"
	"github.com/golang/protobuf/proto"
)

func CompareLogs(t *testing.T, testName string, got *raft.LogStore, expected *raft.LogStore) {
	// Note: reflect.DeepEqual failed to return true for `LogRecord`s with
	// identical contents, so have to do this instead... (DeepEqual probably
	// can't reliably traverse objects with arrays of pointers to objects)

	lengthG := len(got.Entries)
	lengthE := len(expected.Entries)
	if lengthG != lengthE {
		t.Errorf(
			"[%s] Expected %d log entries roundtrip but got %d\n",
			testName,
			lengthE,
			lengthG)
	} else {
		for idx, entry := range got.Entries {
			if entry.Term != expected.Entries[idx].Term {
				t.Errorf(
					"[%s] Expected term %d but got %d\n",
					testName,
					expected.Entries[idx].Term,
					entry.Term)
			}
			if entry.Key != expected.Entries[idx].Key {
				t.Errorf(
					"[%s] Expected key %s but got %s\n",
					testName,
					expected.Entries[idx].Key,
					entry.Key)
			}
			if entry.Value != expected.Entries[idx].Value {
				t.Errorf(
					"[%s] Expected value %s but got %s\n",
					testName,
					expected.Entries[idx].Value,
					entry.Value)
			}
		}
	}

}

type AppendTestCase struct {
	name            string
	sendTerm        int64
	sendId          string
	sendPrevIdx     int64
	sendPrevTerm    int64
	sendCommit      int64
	sendRecords     []*raft.LogRecord
	expectedSuccess bool
	expectedStore   *raft.LogStore
	expectedDb      map[string]string
}

func TestAppend(t *testing.T) {
	log.Println("~~~ TestAppend")

	addr := "localhost:8080"
	testDir, _ := util.CreateTmpDir(".tmp-leifdb")
	t.Cleanup(func() {
		util.RemoveTmpDir(testDir)
	})

	config := NewNodeConfig(testDir, addr)

	termRecord := &raft.TermRecord{Term: 5, VotedFor: "localhost:8181"}
	WriteTerm(config.TermFile, termRecord)

	starterLog := &raft.LogStore{
		Entries: []*raft.LogRecord{
			{
				Term:   1,
				Action: raft.LogRecord_SET,
				Key:    "Harry",
				Value:  "present"},
			{
				Term:   2,
				Action: raft.LogRecord_SET,
				Key:    "Ron",
				Value:  "absent"},
			{
				Term:   5,
				Action: raft.LogRecord_SET,
				Key:    "Hermione",
				Value:  "present"}}}

	out, err := proto.Marshal(starterLog)
	if err != nil {
		log.Fatalln("Failed to encode logs:", err)
	}
	if err := ioutil.WriteFile(config.LogFile, out, 0644); err != nil {
		log.Fatalln("Failed to write log file:", err)
	}

	validLeaderId := "localhost:8181"
	invalidLeaderId := "localhost:12345"
	prevIdx := int64(len(starterLog.Entries) - 1)
	prevTerm := starterLog.Entries[prevIdx].Term
	newRecord := &raft.LogRecord{
		Term:   termRecord.Term,
		Action: raft.LogRecord_SET,
		Key:    "Ginny",
		Value:  "adventuring"}
	updatedLog := &raft.LogStore{Entries: append(starterLog.Entries, newRecord)}

	// todo: construct test case that drops some uncommitted entries
	// todo: construct test case that includes a delete action
	testCases := []AppendTestCase{
		{
			name:            "Expired term",
			sendTerm:        termRecord.Term - 1,
			sendId:          validLeaderId,
			sendPrevIdx:     0,
			sendPrevTerm:    0,
			sendCommit:      0,
			sendRecords:     make([]*raft.LogRecord, 0, 0),
			expectedSuccess: false,
			expectedStore:   starterLog,
			expectedDb:      make(map[string]string)},
		{
			name:            "Invalid leader",
			sendTerm:        termRecord.Term,
			sendId:          invalidLeaderId,
			sendPrevIdx:     0,
			sendPrevTerm:    0,
			sendCommit:      2,
			sendRecords:     make([]*raft.LogRecord, 0, 0),
			expectedSuccess: false,
			expectedStore:   starterLog,
			expectedDb:      make(map[string]string)},
		{
			name:            "Empty valid request",
			sendTerm:        termRecord.Term,
			sendId:          validLeaderId,
			sendPrevIdx:     prevIdx,
			sendPrevTerm:    prevTerm,
			sendCommit:      0,
			sendRecords:     make([]*raft.LogRecord, 0, 0),
			expectedSuccess: true,
			expectedStore:   starterLog,
			expectedDb:      make(map[string]string)},
		{
			name:            "New record",
			sendTerm:        termRecord.Term,
			sendId:          validLeaderId,
			sendPrevIdx:     prevIdx,
			sendPrevTerm:    prevTerm,
			sendCommit:      0,
			sendRecords:     []*raft.LogRecord{newRecord},
			expectedSuccess: true,
			expectedStore:   updatedLog,
			expectedDb:      make(map[string]string)},
		{
			name:            "Commit some logs",
			sendTerm:        termRecord.Term,
			sendId:          validLeaderId,
			sendPrevIdx:     prevIdx,
			sendPrevTerm:    prevTerm,
			sendCommit:      2,
			sendRecords:     make([]*raft.LogRecord, 0, 0),
			expectedSuccess: true,
			expectedStore:   updatedLog,
			expectedDb: map[string]string{
				"Harry":    "present",
				"Ron":      "absent",
				"Hermione": "",
				"Ginny":    ""}},
		{
			name:            "Commit all logs",
			sendTerm:        termRecord.Term,
			sendId:          validLeaderId,
			sendPrevIdx:     prevIdx,
			sendPrevTerm:    prevTerm,
			sendCommit:      int64(len(updatedLog.Entries)),
			sendRecords:     make([]*raft.LogRecord, 0, 0),
			expectedSuccess: true,
			expectedStore:   updatedLog,
			expectedDb: map[string]string{
				"Harry":    "present",
				"Ron":      "absent",
				"Hermione": "present",
				"Ginny":    "adventuring"}},
	}

	store := db.NewDatabase()
	node, _ := NewNode(config, store)
	s := server{Node: node}

	for _, tc := range testCases {
		req := &raft.AppendRequest{
			Term:         tc.sendTerm,
			LeaderId:     tc.sendId,
			PrevLogIndex: tc.sendPrevIdx,
			PrevLogTerm:  tc.sendPrevTerm,
			Entries:      tc.sendRecords,
			LeaderCommit: tc.sendCommit}
		reply, err := s.AppendLogs(context.Background(), req)
		if err != nil {
			t.Errorf("[%s] Unexpected error: %v", tc.name, err)
		}
		// Check for expected success
		if reply.Success != tc.expectedSuccess {
			t.Errorf("[%s] Expected success %t but got %t",
				tc.name,
				tc.expectedSuccess,
				reply.Success)
		}
		// Ensure node logs are as expected
		CompareLogs(t, tc.name, node.Log, tc.expectedStore)
		// Ensure database state is as expected
		for k := range tc.expectedDb {
			expect := tc.expectedDb[k]
			if store.Get(k) != expect {
				t.Errorf("[%s] Expected %s to be \"%s\"", tc.name, k, expect)
			}
		}
	}
}
