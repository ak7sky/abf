package service

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/ak7sky/abf-service/internal/core/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockNetStorage struct {
	mock.Mock
	getCallsCnt  int
	saveCallsCnt int
	delCallsCnt  int
}

func (mns *mockNetStorage) Save(net *model.Net, netType model.NetType) error {
	mns.saveCallsCnt++
	args := mns.Called(net, netType)
	return args.Error(0)
}

func (mns *mockNetStorage) Get(addr uint32, netType model.NetType) (*model.Net, error) {
	mns.getCallsCnt++
	args := mns.Called(addr, netType)
	return args.Get(0).(*model.Net), args.Error(1)
}

func (mns *mockNetStorage) GetList(netType model.NetType) ([]*model.Net, error) {
	args := mns.Called(netType)
	return args.Get(0).([]*model.Net), args.Error(1)
}

func (mns *mockNetStorage) Delete(addr uint32, maskLen uint8, netType model.NetType) error {
	mns.delCallsCnt++
	args := mns.Called(addr, maskLen, netType)
	return args.Error(0)
}

type mockBucketStorage struct {
	mock.Mock
	getCallsCnt  int
	saveCallsCnt int
}

func (mbs *mockBucketStorage) Save(bucket *model.Bucket) error {
	mbs.saveCallsCnt++
	args := mbs.Called(bucket)
	return args.Error(0)
}

func (mbs *mockBucketStorage) Get(id string) (*model.Bucket, error) {
	mbs.getCallsCnt++
	args := mbs.Called(id)
	return args.Get(0).(*model.Bucket), args.Error(1)
}

func (mbs *mockBucketStorage) Delete(id string) error {
	args := mbs.Called(id)
	return args.Error(0)
}

func TestOkIPCheck(t *testing.T) {
	inIP := ipV4(192, 168, 72, 31)
	testCases := []struct {
		name                         string
		errGetWhtList, errGetBlkList error
		whtList, blkList             []*model.Net
		expResult                    bool
		expErr                       error
	}{
		{
			name: "true: white listed ip",
			whtList: []*model.Net{
				{Addr: ipV4(192, 168, 0, 0), MaskLen: 16},
				{Addr: ipV4(192, 168, 5, 0), MaskLen: 24},
			},
			expResult: true,
		},
		{
			name:          "false: err get white list",
			errGetWhtList: errors.New("any err"),
			expResult:     false,
			expErr:        fmt.Errorf("%s %s: %w", errCheckIPInList, model.White, errors.New("any err")),
		},
		{
			name: "false: black listed ip",
			blkList: []*model.Net{
				{Addr: ipV4(192, 168, 0, 0), MaskLen: 16},
				{Addr: ipV4(192, 168, 5, 0), MaskLen: 24},
			},
			expResult: false,
		},
		{
			name:          "false: err get black list",
			errGetBlkList: errors.New("any err"),
			expResult:     false,
			expErr:        fmt.Errorf("%s %s: %w", errCheckIPInList, model.Black, errors.New("any err")),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			netStorage := &mockNetStorage{}
			bktStorage := &mockBucketStorage{}
			netStorage.On("GetList", model.White).Return(tc.whtList, tc.errGetWhtList)
			netStorage.On("GetList", model.Black).Return(tc.blkList, tc.errGetBlkList)
			rlsrv := NewRateLimitService(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

			ok, err := rlsrv.Ok("login123", "login123p@ssw0rD", inIP)
			require.Equal(t, err, tc.expErr)
			require.Equal(t, tc.expResult, ok)
		})
	}
}

func TestOkBucketsCheck(t *testing.T) {
	inLogin, inPswd, inIP := "login123", "login123p@ssw0rD", ipV4(192, 168, 72, 31)
	testCases := []struct {
		name                                          string
		currLoginBkt, currPswdBkt, currIPBkt          *model.Bucket
		errGetLoginBkt, errGetPswdBkt, errGetIPBkt    error
		errSaveLoginBkt, errSavePswdBkt, errSaveIPBkt error
		expGetCallsCnt                                int
		expSaveCallsCnt                               int
		expResult                                     bool
		expErr                                        error
	}{
		{
			name:            "true: new bkts",
			expGetCallsCnt:  3, // get 3 buckets: login, pswd, ip
			expSaveCallsCnt: 3, // save 3 new buckets: login, pswd, ip
			expResult:       true,
		},
		{
			name:            "true: bkts free space > 0",
			currLoginBkt:    freeBkt(inLogin, 10),
			currPswdBkt:     freeBkt(inPswd, 100),
			currIPBkt:       freeBkt(iptoa(inIP), 1000),
			expGetCallsCnt:  3, // get 3 buckets: login, pswd, ip
			expSaveCallsCnt: 3, // save 3 existing buckets: login, pswd, ip
			expResult:       true,
		},
		{
			name:            "false: login bkt full",
			currLoginBkt:    fullBkt(inLogin, 10),
			expGetCallsCnt:  1, // get 1 bucket: login
			expSaveCallsCnt: 0, // no calls of save
			expResult:       false,
		},
		{
			name:            "false: pswd bkt full",
			currLoginBkt:    freeBkt(inLogin, 10),
			currPswdBkt:     fullBkt(inPswd, 100),
			expGetCallsCnt:  2, // get 2 buckets: login, pswd
			expSaveCallsCnt: 1, // save 1 bucket: login
			expResult:       false,
		},
		{
			name:            "false: ip bkt full",
			currLoginBkt:    freeBkt(inLogin, 10),
			currPswdBkt:     freeBkt(inPswd, 100),
			currIPBkt:       fullBkt(iptoa(inIP), 1000),
			expGetCallsCnt:  3, // get 3 buckets: login, pswd, ip
			expSaveCallsCnt: 2, // save 2 bucket: login, pswd
			expResult:       false,
		},
		{
			name:            "false: err get login bkt",
			errGetLoginBkt:  errors.New("any err"),
			expResult:       false,
			expGetCallsCnt:  1, // get 1 bucket: login
			expSaveCallsCnt: 0, // no calls of save
			expErr:          fmt.Errorf("%s %s: %w", errCheckBucket, model.LoginBkt, errors.New("any err")),
		},
		{
			name:            "false: err get pswd bkt",
			errGetPswdBkt:   errors.New("any err"),
			expResult:       false,
			expErr:          fmt.Errorf("%s %s: %w", errCheckBucket, model.PswdBkt, errors.New("any err")),
			expGetCallsCnt:  2, // get 2 buckets: login, pswd
			expSaveCallsCnt: 1, // save 1 bucket: login
		},
		{
			name:            "false: err get ip bkt",
			errGetIPBkt:     errors.New("any err"),
			expResult:       false,
			expGetCallsCnt:  3, // get 2 bucket: login, pswd, ip
			expSaveCallsCnt: 2, // save 2 buckets: login, pswd
			expErr:          fmt.Errorf("%s %s: %w", errCheckBucket, model.IPBkt, errors.New("any err")),
		},
		{
			name:            "false: err save login bkt",
			errSaveLoginBkt: errors.New("any err"),
			expResult:       false,
			expGetCallsCnt:  1, // get 1 bucket: login
			expSaveCallsCnt: 1, // save 1 bucket: login
			expErr:          fmt.Errorf("%s %s: %w", errCheckBucket, model.LoginBkt, errors.New("any err")),
		},
		{
			name:            "false: err save pswd bkt",
			errSavePswdBkt:  errors.New("any err"),
			expResult:       false,
			expGetCallsCnt:  2, // get 2 buckets: login, pswd
			expSaveCallsCnt: 2, // save 2 buckets: login, pswd
			expErr:          fmt.Errorf("%s %s: %w", errCheckBucket, model.PswdBkt, errors.New("any err")),
		},
		{
			name:            "false: err save ip bkt",
			errSaveIPBkt:    errors.New("any err"),
			expResult:       false,
			expGetCallsCnt:  3, // get 3 buckets: login, pswd, ip
			expSaveCallsCnt: 3, // save 3 buckets: login, pswd, ip
			expErr:          fmt.Errorf("%s %s: %w", errCheckBucket, model.IPBkt, errors.New("any err")),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			netStorage := &mockNetStorage{}
			bktStorage := &mockBucketStorage{}
			netStorage.On("GetList", model.White).Return([]*model.Net{}, nil)
			netStorage.On("GetList", model.Black).Return([]*model.Net{}, nil)
			bktStorage.On("Get", inLogin).Return(tc.currLoginBkt, tc.errGetLoginBkt)
			bktStorage.On("Get", inPswd).Return(tc.currPswdBkt, tc.errGetPswdBkt)
			bktStorage.On("Get", iptoa(inIP)).Return(tc.currIPBkt, tc.errGetIPBkt)
			bktStorage.On("Save", mock.MatchedBy(bktIDMatcher(inLogin))).Return(tc.errSaveLoginBkt)
			bktStorage.On("Save", mock.MatchedBy(bktIDMatcher(inPswd))).Return(tc.errSavePswdBkt)
			bktStorage.On("Save", mock.MatchedBy(bktIDMatcher(iptoa(inIP)))).Return(tc.errSaveIPBkt)

			rlsrv := NewRateLimitService(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

			ok, err := rlsrv.Ok(inLogin, inPswd, inIP)

			require.Equal(t, err, tc.expErr)
			require.Equal(t, tc.expResult, ok)
			require.Equal(t, tc.expGetCallsCnt, bktStorage.getCallsCnt)
			require.Equal(t, tc.expSaveCallsCnt, bktStorage.saveCallsCnt)
		})
	}
}

func TestReset(t *testing.T) {
	inLogin, inIP := "login123", ipV4(192, 168, 72, 31)
	testCases := []struct {
		name                            string
		currLoginBkt, currIPBkt         *model.Bucket
		errGetLoginBkt, errGetIPBkt     error
		errSaveLoginBkt, errSaveIPBkt   error
		expGetCallsCnt, expSaveCallsCnt int
		expErr                          error
	}{
		{
			name:            "err: get login bkt",
			errGetLoginBkt:  errors.New("any err"),
			expGetCallsCnt:  1,
			expSaveCallsCnt: 0,
			expErr:          fmt.Errorf("%s '%s': %w", errResetBucket, inLogin, errors.New("any err")),
		},
		{
			name:            "err: get ip bkt",
			currLoginBkt:    fullBkt(inLogin, 10),
			errGetIPBkt:     errors.New("any err"),
			expGetCallsCnt:  2,
			expSaveCallsCnt: 0,
			expErr:          fmt.Errorf("%s '%s': %w", errResetBucket, iptoa(inIP), errors.New("any err")),
		},
		{
			name:            "err: login bkt not found",
			currLoginBkt:    nil,
			expGetCallsCnt:  1,
			expSaveCallsCnt: 0,
			expErr:          fmt.Errorf("%s: bucket '%s' not found", errResetBucket, inLogin),
		},
		{
			name:            "err: ip bkt not found",
			currLoginBkt:    fullBkt(inLogin, 10),
			currIPBkt:       nil,
			expGetCallsCnt:  2,
			expSaveCallsCnt: 0,
			expErr:          fmt.Errorf("%s: bucket '%s' not found", errResetBucket, iptoa(inIP)),
		},
		{
			name:            "err: save login bkt",
			currLoginBkt:    fullBkt(inLogin, 10),
			currIPBkt:       fullBkt(iptoa(inIP), 1000),
			errSaveLoginBkt: errors.New("any err"),
			expGetCallsCnt:  2,
			expSaveCallsCnt: 1,
			expErr:          fmt.Errorf("%s '%s': %w", errResetBucket, inLogin, errors.New("any err")),
		},
		{
			name:            "err: save ip bkt",
			currLoginBkt:    fullBkt(inLogin, 10),
			currIPBkt:       fullBkt(iptoa(inIP), 1000),
			errSaveIPBkt:    errors.New("any err"),
			expGetCallsCnt:  2,
			expSaveCallsCnt: 2,
			expErr:          fmt.Errorf("%s '%s': %w", errResetBucket, iptoa(inIP), errors.New("any err")),
		},
		{
			name:            "success",
			currLoginBkt:    fullBkt(inLogin, 10),
			currIPBkt:       fullBkt(iptoa(inIP), 1000),
			expGetCallsCnt:  2,
			expSaveCallsCnt: 2,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			netStorage := &mockNetStorage{}
			bktStorage := &mockBucketStorage{}
			bktStorage.On("Get", inLogin).Return(tc.currLoginBkt, tc.errGetLoginBkt)
			bktStorage.On("Get", iptoa(inIP)).Return(tc.currIPBkt, tc.errGetIPBkt)
			bktStorage.On("Save", mock.MatchedBy(bktIDMatcher(inLogin))).Return(tc.errSaveLoginBkt)
			bktStorage.On("Save", mock.MatchedBy(bktIDMatcher(iptoa(inIP)))).Return(tc.errSaveIPBkt)

			rlsrv := NewRateLimitService(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

			err := rlsrv.Reset(inLogin, inIP)
			require.Equal(t, err, tc.expErr)
			require.Equal(t, tc.expGetCallsCnt, bktStorage.getCallsCnt)
			require.Equal(t, tc.expSaveCallsCnt, bktStorage.saveCallsCnt)
			if err == nil {
				require.True(t, tc.currLoginBkt.FreeSpace == tc.currLoginBkt.Capacity)
				require.True(t, tc.currIPBkt.FreeSpace == tc.currIPBkt.Capacity)
			}
		})
	}
}

func TestAddToList(t *testing.T) {
	inMaskLen := uint8(16)
	inIP := ipV4(192, 168, 72, 31)
	expNetAddr := ipV4(192, 168, 0, 0)
	inNetType := model.Black

	testCases := []struct {
		name                            string
		errGetNet, errSaveNet           error
		foundNet                        *model.Net
		expGetCallsCnt, expSaveCallsCnt int
		expErr                          error
	}{
		{
			name:            "err: get net",
			errGetNet:       errors.New("any err"),
			expGetCallsCnt:  1,
			expSaveCallsCnt: 0,
			expErr:          fmt.Errorf("%s %s: %w", errAddIPToList, inNetType, errors.New("any err")),
		},
		{
			name:            "success: found mask <= input mask",
			foundNet:        &model.Net{Addr: expNetAddr, MaskLen: 8},
			expGetCallsCnt:  1,
			expSaveCallsCnt: 0,
		},
		{
			name:            "success: found mask > input mask",
			foundNet:        &model.Net{Addr: expNetAddr, MaskLen: 24},
			expGetCallsCnt:  1,
			expSaveCallsCnt: 1,
		},
		{
			name:            "success: new net",
			expGetCallsCnt:  1,
			expSaveCallsCnt: 1,
		},
		{
			name:            "err: save net",
			errSaveNet:      errors.New("any err"),
			expGetCallsCnt:  1,
			expSaveCallsCnt: 1,
			expErr:          fmt.Errorf("%s %s: %w", errAddIPToList, inNetType, errors.New("any err")),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			netStorage := &mockNetStorage{}
			bktStorage := &mockBucketStorage{}
			netStorage.On("Get", expNetAddr, inNetType).Return(tc.foundNet, tc.errGetNet)
			netStorage.
				On("Save", mock.MatchedBy(netMatcher(expNetAddr, inMaskLen)), model.Black).
				Return(tc.errSaveNet)

			rlsrv := NewRateLimitService(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

			err := rlsrv.AddToList(inIP, inMaskLen, inNetType)
			require.Equal(t, err, tc.expErr)
			require.Equal(t, tc.expGetCallsCnt, netStorage.getCallsCnt)
			require.Equal(t, tc.expSaveCallsCnt, netStorage.saveCallsCnt)
		})
	}
}

func TestRemoveFromList(t *testing.T) {
	inMaskLen := uint8(16)
	inIP := ipV4(192, 168, 72, 31)
	expNetAddr := ipV4(192, 168, 0, 0)
	inNetType := model.Black

	testCases := []struct {
		name           string
		errDelNet      error
		expDelCallsCnt int
		expErr         error
	}{
		{
			name:           "err: del net",
			errDelNet:      errors.New("any err"),
			expDelCallsCnt: 1,
			expErr:         fmt.Errorf("%s %s: %w", errRemoveIPFromList, inNetType, errors.New("any err")),
		},
		{
			name:           "success",
			expDelCallsCnt: 1,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			netStorage := &mockNetStorage{}
			bktStorage := &mockBucketStorage{}
			netStorage.On("Delete", expNetAddr, inMaskLen, inNetType).Return(tc.errDelNet)

			rlsrv := NewRateLimitService(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

			err := rlsrv.RemoveFromList(inIP, inMaskLen, inNetType)
			require.Equal(t, err, tc.expErr)
			require.Equal(t, tc.expDelCallsCnt, netStorage.delCallsCnt)
		})
	}
}

func ipV4(ip ...byte) uint32 {
	return binary.BigEndian.Uint32(ip)
}

func iptoa(ip uint32) string {
	return strconv.Itoa(int(ip))
}

func freeBkt(id string, capacity uint) *model.Bucket {
	bkt := model.NewBucket(id, capacity, time.Minute)
	bkt.FreeSpace = bkt.Capacity / 2
	return bkt
}

func fullBkt(id string, capacity uint) *model.Bucket {
	bkt := model.NewBucket(id, capacity, time.Minute)
	bkt.FreeSpace = 0
	return bkt
}

func bktIDMatcher(id string) func(bkt *model.Bucket) bool {
	return func(bkt *model.Bucket) bool { return bkt.ID == id }
}

func netMatcher(netAddr uint32, maskLen uint8) func(net *model.Net) bool {
	return func(net *model.Net) bool { return net.Addr == netAddr && net.MaskLen == maskLen }
}
