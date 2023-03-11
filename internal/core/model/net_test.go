package model

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNet_Contains(t *testing.T) {
	testCases := []struct {
		name      string
		netAddr   []byte
		maskLen   uint8
		ipToCheck []byte
		expected  bool
	}{
		{
			name:      "192.163.1.1 in 192.163.0.0/16",
			netAddr:   []byte{192, 163, 0, 0},
			maskLen:   16,
			ipToCheck: []byte{192, 163, 1, 1},
			expected:  true,
		},
		{
			name:      "192.163.254.254 in 192.163.0.0/16",
			netAddr:   []byte{192, 163, 0, 0},
			maskLen:   16,
			ipToCheck: []byte{192, 163, 254, 254},
			expected:  true,
		},
		{
			name:      "192.163.235.74 in 192.163.0.0/16",
			netAddr:   []byte{192, 163, 0, 0},
			maskLen:   16,
			ipToCheck: []byte{192, 163, 235, 74},
			expected:  true,
		},
		{
			name:      "192.164.235.74 in 192.163.0.0/16",
			netAddr:   []byte{192, 163, 0, 0},
			maskLen:   16,
			ipToCheck: []byte{192, 164, 235, 74},
			expected:  false,
		},
		{
			name:      "192.162.235.74 in 192.163.0.0/16",
			netAddr:   []byte{192, 163, 0, 0},
			maskLen:   16,
			ipToCheck: []byte{192, 162, 235, 74},
			expected:  false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			netAddrUint32 := binary.BigEndian.Uint32(tc.netAddr)
			net := Net{Addr: netAddrUint32, MaskLen: tc.maskLen}
			ipToCheckUint32 := binary.BigEndian.Uint32(tc.ipToCheck)
			require.Equal(t, tc.expected, net.Contains(ipToCheckUint32))
		})
	}

}
