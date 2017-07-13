package etcd

import (
	"testing"

	"github.com/coreos/etcd/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/netw00rk/encoding/etcd/test"
)

func TestEncodePrimitiveTypes(t *testing.T) {
	var actual interface{}
	etcd := new(test.KeysAPIMock)
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/value", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		actual = args.Get(2)
	})
	encoder := NewEncoder(etcd)

	a := 10
	err := encoder.Encode("/path/to/some/value", a)
	assert.Nil(t, err)
	assert.Equal(t, "10", actual)

	b := 10.5
	err = encoder.Encode("/path/to/some/value", b)
	assert.Nil(t, err)
	assert.Equal(t, "10.5", actual)

	c := true
	err = encoder.Encode("/path/to/some/value", c)
	assert.Nil(t, err)
	assert.Equal(t, "true", actual)

	d := 5
	err = encoder.Encode("/path/to/some/value", &d)
	assert.Nil(t, err)
	assert.Equal(t, "5", actual)
}

func TestEncodeInterface(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/value", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "10", args.Get(2))
	})

	var a interface{}
	a = 10
	encoder := NewEncoder(etcd)
	err := encoder.Encode("/path/to/some/value", &a)
	assert.Nil(t, err)
}

func TestEncodeSimpleStruct(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Delete", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct", mock.AnythingOfType("*client.DeleteOptions")).Return(nil, nil)
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field1", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "10", args.Get(2))
	})
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field2", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "false", args.Get(2))
	})

	var s = struct {
		Field1 int
		Field2 bool
	}{Field1: 10, Field2: false}

	encoder := NewEncoder(etcd)
	err := encoder.Encode("/path/to/some/struct", s)
	assert.Nil(t, err)
}

func TestEncodeStructWithTag(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field1", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "10", args.Get(2))
	})
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/field_2", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "false", args.Get(2))
	})

	var s = struct {
		Field1 int
		Field2 bool `etcd:"field_2"`
	}{Field1: 10, Field2: false}

	encoder := NewEncoder(etcd)
	err := encoder.Encode("/path/to/some/struct", s)
	assert.Nil(t, err)
}

func TestEncodeMap(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/map/field_1", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "10", args.Get(2))
	})
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/map/field_2", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "20", args.Get(2))
	})

	var m = map[string]int{
		"field_1": 10,
		"field_2": 20,
	}

	encoder := NewEncoder(etcd)
	err := encoder.Encode("/path/to/some/map", m)
	assert.Nil(t, err)
}

func TestEncodeSlice(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/slice/0", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "10", args.Get(2))
	})
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/slice/1", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "20", args.Get(2))
	})

	var s = []int{10, 20}
	encoder := NewEncoder(etcd)
	err := encoder.Encode("/path/to/some/slice", s)
	assert.Nil(t, err)
}

func TestEncodeComplexStruct(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/complex/struct/Field1", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "10", args.Get(2))
	})
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/complex/struct/Field2/field_2_1", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "false", args.Get(2))
	})
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/complex/struct/Field3/field_3_1/0", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "30", args.Get(2))
	})

	var s = struct {
		Field1 int
		Field2 struct {
			Field21 bool `etcd:"field_2_1"`
		}
		Field3 map[string][]int
	}{
		Field1: 10,
		Field2: struct {
			Field21 bool `etcd:"field_2_1"`
		}{
			Field21: false,
		},
		Field3: map[string][]int{
			"field_3_1": []int{30},
		},
	}

	encoder := NewEncoder(etcd)
	err := encoder.Encode("/path/to/complex/struct", s)
	assert.Nil(t, err)
}

func TestEncodeNotImplementedType(t *testing.T) {
	var c = make(chan bool)
	encoder := NewEncoder(new(test.KeysAPIMock))
	err := encoder.Encode("/path", c)
	assert.Equal(t, "can't encode type chan bool", err.Error())
}

func TestEncodeIgnoreTag(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Set", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/field", mock.Anything, mock.AnythingOfType("*client.SetOptions")).Return(&client.Response{}, nil).Run(func(args mock.Arguments) {
		assert.Equal(t, "10", args.Get(2))
	})

	var s = struct {
		Field       int  `etcd:"field"`
		IgnoreField bool `etcd:"-"`
	}{
		Field:       10,
		IgnoreField: false,
	}

	encoder := NewEncoder(etcd)
	err := encoder.Encode("/path/to/some/struct", s)
	assert.Nil(t, err)
}
