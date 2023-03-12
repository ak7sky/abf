package service

import (
	"fmt"
	"math"
	"math/bits"
	"strconv"
	"time"

	"github.com/ak7sky/abf-service/internal/core"
	"github.com/ak7sky/abf-service/internal/core/model"
)

var (
	errCheckIPInList    = "failed to check IP in"
	errCheckBucket      = "failed to check bucket of"
	errResetBucket      = "failed to reset bucket"
	errAddIPToList      = "failed to add ip to"
	errRemoveIPFromList = "failed to remove ip from"
)

type BucketCapacities struct {
	loginBktCap, pswdBktCap, ipBktCap uint
}

type RateLimitService struct {
	netStorage core.NetStorage
	bktStorage core.BucketStorage
	bktCap     BucketCapacities
}

func New(netStorage core.NetStorage, bucketStorage core.BucketStorage, bktCap BucketCapacities) *RateLimitService {
	return &RateLimitService{
		netStorage: netStorage,
		bktStorage: bucketStorage,
		bktCap:     bktCap,
	}
}

func (rlsrv *RateLimitService) Ok(login string, pswd string, ip uint32) (bool, error) {
	isIPWhListed, err := rlsrv.isIPInList(ip, model.White)
	if err != nil {
		return false, err
	}
	if isIPWhListed {
		return true, nil
	}

	isIPBlListed, err := rlsrv.isIPInList(ip, model.Black)
	if err != nil {
		return false, err
	}
	if isIPBlListed {
		return false, nil
	}

	ok, err := rlsrv.addToBucket(model.LoginBkt, login)
	if !ok {
		return false, err
	}
	ok, err = rlsrv.addToBucket(model.PswdBkt, pswd)
	if !ok {
		return false, err
	}
	ok, err = rlsrv.addToBucket(model.IPBkt, strconv.Itoa(int(ip)))
	if !ok {
		return false, err
	}

	return true, nil
}

func (rlsrv *RateLimitService) Reset(login string, ip uint32) error {
	loginBucket, err := rlsrv.bktStorage.Get(login)
	if err != nil {
		return fmt.Errorf("%s '%s': %w", errResetBucket, login, err)
	}
	ipBucket, err := rlsrv.bktStorage.Get(strconv.Itoa(int(ip)))
	if err != nil {
		return fmt.Errorf("%s '%d': %w", errResetBucket, ip, err)
	}

	if loginBucket == nil {
		return fmt.Errorf("%s: bucket '%s' not found", errResetBucket, login)
	}

	if ipBucket == nil {
		return fmt.Errorf("%s: bucket '%d' not found", errResetBucket, ip)
	}

	loginBucket.Reset()
	ipBucket.Reset()

	if err = rlsrv.bktStorage.Save(loginBucket); err != nil {
		return fmt.Errorf("%s '%s': %w", errResetBucket, login, err)
	}
	if err = rlsrv.bktStorage.Save(ipBucket); err != nil {
		return fmt.Errorf("%v '%d': %w", errResetBucket, ip, err)
	}

	return nil
}

func (rlsrv *RateLimitService) AddToList(ip uint32, maskLen uint8, netType model.NetType) error {
	netAddr := ip & calcMask(maskLen)
	net, err := rlsrv.netStorage.Get(netAddr, netType)
	if err != nil {
		return fmt.Errorf("%s %s: %w", errAddIPToList, netType, err)
	}

	if net != nil {
		if maskLen >= net.MaskLen {
			return nil
		}
		net.MaskLen = maskLen
	} else {
		net = &model.Net{Addr: netAddr, MaskLen: maskLen}
	}

	err = rlsrv.netStorage.Save(net, netType)
	if err != nil {
		return fmt.Errorf("%s %s: %w", errAddIPToList, netType, err)
	}

	return nil
}

func (rlsrv *RateLimitService) RemoveFromList(ip uint32, maskLen uint8, netType model.NetType) error {
	netAddr := ip & calcMask(maskLen)
	err := rlsrv.netStorage.Delete(netAddr, maskLen, netType)
	if err != nil {
		return fmt.Errorf("%s %s: %w", errRemoveIPFromList, netType, err)
	}
	return nil
}

func (rlsrv *RateLimitService) isIPInList(ip uint32, netType model.NetType) (bool, error) {
	nets, err := rlsrv.netStorage.GetList(netType)
	if err != nil {
		return false, fmt.Errorf("%s %s: %w", errCheckIPInList, netType, err)
	}
	for _, network := range nets {
		if network.Contains(ip) {
			return true, nil
		}
	}
	return false, nil
}

func (rlsrv *RateLimitService) addToBucket(bt model.BktType, id string) (bool, error) {
	bucket, err := rlsrv.bktStorage.Get(id)
	if err != nil {
		return false, fmt.Errorf("%s %s: %w", errCheckBucket, bt, err)
	}

	if bucket == nil {
		switch bt {
		case model.LoginBkt:
			bucket = model.NewBucket(id, rlsrv.bktCap.loginBktCap, time.Minute)
		case model.PswdBkt:
			bucket = model.NewBucket(id, rlsrv.bktCap.pswdBktCap, time.Minute)
		case model.IPBkt:
			bucket = model.NewBucket(id, rlsrv.bktCap.ipBktCap, time.Minute)
		}
	}
	if !bucket.Add() {
		return false, nil
	}

	err = rlsrv.bktStorage.Save(bucket)
	if err != nil {
		return false, fmt.Errorf("%s %s: %w", errCheckBucket, bt, err)
	}

	return true, nil
}

func calcMask(maskLen uint8) uint32 {
	return bits.Reverse32(math.MaxUint32 >> (32 - maskLen))
}
