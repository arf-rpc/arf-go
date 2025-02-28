package proto

import (
	"fmt"
	"reflect"
)

func Encode(value any) ([]byte, error) {
	if value == nil {
		return []byte{byte(TypeVoid)}, nil
	}

	t := reflect.TypeOf(value)
	v := reflect.ValueOf(value)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		if v.IsNil() {
			return []byte{byte(TypeVoid)}, nil
		}
		v = v.Elem()
		value = v.Interface()
	}

	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return EncodeBytes(value.([]uint8)), nil
		}
		return encodeArray(v)
	case reflect.String:
		return EncodeString(value.(string)), nil
	case reflect.Bool:
		return encodeBoolean(value.(bool)), nil
	case reflect.Int:
		return encodeScalar(v.Int()), nil
	case reflect.Int8:
		return encodeScalar(int8(v.Int())), nil
	case reflect.Int16:
		return encodeScalar(int16(v.Int())), nil
	case reflect.Int32:
		return encodeScalar(int32(v.Int())), nil
	case reflect.Int64:
		return encodeScalar(v.Int()), nil
	case reflect.Uint:
		return encodeScalar(uint64(v.Uint())), nil
	case reflect.Uint8:
		return encodeScalar(uint8(v.Uint())), nil
	case reflect.Uint16:
		return encodeScalar(uint16(v.Uint())), nil
	case reflect.Uint32:
		return encodeScalar(uint32(v.Uint())), nil
	case reflect.Uint64:
		return encodeScalar(uint64(v.Uint())), nil
	case reflect.Float32:
		return encodeFloat32(float32(v.Float())), nil
	case reflect.Float64:
		return encodeFloat64(v.Float()), nil
	case reflect.Interface, reflect.Struct:
		return encodeStruct(v)
	case reflect.Map:
		return encodeMap(v)

	default:
		return nil, fmt.Errorf("cannot Encode value of type %s", t.Kind())
	}
}
