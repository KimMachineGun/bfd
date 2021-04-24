package fsm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/KimMachineGun/bloomfilterd/internal/bloomfilter"

	"github.com/hashicorp/raft"
)

type FSM struct {
	mu   *sync.RWMutex
	twbf *bloomfilter.TermWindowedBloomFilter
}

func New(n uint64, p float64, retention uint64) *FSM {
	return &FSM{
		mu:   &sync.RWMutex{},
		twbf: bloomfilter.NewTermWindowedBloomFilter(n, p, retention),
	}
}

func (m *FSM) Apply(l *raft.Log) interface{} {
	var msg Message
	err := json.Unmarshal(l.Data, &msg)
	if err != nil {
		return fmt.Errorf("cannot unmarshal message: %v", err)
	}

	switch msg.Op {
	case OpSet:
		return m.opSet(msg)
	case OpCheck:
		return m.opCheck(msg)
	}

	return fmt.Errorf("unknown operation: %v", msg.Op)
}

func (m *FSM) Snapshot() (raft.FSMSnapshot, error) {
	var (
		tbfs             = m.twbf.TBFS()
		earliest, latest = m.twbf.Terms()
		curr             = bytes.NewBuffer(nil)
	)
	for _, tbf := range tbfs {
		if tbf.Term() < latest {
			err := saveSnapshot(tbf)
			if err != nil {
				return nil, err
			}
		} else if tbf.Term() == latest {
			err := tbf.Snapshot(curr)
			if err != nil {
				return nil, err
			}
		}
	}

	return &Snapshot{
		header: SnapshotHeader{
			Latest:   latest,
			Earliest: earliest,
		},
		b: curr.Bytes(),
	}, nil
}

func (m *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	br := bufio.NewReader(rc)

	b, err := br.ReadBytes('\n')
	if err != nil {
		return err
	}
	_, err = br.Discard(1)
	if err != nil {
		return err
	}

	var header SnapshotHeader
	err = json.Unmarshal(b, &header)
	if err != nil {
		return err
	}

	m.twbf.SetTerms(header.Earliest, header.Latest)

	tbfs := m.twbf.TBFS()
	for i := header.Earliest; i <= header.Latest; i++ {
		tbf := tbfs[i%m.twbf.Retention()]
		if i < header.Latest {
			err = restoreSnapshot(i, tbf)
			if err != nil {
				return err
			}
		} else if i == header.Latest {
			err = tbf.Restore(br)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *FSM) Check(key string) bool {
	return m.twbf.Check([]byte(key))
}

func (m *FSM) opSet(msg Message) interface{} {
	if len(msg.Args) != 1 {
		return errors.New("invalid arguments")
	}

	return m.twbf.Set([]byte(msg.Args[0]))
}

func (m *FSM) opCheck(msg Message) interface{} {
	if len(msg.Args) != 1 {
		return errors.New("invalid arguments")
	}

	return m.twbf.Check([]byte(msg.Args[0]))
}
