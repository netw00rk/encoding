package etcd

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"golang.org/x/net/context"

	"github.com/coreos/etcd/client"
)

type Encoder interface {
	Encode(string, interface{}) error
}

type encoder struct {
	client client.KeysAPI
}

func NewEncoder(client client.KeysAPI) Encoder {
	return &encoder{
		client: client,
	}
}

func (e *encoder) Encode(path string, v interface{}) error {
	return e.encode(path, reflect.ValueOf(v))
}

func (e *encoder) encode(path string, value reflect.Value) error {

	switch value.Kind() {
	case reflect.Interface:
		return e.encode(path, value.Elem())

	case reflect.Struct:
		return e.encodeStruct(path, value)

	case reflect.Map:
		return e.encodeMap(path, value)

	case reflect.Slice:
		return e.encodeSlice(path, value)

	case reflect.Ptr:
		return e.encode(path, value.Elem())

	default:
		s, err := valueToString(value)
		if err != nil {
			return err
		}

		_, err = e.client.Set(context.Background(), path, s, &client.SetOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *encoder) encodeStruct(path string, value reflect.Value) error {
	for i := 0; i < value.NumField(); i++ {
		typeField := value.Type().Field(i)
		name := typeField.Tag.Get("etcd")
		if name == "" {
			name = typeField.Name
		}
		if err := e.encode(fmt.Sprintf("%s/%s", path, name), value.Field(i)); err != nil {
			return err
		}
	}

	return nil
}

func (e *encoder) encodeMap(path string, value reflect.Value) error {
	for _, key := range value.MapKeys() {
		v := value.MapIndex(key)
		if err := e.encode(fmt.Sprintf("%s/%s", path, key.String()), v); err != nil {
			return err
		}
	}

	return nil
}

func (e *encoder) encodeSlice(path string, value reflect.Value) error {
	for i := 0; i < value.Len(); i++ {
		if err := e.encode(fmt.Sprintf("%s/%d", path, i), value.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func valueToString(val reflect.Value) (string, error) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(val.Uint(), 10), nil

	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'g', -1, 64), nil

	case reflect.String:
		return val.String(), nil

	case reflect.Bool:
		if val.Bool() {
			return "true", nil
		} else {
			return "false", nil
		}
	default:
		return "", errors.New(fmt.Sprintf("can't encode type %s", val.Type()))
	}
}
