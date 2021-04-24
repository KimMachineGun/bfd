package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/KimMachineGun/bloomfilterd/internal/fsm"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type Config struct {
	N         uint64
	P         float64
	Retention uint64

	NodeID           string
	Addr             string
	TransportMaxPool int           // 5
	TransportTimeout time.Duration // 10s
	Dir              string
	SnapshotRetain   int // 5

	ApplyTimeout time.Duration // 10s
}

type Store struct {
	config *Config
	fsm    *fsm.FSM

	raft          *raft.Raft
	raftConfig    *raft.Config
	transport     *raft.NetworkTransport
	snapshotStore *raft.FileSnapshotStore
	boltStore     *raftboltdb.BoltStore
}

func New(c *Config) (*Store, error) {
	bf := fsm.New(c.N, c.P, c.Retention)

	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(c.NodeID)

	// raft transport
	addr, err := net.ResolveTCPAddr("tcp", c.Addr)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve tcp addr: %v", err)
	}
	transport, err := raft.NewTCPTransport(c.Addr, addr, c.TransportMaxPool, c.TransportTimeout, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("cannot create tcp transport: %v", err)
	}

	// raft snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(c.Dir, c.SnapshotRetain, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("cannot create file snapshot store: %v", err)
	}

	// raft log & stable store
	boltStore, err := raftboltdb.NewBoltStore(filepath.Join(c.Dir, "raft.db"))
	if err != nil {
		return nil, fmt.Errorf("cannot create bolt store: %v", err)
	}

	// create raft
	ra, err := raft.NewRaft(raftConfig, bf, boltStore, boltStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("cannot create raft: %v", err)
	}

	return &Store{
		config: c,
		fsm:    bf,

		raft:          ra,
		raftConfig:    raftConfig,
		transport:     transport,
		snapshotStore: snapshotStore,
		boltStore:     boltStore,
	}, nil
}

func (s *Store) Bootstrap() error {
	err := s.raft.BootstrapCluster(raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      s.raftConfig.LocalID,
				Address: s.transport.LocalAddr(),
			},
		},
	}).Error()
	if err != nil {
		return fmt.Errorf("cannot bootstrap cluster: %v", err)
	}

	return nil
}

func (s *Store) Join(nodeID string, addr string) error {
	configFuture := s.raft.GetConfiguration()
	err := configFuture.Error()
	if err != nil {
		return fmt.Errorf("cannnot get raft configuration: %v", err)
	}

	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			return fmt.Errorf("given node already member of cluster")
		}
	}

	indexFuture := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	err = indexFuture.Error()
	if err != nil {
		return fmt.Errorf("cannot add voter: %v", err)
	}

	return nil
}

func (s *Store) Snapshot() error {
	err := s.raft.Snapshot().Error()
	if err != nil {
		return fmt.Errorf("cannot snapshot manually: %v", err)
	}

	return nil
}

func (s *Store) Check(key string) (bool, error) {
	return s.fsm.Check(key), nil
}

func (s *Store) Set(key string) (bool, error) {
	if s.raft.State() != raft.Leader {
		return false, errors.New("this node is not a leader")
	}

	msg := fsm.Message{
		Op:   fsm.OpSet,
		Args: []string{key},
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return false, fmt.Errorf("cannot marshal message: %v", err)
	}

	f := s.raft.Apply(b, s.config.ApplyTimeout)
	err = f.Error()
	if err != nil {
		return false, fmt.Errorf("cannot marshal message: %v", err)
	}

	return f.Response().(bool), nil
}
