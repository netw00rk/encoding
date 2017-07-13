package etcd

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/coreos/etcd/client"
)

type Decoder interface {
	Decode(string, interface{}) error
	DecodeWithContext(string, interface{}, context.Context) error
	SkipMissing(bool)
}

type decoder struct {
	client      client.KeysAPI
	skipMissing bool
}

func NewDecoder(client client.KeysAPI) Decoder {
	return &decoder{
		client: client,
	}
}

func (d *decoder) SkipMissing(skip bool) {
	d.skipMissing = skip
}

func (d *decoder) Decode(path string, v interface{}) error {
	return d.DecodeWithContext(path, v, context.Background())
}

func (d *decoder) DecodeWithContext(path string, v interface{}, ctx context.Context) error {
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Ptr {
		return errors.New("destination has to be a pointer")
	}

	return d.decode(path, value.Elem(), ctx)
}

func (d *decoder) decode(path string, value reflect.Value, ctx context.Context) error {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Func:
		return errors.New("can't decode func: not implemented")

	case reflect.Chan:
		return errors.New("can't decode channel: not implemented")

	case reflect.Interface:
		v := reflect.New(value.Elem().Type()).Elem()
		d.decode(path, v, ctx)
		value.Set(v)

	case reflect.Struct:
		if err := d.decodeStruct(path, value, ctx); err != nil {
			return err
		}

	case reflect.Map:
		if err := d.decodeMap(path, value, ctx); err != nil {
			return err
		}

	case reflect.Slice, reflect.Array:
		if err := d.decodeSlice(path, value, ctx); err != nil {
			return err
		}

	default:
		node, err := d.getNode(path, ctx)
		if node == nil {
			return err
		}

		return decodePrimitive(node.Value, value)
	}

	return nil
}

func (d *decoder) decodeSlice(path string, value reflect.Value, ctx context.Context) error {
	node, err := d.getNode(path, ctx)
	if node == nil {
		return err
	}

	if !node.Dir {
		return errors.New(fmt.Sprintf("%s is not a dir", path))
	}

	if value.IsNil() {
		value.Set(reflect.MakeSlice(value.Type(), value.Len(), value.Cap()))
	}

	for _, node := range node.Nodes {
		sliceValue := reflect.New(value.Type().Elem()).Elem()
		if err := d.decode(node.Key, sliceValue, ctx); err != nil {
			return err
		}
		value.Set(reflect.Append(value, sliceValue))
	}

	return nil
}

func (d *decoder) decodeMap(path string, value reflect.Value, ctx context.Context) error {
	node, err := d.getNode(path, ctx)
	if node == nil {
		return err
	}

	if !node.Dir {
		return errors.New(fmt.Sprintf("%s is not a dir", path))
	}

	if value.IsNil() {
		value.Set(reflect.MakeMap(value.Type()))
	}

	for _, node := range node.Nodes {
		mapValue := reflect.New(value.Type().Elem()).Elem()
		if err := d.decode(node.Key, mapValue, ctx); err != nil {
			return err
		}

		mapKey := reflect.New(value.Type().Key()).Elem()
		p := strings.Split(node.Key, "/")
		mapKey.SetString(p[len(p)-1])
		value.SetMapIndex(mapKey, mapValue)
	}

	return nil
}

func (d *decoder) decodeStruct(path string, value reflect.Value, ctx context.Context) error {
	for i := 0; i < value.NumField(); i++ {
		typeField := value.Type().Field(i)
		name := typeField.Tag.Get("etcd")
		if name != "-" {
			if name == "" {
				name = typeField.Name
			}
			if err := d.decode(fmt.Sprintf("%s/%s", path, name), value.Field(i), ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *decoder) getNode(path string, ctx context.Context) (*client.Node, error) {
	op, ok := ctx.Value("options").(*client.GetOptions)
	if !ok {
		op = &client.GetOptions{}
	}

	r, err := d.client.Get(ctx, path, op)
	if err != nil {
		if canSkipMissing(err, d.skipMissing) {
			return nil, nil
		}
		return nil, err
	}
	return r.Node, nil
}

func decodePrimitive(nodeValue string, value reflect.Value) error {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(nodeValue, 10, 64)
		if err != nil {
			return err
		}
		value.SetInt(v)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v, err := strconv.ParseUint(nodeValue, 10, 64)
		if err != nil {
			return err
		}
		value.SetUint(v)

	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(nodeValue, 64)
		if err != nil {
			return err
		}
		value.SetFloat(v)

	case reflect.Bool:
		v, err := strconv.ParseBool(nodeValue)
		if err != nil {
			return err
		}
		value.SetBool(v)
	case reflect.String:
		value.SetString(nodeValue)
	default:
		return errors.New(fmt.Sprintf("can't decode value of type %s", value.Type()))
	}

	return nil
}

func canSkipMissing(err error, skipFlag bool) bool {
	if e, ok := err.(client.Error); ok && skipFlag && e.Code == client.ErrorCodeKeyNotFound {
		return true
	}

	return false
}
