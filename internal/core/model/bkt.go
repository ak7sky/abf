package model

import (
	"sync"
	"time"
)

type BktType string

const (
	LoginBkt = "login"
	PswdBkt  = "password"
	IpBkt    = "ip"
)

type Bucket struct {
	ID        string
	Capacity  uint
	FreeSpace uint
	ResetTime time.Time
	Duration  time.Duration
	mtx       *sync.Mutex
}

func NewBucket(id string, capacity uint, duration time.Duration) *Bucket {
	return &Bucket{
		ID:        id,
		Capacity:  capacity,
		FreeSpace: capacity,
		ResetTime: time.Now().Add(duration),
		Duration:  duration,
		mtx:       &sync.Mutex{},
	}
}

func (bucket *Bucket) Add() bool {
	bucket.mtx.Lock()
	defer bucket.mtx.Unlock()

	if time.Now().After(bucket.ResetTime) {
		bucket.ResetTime = time.Now().Add(bucket.Duration)
		bucket.FreeSpace = bucket.Capacity
	}

	if bucket.FreeSpace < 1 {
		return false
	}

	bucket.FreeSpace -= 1
	return true
}

func (bucket *Bucket) Reset() {
	bucket.mtx.Lock()
	bucket.FreeSpace = bucket.Capacity
	bucket.ResetTime = time.Now().Add(bucket.Duration)
	bucket.mtx.Unlock()
}
