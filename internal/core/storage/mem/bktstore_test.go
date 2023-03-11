package mem

import (
	"abf-service/internal/core/model"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

type bktMemStorageTestSuit struct {
	suite.Suite
	buckets       []*model.Bucket
	bktMemStorage *BucketMemStorage
}

func (suite *bktMemStorageTestSuit) SetupSuite() {
	suite.buckets = []*model.Bucket{
		model.NewBucket("login0", 5, time.Minute),
		model.NewBucket("login1", 5, time.Minute),
		model.NewBucket("login2", 5, time.Minute),
	}
}

func (suite *bktMemStorageTestSuit) SetupTest() {
	bktStorage := NewBktMemStorage()
	for _, bkt := range suite.buckets {
		bktStorage.buckets[bkt.ID] = bkt
	}
	suite.bktMemStorage = bktStorage
}

func (suite *bktMemStorageTestSuit) TestSave() {
	storageConsumers := 10
	wg := sync.WaitGroup{}

	for i := len(suite.buckets); i < len(suite.buckets)+storageConsumers; i++ {
		i := i
		wg.Add(1)
		go func() {
			id := fmt.Sprint("login", i)
			bucket := model.NewBucket(id, 5, time.Minute)
			err := suite.bktMemStorage.Save(bucket)
			require.NoError(suite.T(), err)
			wg.Done()
		}()
	}

	wg.Wait()
	require.Equal(suite.T(), len(suite.buckets)+storageConsumers, len(suite.bktMemStorage.buckets))
}

func (suite *bktMemStorageTestSuit) TestGet() {
	wg := sync.WaitGroup{}

	for _, bkt := range suite.buckets {
		bkt := bkt
		wg.Add(1)
		go func() {
			foundBkt, err := suite.bktMemStorage.Get(bkt.ID)
			require.NoError(suite.T(), err)
			require.NotNil(suite.T(), foundBkt)
			require.Equal(suite.T(), bkt.ID, foundBkt.ID)
			wg.Done()
		}()
	}

	wg.Wait()
}

func (suite *bktMemStorageTestSuit) TestDelete() {
	wg := sync.WaitGroup{}

	for _, bkt := range suite.buckets {
		bkt := bkt
		wg.Add(1)
		go func() {
			err := suite.bktMemStorage.Delete(bkt.ID)
			require.NoError(suite.T(), err)
			wg.Done()
		}()
	}

	wg.Wait()
	require.Equal(suite.T(), 0, len(suite.bktMemStorage.buckets))
}

func TestBktMemStorageTestSuit(t *testing.T) {
	suite.Run(t, new(bktMemStorageTestSuit))
}
