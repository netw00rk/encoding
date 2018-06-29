package etcd

import (
	"testing"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/netw00rk/encoding/etcd/test"
)

type customJsonUnmarshaler struct {
	Field string
}

func (c *customJsonUnmarshaler) UnmarshalJSON(data []byte) error {
	c.Field = "unmarshalled:" + string(data)
	return nil
}

type customTextUnmarshaler struct {
	Field string
}

func (c *customTextUnmarshaler) UnmarshalText(data []byte) error {
	c.Field = "unmarshalled:" + string(data)
	return nil
}

func TestFailIfDestinationIsNotAPointer(t *testing.T) {
	decoder := NewDecoder(new(test.KeysAPIMock))
	err := decoder.Decode("/some/path", 10)
	assert.Equal(t, "destination has to be a pointer", err.Error())
}

func TestDecodePrimitiveTypes(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/value", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/value/bool", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "false"}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/value/duration", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "10s"}}, nil)
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
	err = decoder.Decode("/path/to/some/value/duration", &g)
	assert.Nil(t, err)
	assert.Equal(t, time.Second*10, g)
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
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir: true,
		Nodes: []*client.Node{
			&client.Node{Key: "/path/to/some/struct/Field1", Value: "10"},
			&client.Node{Key: "/path/to/some/struct/Field2", Value: "string"},
			&client.Node{Key: "/path/to/some/struct/Field3", Value: "true"},
		},
	}}, nil)

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
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir: true,
		Nodes: []*client.Node{
			&client.Node{Key: "/path/to/some/struct/Field1", Value: "10"},
			&client.Node{Key: "/path/to/some/struct/Field2", Dir: true},
		},
	}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Field2", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir: true,
		Nodes: []*client.Node{
			&client.Node{Key: "/path/to/some/struct/Field2/Field1", Value: "string"},
			&client.Node{Key: "/path/to/some/struct/Field2/Field2", Value: "true"},
		},
	}}, nil)

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
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir: true,
		Nodes: []*client.Node{
			&client.Node{Key: "/path/to/some/struct/field_1", Value: "10"},
			&client.Node{Key: "/path/to/some/struct/Field2", Value: "20"},
		},
	}}, nil)

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
		Nodes: []*client.Node{&client.Node{Key: "/path/to/some/struct/field_1", Value: "10"}, &client.Node{Key: "/path/to/some/struct/field_2", Value: "20"}},
	}}, nil)

	var m map[string]int

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/struct", &m)
	assert.Nil(t, err)
	assert.Equal(t, 10, m["field_1"])
	assert.Equal(t, 20, m["field_2"])
}

func TestDecodeMapWithIntKeys(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir:   true,
		Nodes: []*client.Node{&client.Node{Key: "/path/to/some/struct/10", Value: "10"}, &client.Node{Key: "/path/to/some/struct/20", Value: "20"}},
	}}, nil)

	var m map[int]int

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/struct", &m)
	assert.Nil(t, err)
	assert.Equal(t, 10, m[10])
	assert.Equal(t, 20, m[20])
}

func TestDecodeSimpleMapInsideStruct(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir: true,
		Nodes: []*client.Node{
			&client.Node{Key: "/path/to/some/struct/Map", Dir: true},
		},
	}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct/Map", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir:   true,
		Nodes: []*client.Node{&client.Node{Key: "/path/to/some/struct/Map/field_1", Value: "10"}, &client.Node{Key: "/path/to/some/struct/Map/field_2", Value: "30"}},
	}}, nil)

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
		Nodes: []*client.Node{&client.Node{Key: "/path/to/some/map/field_1", Dir: true}, &client.Node{Key: "/path/to/some/map/field_2", Dir: true}},
	}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/map/field_1", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir: true,
		Nodes: []*client.Node{
			&client.Node{Key: "/path/to/some/struct/field_1/Field1", Value: "10"},
		},
	}}, nil)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/map/field_2", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir: true,
		Nodes: []*client.Node{
			&client.Node{Key: "/path/to/some/struct/field_1/Field1", Value: "30"},
		},
	}}, nil)

	var m map[string]struct {
		Field1 int64
	}

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/map", &m)
	assert.Nil(t, err)
	assert.Equal(t, int64(10), m["field_1"].Field1)
	assert.Equal(t, int64(30), m["field_2"].Field1)
}

func TestDecodeJsonUnmarshaller(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/custom", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "value"}}, nil)

	c := new(customJsonUnmarshaler)
	c.UnmarshalJSON([]byte("hello"))
	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/custom", &c)
	assert.Nil(t, err)
	assert.Equal(t, "unmarshalled:value", c.Field)
}

func TestDecodeTextUnmarshaller(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/custom", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{Value: "value"}}, nil)

	c := customTextUnmarshaler{}
	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/custom", &c)
	assert.Nil(t, err)
	assert.Equal(t, "unmarshalled:value", c.Field)
}

func TestDecodeSimpleSlice(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/slice", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir: true,
		Nodes: []*client.Node{
			&client.Node{Key: "/path/to/slice/1", Value: "10"},
			&client.Node{Key: "/path/to/slice/2", Value: "30"},
			&client.Node{Key: "/path/to/slice/3", Value: "50"},
		},
	}}, nil)

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
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir: true,
		Nodes: []*client.Node{
			&client.Node{Key: "/path/to/some/struct/field", Value: "10"},
		},
	}}, nil)

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

func TestDecodeOmitEmptyError(t *testing.T) {
	etcd := new(test.KeysAPIMock)
	etcd.On("Get", mock.AnythingOfType("*context.emptyCtx"), "/path/to/some/struct", mock.AnythingOfType("*client.GetOptions")).Return(&client.Response{Node: &client.Node{
		Dir: true,
		Nodes: []*client.Node{
			&client.Node{Key: "/path/to/some/struct/Field2", Value: "10"},
		},
	}}, nil)

	var s = struct {
		Field  int64 `etcd:",omitempty"`
		Field2 int64
	}{}

	decoder := NewDecoder(etcd)
	err := decoder.Decode("/path/to/some/struct", &s)
	assert.Nil(t, err)
	assert.Equal(t, s.Field2, int64(10))
}
