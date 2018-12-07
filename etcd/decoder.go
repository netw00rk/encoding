package etcd

import (
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/client"
)

type Decoder interface {
	Decode(string, interface{}) error
	DecodeWithContext(string, interface{}, context.Context) error
	SkipMissing(bool)
}

type decoderFn func(*client.Node, reflect.Value, context.Context) error

type decoder struct {
	client      client.KeysAPI
	skipMissing bool
}

func NewDecoder(client client.KeysAPI) Decoder {
	return &decoder{
		client: client,
	}
}

// Deprecated, use ",omitempty" tag
func (d *decoder) SkipMissing(skip bool) {
	log.Println("SkipMissing method is deprecated, use \",omitempty\" tag instead")
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

func (d *decoder) indirect(v reflect.Value) (json.Unmarshaler, encoding.TextUnmarshaler, reflect.Value) {
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}

	for {
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && e.Elem().Kind() == reflect.Ptr {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if v.Type().NumMethod() > 0 {
			if u, ok := v.Interface().(json.Unmarshaler); ok {
				return u, nil, reflect.Value{}
			}
			if u, ok := v.Interface().(encoding.TextUnmarshaler); ok {
				return nil, u, reflect.Value{}
			}
		}
		v = v.Elem()
	}
	return nil, nil, v
}

func (d *decoder) decode(path string, value reflect.Value, ctx context.Context) error {
	node, err := d.getNode(path, ctx)
	if node == nil {
		return err
	}

	decoder := d.decoder(value)
	return decoder(node, value, ctx)
}

func (d *decoder) decodeNode(node *client.Node, value reflect.Value, ctx context.Context) error {
	decoder := d.decoder(value)
	return decoder(node, value, ctx)
}

func (d *decoder) decoder(value reflect.Value) decoderFn {
	u, tu, value := d.indirect(value)
	if u != nil {
		return func(n *client.Node, v reflect.Value, ctx context.Context) error {
			return d.decodeUnmarshaler(u, n, ctx)
		}
	}

	if tu != nil {
		return func(n *client.Node, v reflect.Value, ctx context.Context) error {
			return d.decodeTextUnmarshaler(tu, n, ctx)
		}
	}

	switch value.Kind() {
	case reflect.Func:
		return func(n *client.Node, v reflect.Value, ctx context.Context) error {
			return errors.New("can't decode func: not implemented")
		}

	case reflect.Chan:
		return func(n *client.Node, v reflect.Value, ctx context.Context) error {
			return errors.New("can't decode channel: not implemented")
		}

	case reflect.Ptr:
		return d.decodePointer

	case reflect.Interface:
		return d.decodeInterface

	case reflect.Struct:
		return d.decodeStruct

	case reflect.Map:
		return d.decodeMap

	case reflect.Slice, reflect.Array:
		return d.decodeSlice

	default:
		return func(n *client.Node, v reflect.Value, ctx context.Context) error {
			return decodePrimitive(n.Value, v)
		}
	}

	return nil
}

func (d *decoder) decodePointer(node *client.Node, value reflect.Value, ctx context.Context) error {
	return d.decodeNode(node, value.Elem(), ctx)
}

func (d *decoder) decodeInterface(node *client.Node, value reflect.Value, ctx context.Context) error {
	v := reflect.New(value.Elem().Type()).Elem()
	if err := d.decodeNode(node, v, ctx); err != nil {
		return err
	}

	value.Set(v)
	return nil
}

func (d *decoder) decodeUnmarshaler(u json.Unmarshaler, node *client.Node, ctx context.Context) error {
	return u.UnmarshalJSON([]byte(node.Value))
}

func (d *decoder) decodeTextUnmarshaler(u encoding.TextUnmarshaler, node *client.Node, ctx context.Context) error {
	return u.UnmarshalText([]byte(node.Value))
}

func (d *decoder) decodeSlice(node *client.Node, value reflect.Value, ctx context.Context) error {
	if !node.Dir {
		return errors.New(fmt.Sprintf("%s is not a dir", node.Key))
	}

	if value.IsNil() || value.Len() == 0 {
		value.Set(reflect.MakeSlice(value.Type(), len(node.Nodes), len(node.Nodes)))
	}

	for _, node := range node.Nodes {
		sliceValue := reflect.New(value.Type().Elem()).Elem()
		if node.Dir {
			if err := d.decode(node.Key, sliceValue, ctx); err != nil {
				return err
			}
		} else {
			if err := d.decodeNode(node, sliceValue, ctx); err != nil {
				return err
			}
		}

		tmp := strings.Split(node.Key, "/")
		index, _ := strconv.Atoi(tmp[len(tmp)-1])

		el := value.Index(index)
		el.Set(sliceValue)
	}

	return nil
}

func (d *decoder) decodeMap(node *client.Node, value reflect.Value, ctx context.Context) error {
	if !node.Dir {
		return errors.New(fmt.Sprintf("%s is not a dir", node.Key))
	}

	if value.IsNil() {
		value.Set(reflect.MakeMap(value.Type()))
	}

	for _, node := range node.Nodes {
		mapValue := reflect.New(value.Type().Elem()).Elem()
		if node.Dir {
			if err := d.decode(node.Key, mapValue, ctx); err != nil {
				return err
			}
		} else {
			if err := d.decodeNode(node, mapValue, ctx); err != nil {
				return err
			}
		}

		mapKey := reflect.New(value.Type().Key()).Elem()
		p := strings.Split(node.Key, "/")
		decodePrimitive(p[len(p)-1], mapKey)
		value.SetMapIndex(mapKey, mapValue)
	}

	return nil
}

func (d *decoder) decodeStruct(top *client.Node, value reflect.Value, ctx context.Context) error {
	if !top.Dir {
		return errors.New(fmt.Sprintf("%s is not a dir", top.Key))
	}

	nodes := make(map[string]*client.Node)
	for _, node := range top.Nodes {
		p := strings.Split(node.Key, "/")
		nodes[p[len(p)-1]] = node
	}

	for i := 0; i < value.NumField(); i++ {
		typeField := value.Type().Field(i)
		name := typeField.Name
		tag := typeField.Tag.Get("etcd")
		if tag != "-" {
			params := strings.Split(tag, ",")
			if len(params) > 0 && params[0] != "" {
				name = params[0]
			}

			node, ok := nodes[name]
			if !ok {
				if isOmitEmpty(params) {
					continue
				}
				return fmt.Errorf("Key %s not found", fmt.Sprintf("%s/%s", top.Key, name))
			}

			if node.Dir {
				if err := d.decode(node.Key, value.Field(i), ctx); err != nil {
					if !canOmitEmpty(err, params) {
						return err
					}
				}
			} else {
				if err := d.decodeNode(node, value.Field(i), ctx); err != nil {
					return err
				}
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
		switch value.Interface().(type) {
		case time.Duration:
			v, err := time.ParseDuration(nodeValue)
			if err != nil {
				return err
			}
			value.SetInt(int64(v))

		default:
			v, err := strconv.ParseInt(nodeValue, 10, 64)
			if err != nil {
				return err
			}
			value.SetInt(v)
		}

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

func isOmitEmpty(params []string) bool {
	if len(params) < 2 {
		return false
	}

	return params[1] == "omitempty"
}

func canOmitEmpty(err error, params []string) bool {
	if e, ok := err.(client.Error); ok && isOmitEmpty(params) && e.Code == client.ErrorCodeKeyNotFound {
		return true
	}

	return false
}

func canSkipMissing(err error, skipFlag bool) bool {
	if e, ok := err.(client.Error); ok && skipFlag && e.Code == client.ErrorCodeKeyNotFound {
		return true
	}

	return false
}
