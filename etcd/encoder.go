package etcd

import (
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"

	//	"golang.org/x/net/context"

	"github.com/coreos/etcd/client"
)

var (
	jsonMarshalerType = reflect.TypeOf(new(json.Marshaler)).Elem()
	textMarshalerType = reflect.TypeOf(new(encoding.TextMarshaler)).Elem()
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

func (e *encoder) indirect(v reflect.Value) (json.Marshaler, encoding.TextMarshaler) {
	t := v.Type()
	if t.Implements(jsonMarshalerType) {
		return v.Interface().(json.Marshaler), nil
	}

	if t.Kind() != reflect.Ptr && v.CanAddr() {
		if reflect.PtrTo(t).Implements(jsonMarshalerType) {
			return v.Interface().(json.Marshaler), nil
		}
	}

	if t.Implements(textMarshalerType) {
		return nil, v.Interface().(encoding.TextMarshaler)
	}

	if t.Kind() != reflect.Ptr && v.CanAddr() {
		if reflect.PtrTo(t).Implements(textMarshalerType) {
			va := v.Addr()
			if !va.IsNil() {
				return nil, v.Interface().(encoding.TextMarshaler)
			}
		}
	}

	return nil, nil
}

func (e *encoder) encode(path string, value reflect.Value, ctx context.Context) error {
	m, tm := e.indirect(value)
	if m != nil {
		return e.encodeMarshaler(m, path, ctx)
	}

	if tm != nil {
		return e.encodeTextMarshaler(tm, path, ctx)
	}

	switch value.Kind() {
	case reflect.Interface:
		return e.encode(path, value.Elem(), ctx)

	case reflect.Struct:
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

		if err := e.setNode(path, s, ctx); err != nil {
			return err
		}
	}

	return nil
}

func (e *encoder) encodeMarshaler(u json.Marshaler, path string, ctx context.Context) error {
	s, err := u.MarshalJSON()
	if err != nil {
		return err
	}

	return e.setNode(path, string(s), ctx)
}

func (e *encoder) encodeTextMarshaler(u encoding.TextMarshaler, path string, ctx context.Context) error {
	s, err := u.MarshalText()
	if err != nil {
		return err
	}

	return e.setNode(path, string(s), ctx)
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

func (e *encoder) setNode(path string, value string, ctx context.Context) error {
	op, ok := ctx.Value("options").(*client.SetOptions)
	if !ok {
		op = &client.SetOptions{}
	}

	if _, err := e.client.Set(ctx, path, value, op); err != nil {
		return err
	}

	return nil
}

func valueToString(val reflect.Value) (string, error) {
	return fmt.Sprint(val), nil
}
