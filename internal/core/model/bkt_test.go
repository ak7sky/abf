package model

import (
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestBucket_Add(t *testing.T) {
	bktCap := uint(3)
	bkt := NewBucket("test", bktCap, time.Minute)

	for i := 0; i < int(bktCap); i++ {
		addRes := bkt.Add()
		require.True(t, addRes)
	}

	addRes := bkt.Add()
	require.False(t, addRes)
}

func TestBucket_Reset(t *testing.T) {
	bktCap := uint(3)
	bkt := NewBucket("test", bktCap, time.Minute)

	for i := 0; i < int(bktCap); i++ {
		bkt.Add()
	}

	addRes := bkt.Add()
	require.False(t, addRes)

	bkt.Reset()

	addRes = bkt.Add()
	require.True(t, addRes)
}

func TestBucket_ParallelAdd(t *testing.T) {
	bktCap := uint(3)
	bkt := NewBucket("test", bktCap, time.Minute)

	adders := 100
	wg := sync.WaitGroup{}

	for i := 0; i < adders; i++ {
		wg.Add(1)
		go func() {
			bkt.Add()
			wg.Done()
		}()
	}

	wg.Wait()
	require.Equal(t, 0, int(bkt.FreeSpace))
}
