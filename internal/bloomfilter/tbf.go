package bloomfilter

import (
	"bufio"
	"encoding/json"
	"io"
)

type TBF struct {
	term uint64
	cap  uint64
	bf   *bf
}

func (f *TBF) Term() uint64 {
	return f.term
}

func (f *TBF) Cap() uint64 {
	return f.cap
}

func (f *TBF) Snapshot(w io.Writer) error {
	err := json.NewEncoder(w).Encode(struct {
		Term uint64
		M    uint64
		Cap  uint64
	}{
		Term: f.term,
		M:    f.bf.m,
		Cap:  f.cap,
	})
	if err != nil {
		return err
	}

	_, err = w.Write([]byte{'\n'})
	if err != nil {
		return err
	}

	_, err = w.Write(f.bf.b)
	if err != nil {
		return err
	}

	return nil
}

func (f *TBF) Restore(r io.Reader) error {
	br := bufio.NewReader(r)

	b, err := br.ReadBytes('\n')
	if err != nil {
		return err
	}
	_, err = br.Discard(1)
	if err != nil {
		return err
	}

	var meta struct {
		Term uint64
		M    uint64
		Cap  uint64
	}
	err = json.Unmarshal(b, &meta)
	if err != nil {
		return err
	}

	f.term = meta.Term
	f.bf.m = meta.M
	f.cap = meta.Cap

	_, err = io.ReadFull(br, f.bf.b)
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}
