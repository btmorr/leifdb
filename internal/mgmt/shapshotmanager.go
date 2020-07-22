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

func cloneAndSerialize(node *node.Node) ([]byte, error) {
	node.Lock()
	commitIndex := node.CommitIndex
	clone := db.Clone(node.Store)
	node.Unlock()

	log.Info().
		Int64("commit index", commitIndex).
		Msg("doing snapshot")
	return db.BuildSnapshot(clone)
}

func persistSnapshot(data []byte, filename string) error {
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

func StartSnapshotManager(
	dataDir string,
	logFile string,
	threshold int64,
	retain int,
	node *node.Node) {
	t := time.NewTicker(time.Minute)

	snapshotFiles, nextIndex := findExistingSnapshots(dataDir)

	go func() {
		for {
			select {
			case <-t.C:
				// check file size
				fi, _ := os.Stat(logFile)
				size := fi.Size()
				log.Info().
					Int64("log file size", size).
					Int64("threshold", threshold).
					Msg("snapshot check")
				if size > threshold {
					snapshot, err := cloneAndSerialize(node)
					if err != nil {
						log.Error().Err(err).Msg("error building snapshot")
						continue
					}

					filename := fmt.Sprintf("%s%06d", prefix, nextIndex)
					fullPath := filepath.Join(dataDir, filename)
					err = persistSnapshot(snapshot, fullPath)
					if err != nil {
						log.Error().Err(err).Msg("error persisting snapshot")
						continue
					}

					nextIndex++
					snapshotFiles = append(snapshotFiles, fullPath)
					// todo: trigger node log compaction
				}
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
			default:
			}
		}
	}()
}
