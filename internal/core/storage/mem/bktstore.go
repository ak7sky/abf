package mem

import (
	"sync"

	"github.com/ak7sky/abf-service/internal/core/model"
)

type BucketMemStorage struct {
	buckets map[string]*model.Bucket
	mtx     *sync.RWMutex
}

func NewBktMemStorage() *BucketMemStorage {
	return &BucketMemStorage{
		buckets: map[string]*model.Bucket{},
		mtx:     &sync.RWMutex{},
	}
}

func (storage *BucketMemStorage) Get(id string) (*model.Bucket, error) {
	storage.mtx.RLock()
	defer storage.mtx.RUnlock()
	return storage.buckets[id], nil
}

func (storage *BucketMemStorage) Save(bucket *model.Bucket) error {
	storage.mtx.Lock()
	storage.buckets[bucket.ID] = bucket
	storage.mtx.Unlock()
	return nil
}

func (storage *BucketMemStorage) Delete(id string) error {
	storage.mtx.Lock()
	delete(storage.buckets, id)
	storage.mtx.Unlock()
	return nil
}
