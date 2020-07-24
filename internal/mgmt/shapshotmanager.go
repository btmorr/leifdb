package mgmt

// snapshot manager can be aware of node and database, but node and database
// should not be aware of managers
import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	db "github.com/btmorr/leifdb/internal/database"
	"github.com/btmorr/leifdb/internal/node"
)

const prefix = "ldbsnapshot"

// findExistingSnapshots returns a lexicographically sorted list of snapshot
// files, along with the next available index number for snapshotting
func findExistingSnapshots(dataDir string) ([]string, int) {
	snapshotFiles := []string{}
	nextIndex := 0
	files, err := ioutil.ReadDir(dataDir)
	if err != nil {
		log.Error().Err(err).Msgf("error listing %s", dataDir)
	} else {
		for _, file := range files {
			if strings.HasPrefix(file.Name(), prefix) {
				snapshotFiles = append(snapshotFiles, filepath.Join(dataDir, file.Name()))
			}
		}
		sort.Strings(snapshotFiles)
	}
	log.Info().Int("count", len(snapshotFiles)).Msg("found snapshots")
	if len(snapshotFiles) > 0 {
		latest := snapshotFiles[len(snapshotFiles)-1]
		_, filename := filepath.Split(latest)
		index, err := strconv.Atoi(strings.TrimPrefix(filename, prefix))
		if err != nil {
			log.Error().Err(err).Msg("error parsing latest snapshot index")
		} else {
			log.Info().Int("index", index).Msg("latest snapshot")
			nextIndex = index + 1
		}
	}
	return snapshotFiles, nextIndex
}

// cloneAndSerialize makes a copy of the current commit index and database
// state, then returns a serialized version of the snapshot with metadata, or
// an error
func cloneAndSerialize(n *node.Node) ([]byte, db.Metadata, error) {
	n.Lock()
	commitIndex := n.CommitIndex
	clone := db.Clone(n.Store)
	n.Unlock()

	lastTerm := n.Log.Entries[commitIndex].Term
	metadata := db.Metadata{LastIndex: commitIndex, LastTerm: lastTerm}
	snapshot, err := db.BuildSnapshot(clone, metadata)

	return snapshot, metadata, err
}

// persist writes a byte array (the serialized snapshot) to disk
func persist(data []byte, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	w.Flush()
	return nil
}

// dropOldSnapshots deletes lexicographically earliest snapshots until the
// length of the list of snapshot files is less than or equal to the retain
// parameter
func dropOldSnapshots(snapshotFiles []string, retain int) []string {
	for len(snapshotFiles) > retain {
		drop := snapshotFiles[0]
		log.Info().Str("filename", drop).Msg("removing snapshot")
		err := os.Remove(drop)
		if err != nil {
			log.Error().
				Err(err).
				Str("filename", drop).
				Msg("error removing log file")
		}
		snapshotFiles = snapshotFiles[1:]
	}
	return snapshotFiles
}

// loadSnapshot fetches a snapshot as a byte array from the file specified,
// initializes a database from the snapshot and installs it into the node
func loadSnapshot(n *node.Node, snapshotPath string) error {
	data, err := ioutil.ReadFile(snapshotPath)
	if err != nil {
		return err
	}

	newStore, metadata, err := db.InstallSnapshot(data)
	if err != nil {
		return err
	}
	n.Store = newStore
	n.IndexOffset = metadata.LastIndex
	n.LastSnapshotTerm = metadata.LastTerm
	return nil
}

func StartSnapshotManager(
	dataDir string,
	logFile string,
	threshold int64,
	period time.Duration,
	retain int,
	n *node.Node) {
	t := time.NewTicker(period)

	snapshotFiles, nextIndex := findExistingSnapshots(dataDir)
	if len(snapshotFiles) > 0 {
		err := loadSnapshot(n, snapshotFiles[len(snapshotFiles)-1])
		if err != nil {
			log.Fatal().Err(err).Msg("error loading snapshot")
		}
	}

	go func() {
		for {
			select {
			case <-t.C:
				fi, err := os.Stat(logFile)
				if err != nil {
					log.Debug().Msg("log file does not exists -- skipping snapshot")
					continue
				}
				size := fi.Size()
				log.Info().
					Int64("log file size", size).
					Int64("threshold", threshold).
					Msg("snapshot check")

				if size > threshold {
					snapshot, metadata, err := cloneAndSerialize(n)
					if err != nil {
						log.Error().Err(err).Msg("error building snapshot")
						continue
					}
					log.Debug().
						Int64("index", metadata.LastIndex).
						Int64("term", metadata.LastTerm).
						Msg("doing snapshot")

					filename := fmt.Sprintf("%s%06d", prefix, nextIndex)
					fullPath := filepath.Join(dataDir, filename)
					err = persist(snapshot, fullPath)
					if err != nil {
						log.Error().Err(err).Msg("error persisting snapshot")
						continue
					}

					nextIndex++
					snapshotFiles = append(snapshotFiles, fullPath)
					err = n.CompactLogs(metadata.LastIndex, metadata.LastTerm)
					if err != nil {
						// should this be fatal? (snapshot exists but logs didn't get compacted)
						log.Error().Err(err).Msg("error compacting logs")
						continue
					}
				}

				snapshotFiles = dropOldSnapshots(snapshotFiles, retain)
			default:
			}
		}
	}()
}
