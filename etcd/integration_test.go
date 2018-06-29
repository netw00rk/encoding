package etcd

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/coreos/etcd/client"
	"github.com/stretchr/testify/assert"
)

const ETCD_TEST_KEY = "/etcd/integration/test"

type StructWithMarshaller struct {
	Field string
}

func (c StructWithMarshaller) MarshalJSON() ([]byte, error) {
	return []byte(c.Field), nil
}

func (c *StructWithMarshaller) UnmarshalJSON(data []byte) error {
	c.Field = string(data)
	return nil
}

type NestedComplexStruct struct {
	BooleanField  bool `etcd:"boolean_field"`
	IntMapField   map[string]int
	IntSliceField []int
}

type ComplexStruct struct {
	IntField          int
	Int64Field        int64
	Float32Field      float32
	Float64Field      float64
	StringField       string
	StructField       NestedComplexStruct
	TimeDurationField time.Duration
	WithMarshaller    StructWithMarshaller
}

func getKeysApi() client.KeysAPI {
	c, err := client.New(client.Config{
		Endpoints: []string{"http://localhost:2379"},
		Transport: client.DefaultTransport,
	})
	if err != nil {
		panic(err)
	}
	return client.NewKeysAPI(c)
}

func testEncodeStruct(keysApi client.KeysAPI) (*ComplexStruct, error) {
	var a = ComplexStruct{
		IntField:     10,
		Int64Field:   int64(20),
		Float32Field: float32(10.5),
		Float64Field: float64(20.5),
		StringField:  "value",
		StructField: NestedComplexStruct{
			BooleanField: true,
			IntMapField: map[string]int{
				"field_1": 30,
				"field_2": 40,
			},
			IntSliceField: []int{50, 60},
		},
		TimeDurationField: time.Second * 5,
		WithMarshaller: StructWithMarshaller{
			Field: "foo",
		},
	}

	encoder := NewEncoder(keysApi)
	return &a, encoder.Encode(ETCD_TEST_KEY, a)
}

func testDecodeStruct(keysApi client.KeysAPI) (*ComplexStruct, error) {
	var b = new(ComplexStruct)
	decoder := NewDecoder(keysApi)
	return b, decoder.Decode(ETCD_TEST_KEY, b)
}

func TestIntegrationEncodingDecoding(t *testing.T) {
	keysApi := getKeysApi()

	_, err := keysApi.Delete(context.Background(), ETCD_TEST_KEY, &client.DeleteOptions{Recursive: true})
	assert.Nil(t, err)

	var a *ComplexStruct
	a, err = testEncodeStruct(keysApi)
	assert.Nil(t, err)

	var b *ComplexStruct
	b, err = testDecodeStruct(keysApi)
	assert.Nil(t, err)
	assert.EqualValues(t, a, b)
}

func BenchmarkEncodingDecoding(b *testing.B) {
	keysApi := getKeysApi()
	keysApi.Delete(context.Background(), ETCD_TEST_KEY, &client.DeleteOptions{Recursive: true})
	testEncodeStruct(keysApi)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		testDecodeStruct(keysApi)
	}
}
