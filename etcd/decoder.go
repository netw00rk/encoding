package etcd

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

type Decoder interface {
	Decode(string, interface{}) error
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
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Ptr {
		return errors.New("destination has to be a pointer")
	}

	return d.decode(path, value.Elem())
}

func (d *decoder) decode(path string, value reflect.Value) error {
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
		d.decode(path, v)
		value.Set(v)

	case reflect.Struct:
		if err := d.decodeStruct(path, value); err != nil {
			return err
		}

	case reflect.Map:
		if err := d.decodeMap(path, value); err != nil {
			return err
		}

	case reflect.Slice, reflect.Array:
		if err := d.decodeSlice(path, value); err != nil {
			return err
		}

	default:
		r, err := d.client.Get(context.Background(), path, &client.GetOptions{})
		if err != nil {
			if d.skipMissing && err.(client.Error).Code == client.ErrorCodeKeyNotFound {
				return nil
			}
			return err
		}
		return decodePrimitive(r.Node.Value, value)
	}

	return nil
}

func (d *decoder) decodeSlice(path string, value reflect.Value) error {
	r, _ := d.client.Get(context.Background(), path, &client.GetOptions{})
	if !r.Node.Dir {
		return errors.New(fmt.Sprintf("%s is not a dir", path))
	}

	if value.IsNil() {
		value.Set(reflect.MakeSlice(value.Type(), value.Len(), value.Cap()))
	}

	for _, node := range r.Node.Nodes {
		sliceValue := reflect.New(value.Type().Elem()).Elem()
		if err := d.decode(node.Key, sliceValue); err != nil {
			return err
		}
		value.Set(reflect.Append(value, sliceValue))
	}

	return nil
}

func (d *decoder) decodeMap(path string, value reflect.Value) error {
	r, _ := d.client.Get(context.Background(), path, &client.GetOptions{})
	if !r.Node.Dir {
		return errors.New(fmt.Sprintf("%s is not a dir", path))
	}

	if value.IsNil() {
		value.Set(reflect.MakeMap(value.Type()))
	}

	for _, node := range r.Node.Nodes {
		mapValue := reflect.New(value.Type().Elem()).Elem()
		if err := d.decode(node.Key, mapValue); err != nil {
			return err
		}

		mapKey := reflect.New(value.Type().Key()).Elem()
		p := strings.Split(node.Key, "/")
		mapKey.SetString(p[len(p)-1])
		value.SetMapIndex(mapKey, mapValue)
	}

	return nil
}

func (d *decoder) decodeStruct(path string, value reflect.Value) error {
	for i := 0; i < value.NumField(); i++ {
		typeField := value.Type().Field(i)
		name := typeField.Tag.Get("etcd")
		if name != "-" {
			if name == "" {
				name = typeField.Name
			}
			if err := d.decode(fmt.Sprintf("%s/%s", path, name), value.Field(i)); err != nil {
				return err
			}
		}
	}

	return nil
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
