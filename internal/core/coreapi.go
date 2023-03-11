package core

import "abf-service/internal/core/model"

type RateLimitingService interface {
	Ok(login string, pswd string, ip uint32) (bool, error)
	Reset(login string, ip uint32) error
	AddToList(ip uint32, maskLen uint8, netType model.NetType) error
	RemoveFromList(ip uint32, maskLen uint8, netType model.NetType) error
}

type BucketStorage interface {
	Save(bucket *model.Bucket) error
	Get(id string) (*model.Bucket, error)
	Delete(id string) error
}

type NetStorage interface {
	Save(net *model.Net, netType model.NetType) error
	Get(addr uint32, netType model.NetType) (*model.Net, error)
	GetList(netType model.NetType) ([]*model.Net, error)
	Delete(addr uint32, maskLen uint8, netType model.NetType) error
}
