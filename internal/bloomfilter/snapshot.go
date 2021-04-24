package bloomfilter

import (
	"errors"
	"fmt"
	"os"
	"path"
)

const snapshotBase = "/Users/kimmachinegun/Desktop/Programming/bloomfilterd/snapshots"

func snapshotFileName(term uint64) string {
	return path.Join(snapshotBase, fmt.Sprintf("term_%d.snapshot", term))
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func snapshotExists(term uint64) bool {
	info, err := os.Stat(snapshotFileName(term))
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func saveSnapshot(tbf *TBF) error {
	if snapshotExists(tbf.Term()) {
		return nil
	}

	f, err := os.Create(snapshotFileName(tbf.Term()))
	if err != nil {
		return err
	}
	defer f.Close()

	return tbf.Snapshot(f)
}

func restoreSnapshot(term uint64, tbf *TBF) error {
	if !snapshotExists(term) {
		return errors.New("snapshot not exists")
	}

	f, err := os.Open(snapshotFileName(term))
	if err != nil {
		return err
	}
	defer f.Close()

	return tbf.Restore(f)
}
