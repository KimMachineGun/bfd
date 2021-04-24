package bloomfilter

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	a := assert.New(t)

	twbf := NewTermWindowedBloomFilter(100, 0.00000000001, 3)
	a.NotNil(twbf)
	earliest, latest := twbf.Terms()
	a.Equal(uint64(0), earliest)
	a.Equal(uint64(0), latest)

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("0_%d", i))
		a.False(twbf.Check(key))
		a.False(twbf.Set(key))
		a.True(twbf.Check(key))
	}
	earliest, latest = twbf.Terms()
	a.Equal(uint64(0), earliest)
	a.Equal(uint64(1), latest)

	b := bytes.NewBuffer(nil)
	tbf := twbf.tbf(0)

	err := tbf.Snapshot(b)
	a.NoError(err)

	nTbf := &TBF{
		bf: newBF(tbf.bf.m),
	}
	err = nTbf.Restore(b)
	a.NoError(err)

	a.Equal(tbf.term, nTbf.term)
	a.Equal(tbf.bf.m, nTbf.bf.m)
	a.Equal(tbf.bf.b, nTbf.bf.b)
	a.Equal(tbf.cap, nTbf.cap)

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("1_%d", i))
		a.False(twbf.Check(key))
		a.False(twbf.Set(key))
		a.True(twbf.Check(key))
	}
	earliest, latest = twbf.Terms()
	a.Equal(uint64(0), earliest)
	a.Equal(uint64(2), latest)

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("2_%d", i))
		a.False(twbf.Check(key))
		a.False(twbf.Set(key))
		a.True(twbf.Check(key))
	}
	earliest, latest = twbf.Terms()
	a.Equal(uint64(1), earliest)
	a.Equal(uint64(3), latest)

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("3_%d", i))
		a.False(twbf.Check(key))
		a.False(twbf.Set(key))
		a.True(twbf.Check(key))
	}
	earliest, latest = twbf.Terms()
	a.Equal(uint64(2), earliest)
	a.Equal(uint64(4), latest)

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("0_%d", i))
		a.False(twbf.Check(key))
	}

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("1_%d", i))
		a.False(twbf.Check(key))
	}

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("2_%d", i))
		a.True(twbf.Check(key))
	}

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("3_%d", i))
		a.True(twbf.Check(key))
	}

	for i := 0; i < 99; i++ {
		key := []byte(fmt.Sprintf("0_%d", i))
		a.False(twbf.Check(key))
		a.False(twbf.Set(key))
		a.True(twbf.Check(key))
	}
	earliest, latest = twbf.Terms()
	a.Equal(uint64(2), earliest)
	a.Equal(uint64(4), latest)

	for i := 0; i < 99; i++ {
		key := []byte(fmt.Sprintf("0_%d", i))
		a.True(twbf.Check(key))
	}

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("1_%d", i))
		a.False(twbf.Check(key))
	}

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("2_%d", i))
		a.True(twbf.Check(key))
	}

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("3_%d", i))
		a.True(twbf.Check(key))
	}

	key := []byte("0_99")
	a.False(twbf.Check(key))
	a.False(twbf.Set(key))
	a.True(twbf.Check(key))
	earliest, latest = twbf.Terms()
	a.Equal(uint64(3), earliest)
	a.Equal(uint64(5), latest)

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("0_%d", i))
		a.True(twbf.Check(key))
	}

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("1_%d", i))
		a.False(twbf.Check(key))
	}

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("2_%d", i))
		a.False(twbf.Check(key))
	}

	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("3_%d", i))
		a.True(twbf.Check(key))
	}
}

func Benchmark(b *testing.B) {
	a := assert.New(b)

	twbf := NewTermWindowedBloomFilter(10000, 0.00000000001, 5)
	a.NotNil(twbf)
	earliest, latest := twbf.Terms()
	a.Equal(uint64(0), earliest)
	a.Equal(uint64(0), latest)

	for i := 0; i < 10000000; i++ {
		key := []byte(fmt.Sprintf("0_%d", i))
		// a.False(twbf.Check(key))
		a.False(twbf.Set(key))
		// a.True(twbf.Check(key))
	}
}
