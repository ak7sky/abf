package mem

import (
	"abf-service/internal/core/model"
	"encoding/binary"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
)

type netMemStorageTestSuit struct {
	suite.Suite
	blk           []*model.Net
	wht           []*model.Net
	netMemStorage *NetMemStorage
}

func (suite *netMemStorageTestSuit) SetupSuite() {
	suite.blk = []*model.Net{
		{Addr: binary.BigEndian.Uint32([]byte{192, 168, 0, 0}), MaskLen: 16},
		{Addr: binary.BigEndian.Uint32([]byte{192, 169, 0, 0}), MaskLen: 16},
		{Addr: binary.BigEndian.Uint32([]byte{192, 170, 0, 0}), MaskLen: 16},
	}
	suite.wht = []*model.Net{
		{Addr: binary.BigEndian.Uint32([]byte{192, 171, 0, 0}), MaskLen: 16},
		{Addr: binary.BigEndian.Uint32([]byte{192, 172, 0, 0}), MaskLen: 16},
		{Addr: binary.BigEndian.Uint32([]byte{192, 173, 0, 0}), MaskLen: 16},
	}
}

func (suite *netMemStorageTestSuit) SetupTest() {
	netStorage := NewNetMemStorage()
	for _, net := range suite.blk {
		netStorage.black[net.Addr] = net
	}
	for _, net := range suite.wht {
		netStorage.white[net.Addr] = net
	}
	suite.netMemStorage = netStorage
}

func (suite *netMemStorageTestSuit) TestSave() {
	storageConsumers := 10
	wg := sync.WaitGroup{}

	for i := 0; i < storageConsumers; i++ {
		i := i
		wg.Add(1)
		go func() {
			addr := binary.BigEndian.Uint32([]byte{193, byte(174 + i), 0, 0})
			net := &model.Net{Addr: addr, MaskLen: 16}
			err := suite.netMemStorage.Save(net, model.Black)
			require.NoError(suite.T(), err)
			wg.Done()
		}()
	}

	wg.Wait()
	require.Equal(suite.T(), len(suite.blk)+storageConsumers, len(suite.netMemStorage.black))
	require.Equal(suite.T(), len(suite.wht), len(suite.netMemStorage.white))
}

func (suite *netMemStorageTestSuit) TestGet() {
	wg := sync.WaitGroup{}

	for _, blkNet := range suite.blk {
		blkNet := blkNet
		wg.Add(1)
		go func() {
			foundInBlk, err := suite.netMemStorage.Get(blkNet.Addr, model.Black)
			require.NoError(suite.T(), err)
			require.NotNil(suite.T(), foundInBlk)
			require.Equal(suite.T(), blkNet.Addr, foundInBlk.Addr)

			notFoundInWh, err := suite.netMemStorage.Get(blkNet.Addr, model.White)
			require.NoError(suite.T(), err)
			require.Nil(suite.T(), notFoundInWh)

			wg.Done()
		}()
	}

	wg.Wait()
}

func (suite *netMemStorageTestSuit) TestGetList() {
	storageConsumers := 10
	wg := sync.WaitGroup{}

	for i := 0; i < storageConsumers; i++ {
		wg.Add(1)
		go func() {
			list, err := suite.netMemStorage.GetList(model.Black)
			require.NoError(suite.T(), err)
			require.Equal(suite.T(), len(suite.blk), len(list))

			list, err = suite.netMemStorage.GetList(model.White)
			require.NoError(suite.T(), err)
			require.Equal(suite.T(), len(suite.wht), len(list))

			wg.Done()
		}()
	}

	wg.Wait()
}

func (suite *netMemStorageTestSuit) TestDelete() {
	wg := sync.WaitGroup{}

	for _, whtNet := range suite.wht {
		whtNet := whtNet
		wg.Add(1)
		go func() {
			err := suite.netMemStorage.Delete(whtNet.Addr, whtNet.MaskLen, model.White)
			require.NoError(suite.T(), err)
			wg.Done()
		}()
	}

	wg.Wait()
	require.Equal(suite.T(), 0, len(suite.netMemStorage.white))
}

func TestNetMemStorageTestSuit(t *testing.T) {
	suite.Run(t, new(netMemStorageTestSuit))
}
