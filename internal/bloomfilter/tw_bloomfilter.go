package bloomfilter

import (
	"hash"
	"sync"
)

type TermWindowedBloomFilter struct {
	n         uint64  // number of items
	p         float64 // probability of false positives
	m         uint64  // number of bits
	k         uint64  // number of hash functions
	retention uint64

	// mu     *sync.RWMutex
	// prevMu *sync.RWMutex
	//
	pool    *sync.Pool
	hashers []hash.Hash64

	tbfs []*TBF
	term struct {
		earliest uint64
		latest   uint64
	}
}

func NewTermWindowedBloomFilter(n uint64, p float64, retention uint64) *TermWindowedBloomFilter {
	m, k := getMKFromNP(n, p)

	tbfs := make([]*TBF, retention)
	for i := range tbfs {
		tbfs[i] = &TBF{
			term: uint64(i),
			cap:  n,
			bf:   newBF(m),
		}
	}

	return &TermWindowedBloomFilter{
		n:         n,
		p:         p,
		m:         m,
		k:         k,
		retention: retention,
		//
		// mu:     &sync.RWMutex{},
		// prevMu: &sync.RWMutex{},
		//
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]uint64, k)
			},
		},
		hashers: getMurmur3Hashers(k),

		tbfs: tbfs,
	}
}

func (f *TermWindowedBloomFilter) N() uint64 {
	return f.n
}

func (f *TermWindowedBloomFilter) P() float64 {
	return f.p
}

func (f *TermWindowedBloomFilter) M() uint64 {
	return f.m
}

func (f *TermWindowedBloomFilter) K() uint64 {
	return f.k
}

func (f *TermWindowedBloomFilter) Retention() uint64 {
	return f.retention
}

func (f *TermWindowedBloomFilter) TBFS() []*TBF {
	return f.tbfs
}

func (f *TermWindowedBloomFilter) Terms() (uint64, uint64) {
	return f.term.earliest, f.term.latest
}

func (f *TermWindowedBloomFilter) SetTerms(earliest, latest uint64) {
	f.term.earliest = earliest
	f.term.latest = latest
}

func (f *TermWindowedBloomFilter) Set(key []byte) bool {
	return f.set(key)
}

func (f *TermWindowedBloomFilter) set(key []byte) bool {
	hashes := f.generateHashes(key)
	defer f.releaseHashes(hashes)

	// f.prevMu.RLock()
	for i := f.term.earliest; i < f.term.latest; i++ {
		tbf := f.tbf(i)
		if tbf.term == i && tbf.bf.check(hashes) {
			// f.prevMu.RUnlock()
			return true
		}
	}
	// f.prevMu.RUnlock()

	// f.mu.Lock()
	tbf := f.tbf(f.term.latest)
	exists := tbf.bf.set(hashes)
	if !exists {
		tbf.cap--
		if tbf.cap == 0 {
			// f.prevMu.Lock()
			f.increaseTerm()
			// f.prevMu.Unlock()
		}
	}
	// f.mu.Unlock()

	return exists
}

func (f *TermWindowedBloomFilter) Check(key []byte) bool {
	return f.check(key)
}

func (f *TermWindowedBloomFilter) check(key []byte) bool {
	hashes := f.generateHashes(key)
	defer f.releaseHashes(hashes)

	// f.prevMu.RLock()
	for i := f.term.earliest; i < f.term.latest; i++ {
		tbf := f.tbf(i)
		if tbf.term == i && tbf.bf.check(hashes) {
			// f.prevMu.RUnlock()
			return true
		}
	}
	// f.prevMu.RUnlock()

	// f.mu.RLock()
	exists := f.tbf(f.term.latest).bf.check(hashes)
	// f.mu.RUnlock()

	return exists
}

func (f *TermWindowedBloomFilter) increaseTerm() {
	_ = saveSnapshot(f.tbf(f.term.latest))

	f.term.latest++
	if (f.retention - 1) <= f.term.latest {
		f.term.earliest = f.term.latest - (f.retention - 1)
	}

	tbf := f.tbf(f.term.latest)
	tbf.term = f.term.latest
	tbf.bf.reset()
	tbf.cap = f.n
}

func (f *TermWindowedBloomFilter) tbf(term uint64) *TBF {
	return f.tbfs[term%f.retention]
}

func (f *TermWindowedBloomFilter) generateHashes(key []byte) []uint64 {
	hashes := f.pool.Get().([]uint64)
	for i, h := range f.hashers {
		h.Write(key)
		hashes[i] = h.Sum64()
		h.Reset()
	}

	return hashes
}

func (f *TermWindowedBloomFilter) releaseHashes(hashes []uint64) {
	f.pool.Put(hashes)
}
