
Golang library for encoding/decoding data from/to [etcd](https://github.com/coreos/etcd).
It supports primitive data types, structs, slices, map with string keys.

### Usage example

```go
import (
    "github.com/coreos/etcd/client"
    "github.com/netw00rk/encoding/etcd"
)

type NestedComplexStruct struct {
	BooleanField  bool `etcd:"boolean_field"`
	IntMapField   map[string]int
	IntSliceField []int
}

type ComplexStruct struct {
	IntField     int
	Int64Field   int64
	Float32Field float32
	Float64Field float64
	StringField  string
	StructField  NestedComplexStruct
}

func main() {
    // ... init etcd keysApi client

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
	}

	encoder := NewEncoder(keysApi)
	err := encoder.Encode("/path/to/struct", a)
    if err != nil {
        panic(err)
    }

    // Result wil be
    // /path/to/struct/IntField = "10"
    // /path/to/struct/Int64Field = "20"
    // /path/to/struct/Float32Field = "10.5"
    // /path/to/struct/Float64Field = "20.5"
    // /path/to/struct/StringField = "value"
    // /path/to/struct/StructField/boolean_field = "true"
    // /path/to/struct/StructField/IntMapField/field_1 = "30"
    // /path/to/struct/StructField/IntMapField/field_2 = "40"
    // /path/to/struct/StructField/IntSliceField/0 = "50"
    // /path/to/struct/StructField/IntSliceField/1 = "60"

    var b = new(ComplexStruct)
    decoder := etcd.Decoder(keysApi)
    err = decoder.Decode("/path/to/struct", b)
    if err != nil {
        panic(err)
    }
}
```

To skip field during encoding use `etcd:"-"` tag.
To skip missing fields during decoding use `etcd:",omitempty"` tag;
