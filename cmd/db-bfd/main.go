package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/KimMachineGun/flago"
	"github.com/lni/dragonboat/v3"
	"github.com/lni/dragonboat/v3/config"
	"github.com/lni/dragonboat/v3/logger"
	"github.com/lni/goutils/syncutil"
)

const (
	clusterID uint64 = 128
)

var (
	addresses = []string{
		"localhost:63001",
		"localhost:63002",
		"localhost:63003",
	}
)

func init() {
	logger.GetLogger("raft").SetLevel(logger.ERROR)
	logger.GetLogger("rsm").SetLevel(logger.WARNING)
	logger.GetLogger("transport").SetLevel(logger.WARNING)
	logger.GetLogger("grpc").SetLevel(logger.WARNING)
}

func main() {
	var c Config
	flago.Bind(flag.CommandLine, &c)
	flag.Parse()

	err := c.Validate()
	if err != nil {
		log.Fatalln(err)
	}

	// https://github.com/golang/go/issues/17393
	if runtime.GOOS == "darwin" {
		signal.Ignore(syscall.Signal(0xd))
	}

	initialMembers := map[uint64]string{}
	if c.JoinAddr != "" {
		for i, addr := range c.InitialMembers {
			initialMembers[uint64(i)+1] = addr
		}
	}

	rc := config.Config{
		NodeID:             c.NodeID,
		ClusterID:          clusterID,
		ElectionRTT:        10,
		HeartbeatRTT:       1,
		CheckQuorum:        true,
		SnapshotEntries:    10,
		CompactionOverhead: 5,
	}
	datadir := filepath.Join(
		"example-data",
		"helloworld-data",
		fmt.Sprintf("node%d", c.NodeID))
	nhc := config.NodeHostConfig{
		WALDir:         datadir,
		NodeHostDir:    datadir,
		RTTMillisecond: 200,
		RaftAddress:    c.Addr,
	}
	nh, err := dragonboat.NewNodeHost(nhc)
	if err != nil {
		panic(err)
	}
	if err := nh.StartCluster(initialMembers, c.JoinAddr != "", NewExampleStateMachine, rc); err != nil {
		fmt.Fprintf(os.Stderr, "failed to add cluster, %v\n", err)
		os.Exit(1)
	}
	raftStopper := syncutil.NewStopper()
	consoleStopper := syncutil.NewStopper()
	ch := make(chan string, 16)
	consoleStopper.RunWorker(func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				close(ch)
				return
			}
			if s == "exit\n" {
				raftStopper.Stop()
				// no data will be lost/corrupted if nodehost.Stop() is not called
				nh.Stop()
				return
			}
			ch <- s
		}
	})
	raftStopper.RunWorker(func() {
		// this goroutine makes a linearizable read every 10 second. it returns the
		// Count value maintained in IStateMachine. see datastore.go for details.
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				result, err := nh.SyncRead(ctx, clusterID, []byte{})
				cancel()
				if err == nil {
					var count uint64
					count = binary.LittleEndian.Uint64(result.([]byte))
					fmt.Fprintf(os.Stdout, "count: %d\n", count)
				}
			case <-raftStopper.ShouldStop():
				return
			}
		}
	})
	raftStopper.Wait()
}
