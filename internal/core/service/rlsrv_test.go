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
}

func (mns *mockNetStorage) Save(net *model.Net, netType model.NetType) error {
	args := mns.Called(net, netType)
	return args.Error(0)
}

func (mns *mockNetStorage) Get(addr uint32, netType model.NetType) (*model.Net, error) {
	args := mns.Called(addr, netType)
	return args.Get(0).(*model.Net), args.Error(1)
}

func (mns *mockNetStorage) GetList(netType model.NetType) ([]*model.Net, error) {
	args := mns.Called(netType)
	return args.Get(0).([]*model.Net), args.Error(1)
}

func (mns *mockNetStorage) Delete(addr uint32, maskLen uint8, netType model.NetType) error {
	args := mns.Called(addr, maskLen, netType)
	return args.Error(0)
}

type mockBucketStorage struct {
	mock.Mock
	triedToSave []*model.Bucket
}

func (mbs *mockBucketStorage) Save(bucket *model.Bucket) error {
	mbs.triedToSave = append(mbs.triedToSave, bucket)
	args := mbs.Called(bucket)
	return args.Error(0)
}

func (mbs *mockBucketStorage) Get(id string) (*model.Bucket, error) {
	args := mbs.Called(id)
	return args.Get(0).(*model.Bucket), args.Error(1)
}

func (mbs *mockBucketStorage) Delete(id string) error {
	args := mbs.Called(id)
	return args.Error(0)
}

func TestOk(t *testing.T) {
	t.Run("true: ip in white list", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		whiteList := []*model.Net{
			{Addr: ipV4(192, 168, 0, 0), MaskLen: 16},
			{Addr: ipV4(192, 168, 5, 0), MaskLen: 24},
		}
		netStorage.On("GetList", model.White).Return(whiteList, nil)

		ok, err := rlsrv.Ok("login123", "login123p@ssw0rD", ipV4(192, 168, 72, 31))

		require.NoError(t, err)
		require.True(t, ok)
		netStorage.AssertCalled(t, "GetList", model.White)
	})

	t.Run("false: err get white list", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		expErrStorage := errors.New("error during getting white list")
		netStorage.On("GetList", model.White).Return(([]*model.Net)(nil), expErrStorage)

		ok, err := rlsrv.Ok("login123", "login123p@ssw0rD", ipV4(192, 168, 72, 31))

		require.EqualError(t, err, fmt.Sprintf("%s %s: %v", errCheckIPInList, model.White, expErrStorage))
		require.False(t, ok)
		netStorage.AssertCalled(t, "GetList", model.White)
	})

	t.Run("false: ip in black list", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		netStorage.On("GetList", model.White).Return([]*model.Net{}, nil)

		blackList := []*model.Net{
			{Addr: ipV4(192, 168, 0, 0), MaskLen: 16},
			{Addr: ipV4(192, 168, 5, 0), MaskLen: 24},
		}
		netStorage.On("GetList", model.Black).Return(blackList, nil)

		ok, err := rlsrv.Ok("login123", "login123p@ssw0rD", ipV4(192, 168, 72, 31))

		require.NoError(t, err)
		require.False(t, ok)
		netStorage.AssertCalled(t, "GetList", model.White)
		netStorage.AssertCalled(t, "GetList", model.Black)
	})

	t.Run("false: err get black list", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		netStorage.On("GetList", model.White).Return([]*model.Net{}, nil)

		expErrStorage := errors.New("error during getting black list")
		netStorage.On("GetList", model.Black).Return(([]*model.Net)(nil), expErrStorage)

		ok, err := rlsrv.Ok("login123", "login123p@ssw0rD", ipV4(192, 168, 72, 31))

		require.EqualError(t, err, fmt.Sprintf("%s %s: %v", errCheckIPInList, model.Black, expErrStorage))
		require.False(t, ok)
		netStorage.AssertCalled(t, "GetList", model.White)
		netStorage.AssertCalled(t, "GetList", model.Black)
	})

	t.Run("false: err get bkt", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		netStorage.On("GetList", model.White).Return([]*model.Net{}, nil)
		netStorage.On("GetList", model.Black).Return([]*model.Net{}, nil)

		inputLogin := "login123"
		errGetLoginBkt := errors.New("err during getting login bkt")
		bktStorage.On("Get", inputLogin).Return((*model.Bucket)(nil), errGetLoginBkt)

		ok, err := rlsrv.Ok(inputLogin, "login123p@ssw0rD", ipV4(192, 168, 72, 31))

		require.EqualError(t, err, fmt.Sprintf("%s %s: %v", errCheckBucket, model.LoginBkt, errGetLoginBkt))
		require.False(t, ok)
		netStorage.AssertCalled(t, "GetList", model.White)
		netStorage.AssertCalled(t, "GetList", model.Black)
		bktStorage.AssertCalled(t, "Get", inputLogin)
	})

	t.Run("false: bkt full", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		netStorage.On("GetList", model.White).Return([]*model.Net{}, nil)
		netStorage.On("GetList", model.Black).Return([]*model.Net{}, nil)

		inputLogin := "login123"
		expLoginBkt := model.NewBucket(inputLogin, 5, time.Minute)
		expLoginBkt.FreeSpace = 0
		bktStorage.On("Get", inputLogin).Return(expLoginBkt, nil)

		ok, err := rlsrv.Ok(inputLogin, "login123p@ssw0rD", ipV4(192, 168, 72, 31))

		require.NoError(t, err)
		require.False(t, ok)

		netStorage.AssertCalled(t, "GetList", model.White)
		netStorage.AssertCalled(t, "GetList", model.Black)
		bktStorage.AssertCalled(t, "Get", inputLogin)
	})

	t.Run("false: err save bkt", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		netStorage.On("GetList", model.White).Return([]*model.Net{}, nil)
		netStorage.On("GetList", model.Black).Return([]*model.Net{}, nil)

		inputLogin := "login123"
		expLoginBkt := model.NewBucket(inputLogin, 5, time.Minute)
		expLoginBkt.FreeSpace = 4
		bktStorage.On("Get", inputLogin).Return(expLoginBkt, nil)

		errSaveLoginBkt := errors.New("err during saving login bkt")
		bktStorage.On("Save", expLoginBkt).Return(errSaveLoginBkt)

		ok, err := rlsrv.Ok(inputLogin, "login123p@ssw0rD", ipV4(192, 168, 72, 31))

		require.EqualError(t, err, fmt.Sprintf("%s %s: %v", errCheckBucket, model.LoginBkt, errSaveLoginBkt))
		require.False(t, ok)

		netStorage.AssertCalled(t, "GetList", model.White)
		netStorage.AssertCalled(t, "GetList", model.Black)
		bktStorage.AssertCalled(t, "Get", inputLogin)
		bktStorage.AssertCalled(t, "Save", expLoginBkt)
	})

	t.Run("true: bkts have free space", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		netStorage.On("GetList", model.White).Return([]*model.Net{}, nil)
		netStorage.On("GetList", model.Black).Return([]*model.Net{}, nil)

		inputLogin, inputPswd, inputIP := "login123", "login123p@ssw0rD", ipV4(192, 168, 72, 31)

		expLoginBkt := model.NewBucket(inputLogin, 5, time.Minute)
		expPswdBkt := model.NewBucket(inputPswd, 10, time.Minute)
		expIPBkt := model.NewBucket(iptoa(inputIP), 1000, time.Minute)

		expLoginBkt.FreeSpace--
		expPswdBkt.FreeSpace--
		expIPBkt.FreeSpace--

		bktStorage.On("Get", inputLogin).Return(expLoginBkt, nil)
		bktStorage.On("Get", inputPswd).Return(expPswdBkt, nil)
		bktStorage.On("Get", iptoa(inputIP)).Return(expIPBkt, nil)

		bktStorage.On("Save", expLoginBkt).Return(nil)
		bktStorage.On("Save", expPswdBkt).Return(nil)
		bktStorage.On("Save", expIPBkt).Return(nil)

		ok, err := rlsrv.Ok(inputLogin, inputPswd, inputIP)

		require.True(t, ok)
		require.NoError(t, err)

		netStorage.AssertCalled(t, "GetList", model.White)
		netStorage.AssertCalled(t, "GetList", model.Black)
		bktStorage.AssertCalled(t, "Get", inputLogin)
		bktStorage.AssertCalled(t, "Get", inputPswd)
		bktStorage.AssertCalled(t, "Get", iptoa(inputIP))
		bktStorage.AssertCalled(t, "Save", expLoginBkt)
		bktStorage.AssertCalled(t, "Save", expPswdBkt)
		bktStorage.AssertCalled(t, "Save", expIPBkt)
	})

	t.Run("true: new bkts", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		loginBktCap, pswdBktCap, ipBktCap, duration := uint(10), uint(100), uint(1000), time.Minute
		rlsrv := New(netStorage, bktStorage, BucketCapacities{loginBktCap, pswdBktCap, ipBktCap})

		netStorage.On("GetList", model.White).Return([]*model.Net{}, nil)
		netStorage.On("GetList", model.Black).Return([]*model.Net{}, nil)

		inputLogin, inputPswd, inputIP := "login123", "login123p@ssw0rD", ipV4(192, 168, 72, 31)

		bktStorage.On("Get", inputLogin).Return((*model.Bucket)(nil), nil)
		bktStorage.On("Get", inputPswd).Return((*model.Bucket)(nil), nil)
		bktStorage.On("Get", iptoa(inputIP)).Return((*model.Bucket)(nil), nil)

		bktStorage.On("Save", mock.AnythingOfType("*model.Bucket")).Return(nil)

		ok, err := rlsrv.Ok(inputLogin, inputPswd, inputIP)

		require.True(t, ok)
		require.NoError(t, err)

		netStorage.AssertCalled(t, "GetList", model.White)
		netStorage.AssertCalled(t, "GetList", model.Black)
		bktStorage.AssertCalled(t, "Get", inputLogin)
		bktStorage.AssertCalled(t, "Get", inputPswd)
		bktStorage.AssertCalled(t, "Get", iptoa(inputIP))

		require.Equal(t, 3, len(bktStorage.triedToSave))
		require.Equal(t, inputLogin, bktStorage.triedToSave[0].ID)
		require.Equal(t, loginBktCap, bktStorage.triedToSave[0].Capacity)
		require.Equal(t, loginBktCap-1, bktStorage.triedToSave[0].FreeSpace)
		require.Equal(t, duration, bktStorage.triedToSave[0].Duration)

		require.Equal(t, inputPswd, bktStorage.triedToSave[1].ID)
		require.Equal(t, pswdBktCap, bktStorage.triedToSave[1].Capacity)
		require.Equal(t, pswdBktCap-1, bktStorage.triedToSave[1].FreeSpace)
		require.Equal(t, duration, bktStorage.triedToSave[1].Duration)

		require.Equal(t, iptoa(inputIP), bktStorage.triedToSave[2].ID)
		require.Equal(t, ipBktCap, bktStorage.triedToSave[2].Capacity)
		require.Equal(t, ipBktCap-1, bktStorage.triedToSave[2].FreeSpace)
		require.Equal(t, duration, bktStorage.triedToSave[2].Duration)
	})
}

func TestReset(t *testing.T) {
	t.Run("err: get login bkt", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		inputLogin := "login123"
		expErrStorage := errors.New("err get login bkt")
		bktStorage.On("Get", inputLogin).Return((*model.Bucket)(nil), expErrStorage)

		err := rlsrv.Reset(inputLogin, ipV4(192, 168, 72, 31))

		require.EqualError(t, err, fmt.Sprintf("%s '%s': %v", errResetBucket, inputLogin, expErrStorage))
		bktStorage.AssertCalled(t, "Get", inputLogin)
	})

	t.Run("err: get ip bkt", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		inputLogin := "login123"
		inputIP := ipV4(192, 168, 72, 31)
		expErrStorage := errors.New("err get ip bkt")

		bktStorage.On("Get", inputLogin).Return(&model.Bucket{}, nil)
		bktStorage.On("Get", iptoa(inputIP)).Return((*model.Bucket)(nil), expErrStorage)

		err := rlsrv.Reset(inputLogin, inputIP)

		require.EqualError(t, err, fmt.Sprintf("%s '%d': %v", errResetBucket, inputIP, expErrStorage))
		bktStorage.AssertCalled(t, "Get", inputLogin)
		bktStorage.AssertCalled(t, "Get", iptoa(inputIP))
	})

	t.Run("err: ip bkt not found", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		inputLogin := "login123"
		inputIP := ipV4(192, 168, 72, 31)

		bktStorage.On("Get", inputLogin).Return(&model.Bucket{}, nil)
		bktStorage.On("Get", iptoa(inputIP)).Return((*model.Bucket)(nil), nil)

		err := rlsrv.Reset(inputLogin, inputIP)

		require.EqualError(t, err, fmt.Sprintf("%s: bucket '%d' not found", errResetBucket, inputIP))
		bktStorage.AssertCalled(t, "Get", inputLogin)
		bktStorage.AssertCalled(t, "Get", iptoa(inputIP))
	})

	t.Run("err: save login bkt", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		loginBktCap, pswdBktCap, ipBktCap, duration := uint(10), uint(100), uint(1000), time.Minute
		rlsrv := New(netStorage, bktStorage, BucketCapacities{loginBktCap, pswdBktCap, ipBktCap})

		inputLogin := "login123"
		inputIP := ipV4(192, 168, 72, 31)

		foundLoginBkt := model.NewBucket(inputLogin, loginBktCap, duration)
		foundLoginBkt.FreeSpace = 0
		foundIPBkt := model.NewBucket(iptoa(inputIP), ipBktCap, duration)
		foundIPBkt.FreeSpace = 0

		bktStorage.On("Get", inputLogin).Return(foundLoginBkt, nil)
		bktStorage.On("Get", iptoa(inputIP)).Return(foundIPBkt, nil)

		expErrStorage := errors.New("err save login bkt")
		bktStorage.On("Save", foundLoginBkt).Return(expErrStorage)

		err := rlsrv.Reset(inputLogin, inputIP)

		require.EqualError(t, err, fmt.Sprintf("%s '%s': %v", errResetBucket, inputLogin, expErrStorage))
		bktStorage.AssertCalled(t, "Get", inputLogin)
		bktStorage.AssertCalled(t, "Get", iptoa(inputIP))
		require.Equal(t, loginBktCap, foundLoginBkt.FreeSpace)
		require.Equal(t, ipBktCap, foundIPBkt.FreeSpace)
		bktStorage.AssertCalled(t, "Save", foundLoginBkt)
	})

	t.Run("err: save ip bkt", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		loginBktCap, pswdBktCap, ipBktCap, duration := uint(10), uint(100), uint(1000), time.Minute
		rlsrv := New(netStorage, bktStorage, BucketCapacities{loginBktCap, pswdBktCap, ipBktCap})

		inputLogin := "login123"
		inputIP := ipV4(192, 168, 72, 31)

		foundLoginBkt := model.NewBucket(inputLogin, loginBktCap, duration)
		foundLoginBkt.FreeSpace = 0
		foundIPBkt := model.NewBucket(iptoa(inputIP), ipBktCap, duration)
		foundIPBkt.FreeSpace = 0

		bktStorage.On("Get", inputLogin).Return(foundLoginBkt, nil)
		bktStorage.On("Get", iptoa(inputIP)).Return(foundIPBkt, nil)

		bktStorage.On("Save", foundLoginBkt).Return(nil)
		expErrStorage := errors.New("err save ip bkt")
		bktStorage.On("Save", foundIPBkt).Return(expErrStorage)

		err := rlsrv.Reset(inputLogin, inputIP)

		require.EqualError(t, err, fmt.Sprintf("%s '%d': %v", errResetBucket, inputIP, expErrStorage))
		bktStorage.AssertCalled(t, "Get", inputLogin)
		bktStorage.AssertCalled(t, "Get", iptoa(inputIP))
		require.Equal(t, loginBktCap, foundLoginBkt.FreeSpace)
		require.Equal(t, ipBktCap, foundIPBkt.FreeSpace)
		bktStorage.AssertCalled(t, "Save", foundLoginBkt)
		bktStorage.AssertCalled(t, "Save", foundIPBkt)
	})

	t.Run("success", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		loginBktCap, pswdBktCap, ipBktCap, duration := uint(10), uint(100), uint(1000), time.Minute
		rlsrv := New(netStorage, bktStorage, BucketCapacities{loginBktCap, pswdBktCap, ipBktCap})

		inputLogin := "login123"
		inputIP := ipV4(192, 168, 72, 31)

		foundLoginBkt := model.NewBucket(inputLogin, loginBktCap, duration)
		foundLoginBkt.FreeSpace = 0
		foundIPBkt := model.NewBucket(iptoa(inputIP), ipBktCap, duration)
		foundIPBkt.FreeSpace = 0

		bktStorage.On("Get", inputLogin).Return(foundLoginBkt, nil)
		bktStorage.On("Get", iptoa(inputIP)).Return(foundIPBkt, nil)
		bktStorage.On("Save", foundLoginBkt).Return(nil)
		bktStorage.On("Save", foundIPBkt).Return(nil)

		err := rlsrv.Reset(inputLogin, inputIP)

		require.NoError(t, err)
		bktStorage.AssertCalled(t, "Get", inputLogin)
		bktStorage.AssertCalled(t, "Get", iptoa(inputIP))
		require.Equal(t, loginBktCap, foundLoginBkt.FreeSpace)
		require.Equal(t, ipBktCap, foundIPBkt.FreeSpace)
		bktStorage.AssertCalled(t, "Save", foundLoginBkt)
		bktStorage.AssertCalled(t, "Save", foundIPBkt)
	})
}

func TestAddToList(t *testing.T) {
	t.Run("err: get net", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		expNetAddr := ipV4(192, 168, 72, 0)
		expErrStorage := errors.New("error during getting net")
		netStorage.On("Get", expNetAddr, model.Black).Return((*model.Net)(nil), expErrStorage)

		err := rlsrv.AddToList(ipV4(192, 168, 72, 31), uint8(24), model.Black)

		require.EqualError(t, err, fmt.Sprintf("%s %s: %v", errAddIPToList, model.Black, expErrStorage))
		netStorage.AssertCalled(t, "Get", expNetAddr, model.Black)
	})

	t.Run("success: found mask <= input mask", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		inputIP := ipV4(192, 168, 0, 31)
		inputMask := uint8(24)

		expNetAddr := ipV4(192, 168, 0, 0)
		expFoundNet := &model.Net{Addr: expNetAddr, MaskLen: 16}
		netStorage.On("Get", expNetAddr, model.Black).Return(expFoundNet, nil)

		err := rlsrv.AddToList(inputIP, inputMask, model.Black)
		require.NoError(t, err)
		netStorage.AssertCalled(t, "Get", expNetAddr, model.Black)
		netStorage.AssertNotCalled(t, "Save")
	})

	t.Run("success: found mask > input mask", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		inputIP := ipV4(192, 168, 0, 31)
		inputMask := uint8(16)

		expNetAddr := ipV4(192, 168, 0, 0)
		expFoundNet := &model.Net{Addr: expNetAddr, MaskLen: 24}
		netStorage.On("Get", expNetAddr, model.Black).Return(expFoundNet, nil)

		netStorage.On("Save", expFoundNet, model.Black).Return(nil)

		err := rlsrv.AddToList(inputIP, inputMask, model.Black)

		require.NoError(t, err)
		require.Equal(t, inputMask, expFoundNet.MaskLen)
		netStorage.AssertCalled(t, "Get", expNetAddr, model.Black)
		netStorage.AssertCalled(t, "Save", expFoundNet, model.Black)
	})

	t.Run("success: new net", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		inputIP := ipV4(192, 168, 0, 31)
		inputMask := uint8(24)

		expNetAddr := ipV4(192, 168, 0, 0)
		expNet := &model.Net{Addr: expNetAddr, MaskLen: 24}
		netStorage.On("Get", expNetAddr, model.Black).Return((*model.Net)(nil), nil)
		netStorage.On("Save", expNet, model.Black).Return(nil)

		err := rlsrv.AddToList(inputIP, inputMask, model.Black)

		require.NoError(t, err)
		netStorage.AssertCalled(t, "Get", expNetAddr, model.Black)
		netStorage.AssertCalled(t, "Save", expNet, model.Black)
	})

	t.Run("err: save net", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		inputIP := ipV4(192, 168, 0, 31)
		inputMask := uint8(24)

		expNetAddr := ipV4(192, 168, 0, 0)
		expNet := &model.Net{Addr: expNetAddr, MaskLen: 24}
		netStorage.On("Get", expNet.Addr, model.Black).Return((*model.Net)(nil), nil)

		expErrStorage := errors.New("error during saving net")
		netStorage.On("Save", expNet, model.Black).Return(expErrStorage)

		err := rlsrv.AddToList(inputIP, inputMask, model.Black)

		require.EqualError(t, err, fmt.Sprintf("%s %s: %v", errAddIPToList, model.Black, expErrStorage))
		netStorage.AssertCalled(t, "Get", expNetAddr, model.Black)
		netStorage.AssertCalled(t, "Save", expNet, model.Black)
	})
}

func TestRemoveFromList(t *testing.T) {
	t.Run("err: delete net", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		inputIP := ipV4(192, 168, 72, 31)
		inputMask := uint8(24)

		expNetAddr := ipV4(192, 168, 72, 0)
		expErrStorage := errors.New("error during deleting net")
		netStorage.On("Delete", expNetAddr, inputMask, model.Black).Return(expErrStorage)

		err := rlsrv.RemoveFromList(inputIP, inputMask, model.Black)
		require.EqualError(t, err, fmt.Sprintf("%s %s: %v", errRemoveIPFromList, model.Black, expErrStorage))
		netStorage.AssertCalled(t, "Delete", expNetAddr, inputMask, model.Black)
	})

	t.Run("success", func(t *testing.T) {
		netStorage := &mockNetStorage{}
		bktStorage := &mockBucketStorage{}
		rlsrv := New(netStorage, bktStorage, BucketCapacities{10, 100, 1000})

		inputIP := ipV4(192, 168, 72, 31)
		inputMask := uint8(24)

		expNetAddr := ipV4(192, 168, 72, 0)
		netStorage.On("Delete", expNetAddr, inputMask, model.Black).Return(nil)

		err := rlsrv.RemoveFromList(inputIP, inputMask, model.Black)
		require.NoError(t, err)
		netStorage.AssertCalled(t, "Delete", expNetAddr, inputMask, model.Black)
	})
}

func ipV4(ip ...byte) uint32 {
	return binary.BigEndian.Uint32(ip)
}

func iptoa(ip uint32) string {
	return strconv.Itoa(int(ip))
}
