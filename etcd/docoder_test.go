package etcd

import (
	"testing"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/netw00rk/encoding/etcd/test"
)

func TestFailIfDestinationIsNotAPointer(t *testing.T) {
	decoder := NewDecoder(new(test.KeysAPIMock))
	err := decoder.Decode("/some/path", 10)
	assert.Equal(t, "destination has to be a pointer", err.Error())
}

func TestDecodePrimitiveTypes(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/value", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/value/bool", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "false"}}, nil)
	decoder := NewDecoder(etcd)

	var a int
	err := decoder.Decode("/path/to/some/value", &a)
	assert.Nil(t, err)
	assert.Equal(t, 10, a)

	var b int64
	err = decoder.Decode("/path/to/some/value", &b)
	assert.Nil(t, err)
	assert.Equal(t, int64(10), b)

	var c float32
	err = decoder.Decode("/path/to/some/value", &c)
	assert.Nil(t, err)
	assert.Equal(t, float32(10), c)

	var d float64
	err = decoder.Decode("/path/to/some/value", &d)
	assert.Nil(t, err)
	assert.Equal(t, float64(10), d)

	var e string
	err = decoder.Decode("/path/to/some/value", &e)
	assert.Nil(t, err)
	assert.Equal(t, "10", e)

	var f bool
	err = decoder.Decode("/path/to/some/value/bool", &f)
	assert.Nil(t, err)
	assert.Equal(t, false, f)

	var g time.Duration
	err = decoder.Decode("/path/to/some/value", &g)
	assert.Nil(t, err)
	assert.Equal(t, time.Duration(10), g)
}

func TestDecodeInterface(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/value", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)

	var a interface{}
	a = 0

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/value", &a)
	assert.Nil(t, err)
	assert.Equal(t, 10, a)
}

func TestDecodeStruct(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field1", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field2", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "string"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field3", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "true"}}, nil)

	var s = struct {
		Field1 int64
		Field2 string
		Field3 bool
	}{}

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/struct", &s)
	assert.Nil(t, err)
	assert.Equal(t, int64(10), s.Field1)
	assert.Equal(t, "string", s.Field2)
	assert.Equal(t, true, s.Field3)
}

func TestDecodeNestedStruct(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field1", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field2/Field1", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "string"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field2/Field2", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "true"}}, nil)

	var s = struct {
		Field1 int64
		Field2 struct {
			Field1 string
			Field2 bool
		}
	}{}

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/struct", &s)
	assert.Nil(t, err)
	assert.Equal(t, int64(10), s.Field1)
	assert.Equal(t, "string", s.Field2.Field1)
	assert.Equal(t, true, s.Field2.Field2)
}

func TestDecodeStructWithTag(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/field_1", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field2", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "20"}}, nil)

	var s = struct {
		Field1 int64 `etcd:"field_1"`
		Field2 int64
	}{}

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/struct", &s)
	assert.Nil(t, err)
	assert.Equal(t, int64(10), s.Field1)
	assert.Equal(t, int64(20), s.Field2)
}

func TestDecodeSimpleMap(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir:   true,
		Nodes: []*client.Node{&client.Node{Key: "/path/to/some/struct/field_1"}, &client.Node{Key: "/path/to/some/struct/field_2"}},
	}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/field_1", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/field_2", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "30"}}, nil)

	var m map[string]int

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/struct", &m)
	assert.Nil(t, err)
	assert.Equal(t, 10, m["field_1"])
	assert.Equal(t, 30, m["field_2"])
}

func TestDecodeSimpleMapInsideStruct(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Map", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir:   true,
		Nodes: []*client.Node{&client.Node{Key: "/path/to/some/struct/Map/field_1"}, &client.Node{Key: "/path/to/some/struct/Map/field_2"}},
	}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Map/field_1", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Map/field_2", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "30"}}, nil)

	var s = struct {
		Map map[string]int
	}{}

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/struct", &s)
	assert.Nil(t, err)
	assert.Equal(t, 10, s.Map["field_1"])
	assert.Equal(t, 30, s.Map["field_2"])
}

func TestDecodeMapOfStructs(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/map", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir:   true,
		Nodes: []*client.Node{&client.Node{Key: "/path/to/some/map/field_1"}, &client.Node{Key: "/path/to/some/map/field_2"}},
	}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/map/field_1/Field1", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/map/field_2/Field1", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "30"}}, nil)

	var m map[string]struct {
		Field1 int64
	}

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/map", &m)
	assert.Nil(t, err)
	assert.Equal(t, int64(10), m["field_1"].Field1)
	assert.Equal(t, int64(30), m["field_2"].Field1)
}

func TestDecodeSimpleSlice(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/slice", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir:   true,
		Nodes: []*client.Node{&client.Node{Key: "/path/to/slice/1"}, &client.Node{Key: "/path/to/slice/2"}, &client.Node{Key: "/path/to/slice/3"}},
	}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/slice/1", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/slice/2", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "30"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/slice/3", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "50"}}, nil)

	var s = make([]int, 0)

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/slice", &s)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(s))
	assert.Equal(t, 10, s[0])
	assert.Equal(t, 30, s[1])
	assert.Equal(t, 50, s[2])
}

func TestDecodeIgnoreTag(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/field", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)

	var s = struct {
		Field1 int64 `etcd:"field"`
		Field2 int64 `etcd:"-"`
	}{}

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/struct", &s)
	assert.Nil(t, err)
	assert.Equal(t, int64(10), s.Field1)
	assert.Equal(t, int64(0), s.Field2)
}

func TestDecodeSkipMissingError(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field", mock.AnythingOfType("*client.GetOptions")).Return(nil, client.Error{Code: client.ErrorCodeKeyNotFound, Message: "Key not found"})
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field2", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)

	var s = struct {
		Field  int64
		Field2 int64
	}{}

	decoder := NewDecoder(etcd)
	decoder.SkipMissing(true)
	err := decoder.Decode("/path/to/some/struct", &s)
	assert.Nil(t, err)
	assert.Equal(t, s.Field2, int64(10))
}
