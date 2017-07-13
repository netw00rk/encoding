package etcd

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	//	"golang.org/x/net/context"

	"github.com/coreos/etcd/client"
)

type Encoder interface {
	Encode(string, interface{}) error
	EncodeWithContext(string, interface{}, context.Context) error
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
	return e.EncodeWithContext(path, v, context.Background())
}

func (e *encoder) EncodeWithContext(path string, v interface{}, ctx context.Context) error {
	return e.encode(path, reflect.ValueOf(v), ctx)
}

func (e *encoder) encode(path string, value reflect.Value, ctx context.Context) error {

	switch value.Kind() {
	case reflect.Interface:
		return e.encode(path, value.Elem(), ctx)

	case reflect.Struct:
		e.deleteNode(path, ctx)
		return e.encodeStruct(path, value, ctx)

	case reflect.Map:
		e.deleteNode(path, ctx)
		return e.encodeMap(path, value, ctx)

	case reflect.Slice:
		e.deleteNode(path, ctx)
		return e.encodeSlice(path, value, ctx)

	case reflect.Ptr:
		return e.encode(path, value.Elem(), ctx)

	default:
		s, err := valueToString(value)
		if err != nil {
			return err
		}

		op, ok := ctx.Value("options").(*client.SetOptions)
		if !ok {
			op = &client.SetOptions{}
		}

		_, err = e.client.Set(ctx, path, s, op)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *encoder) encodeStruct(path string, value reflect.Value, ctx context.Context) error {
	for i := 0; i < value.NumField(); i++ {
		typeField := value.Type().Field(i)
		name := typeField.Tag.Get("etcd")
		if name != "-" {
			if name == "" {
				name = typeField.Name
			}
			if err := e.encode(fmt.Sprintf("%s/%s", path, name), value.Field(i), ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *encoder) encodeMap(path string, value reflect.Value, ctx context.Context) error {
	for _, key := range value.MapKeys() {
		v := value.MapIndex(key)
		if err := e.encode(fmt.Sprintf("%s/%s", path, key.String()), v, ctx); err != nil {
			return err
		}
	}

	return nil
}

func (e *encoder) encodeSlice(path string, value reflect.Value, ctx context.Context) error {
	for i := 0; i < value.Len(); i++ {
		if err := e.encode(fmt.Sprintf("%s/%d", path, i), value.Index(i), ctx); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) deleteNode(path string, ctx context.Context) {
	opt := &client.DeleteOptions{
		Recursive: true,
		Dir:       true,
	}
	e.client.Delete(ctx, path, opt)
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
