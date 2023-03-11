package model

import (
	"math"
	"math/bits"
)

type NetType string

const (
	White NetType = "white list"
	Black NetType = "black list"
)

type Net struct {
	Addr    uint32
	MaskLen uint8
}

func (net *Net) Contains(ip uint32) bool {
	mask := bits.Reverse32(math.MaxUint32 >> (32 - net.MaskLen))
	return ip&mask == net.Addr
}
