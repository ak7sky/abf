package mem

import (
	"sync"

	"github.com/ak7sky/abf-service/internal/core/model"
)

type NetMemStorage struct {
	black, white map[uint32]*model.Net
	bmtx, wmtx   *sync.RWMutex
}

func NewNetMemStorage() *NetMemStorage {
	return &NetMemStorage{
		black: map[uint32]*model.Net{},
		white: map[uint32]*model.Net{},
		bmtx:  &sync.RWMutex{},
		wmtx:  &sync.RWMutex{},
	}
}

func (netStorage *NetMemStorage) Save(net *model.Net, netType model.NetType) error {
	nets, mtx := netStorage.netsOfType(netType)
	mtx.Lock()
	nets[net.Addr] = net
	mtx.Unlock()
	return nil
}

func (netStorage *NetMemStorage) Get(addr uint32, netType model.NetType) (*model.Net, error) {
	nets, mtx := netStorage.netsOfType(netType)
	mtx.RLock()
	defer mtx.RUnlock()
	return nets[addr], nil
}

func (netStorage *NetMemStorage) GetList(netType model.NetType) ([]*model.Net, error) {
	nets, mtx := netStorage.netsOfType(netType)
	mtx.RLock()
	defer mtx.RUnlock()

	netList := make([]*model.Net, 0, len(nets))
	for _, net := range nets {
		netList = append(netList, net)
	}

	return netList, nil
}

func (netStorage *NetMemStorage) Delete(addr uint32, maskLen uint8, netType model.NetType) error {
	nets, mtx := netStorage.netsOfType(netType)
	mtx.Lock()
	defer mtx.Unlock()
	if net, found := nets[addr]; !found || net.MaskLen != maskLen {
		return nil
	}
	delete(nets, addr)
	return nil
}

func (netStorage *NetMemStorage) netsOfType(netType model.NetType) (map[uint32]*model.Net, *sync.RWMutex) {
	if netType == model.Black {
		return netStorage.black, netStorage.bmtx
	}
	return netStorage.white, netStorage.wmtx
}
