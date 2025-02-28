package proto

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"unsafe"
)

type Struct interface {
	ArfStructID() string
}

var typeOfStructInterface = reflect.TypeOf((*Struct)(nil)).Elem()

type encodableField struct {
	field reflect.StructField
	index int
}

func encodableFieldsFromType(t reflect.Type) ([]encodableField, error) {
	if !t.Implements(typeOfStructInterface) {
		return nil, fmt.Errorf("%s does not implement %s",
			t.String(), typeOfStructInterface.String())
	}
	var fields []encodableField
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		rawID, ok := f.Tag.Lookup("arf")
		if !ok {
			continue
		}

		id, err := strconv.Atoi(rawID)
		if err != nil {
			return nil, err
		}
		fields = append(fields, encodableField{
			field: f,
			index: id,
		})
	}

	slices.SortFunc(fields, func(a, b encodableField) int {
		return cmp.Compare(a.index, b.index)
	})

	return fields, nil
}

func fieldsFromUnionStruct(f reflect.Type) (encodableFieldSet, error) {
	if f.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected %s to be a struct", f.String())
	}

	fields := make([]encodableField, 0, f.NumField())
	for i := range f.NumField() {
		f := f.Field(i)
		val, ok := f.Tag.Lookup("arf")
		if !ok {
			continue
		}
		if val == "union" {
			return nil, fmt.Errorf("nested unions are not supported")
		}
		id, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}

		fields = append(fields, encodableField{
			field: f,
			index: id,
		})
	}

	return fields, nil

}

func encodeStruct(v reflect.Value) ([]byte, error) {
	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	if !t.Implements(typeOfStructInterface) {
		return nil, fmt.Errorf("%s does not implement %s",
			t.String(), typeOfStructInterface.String())
	}

	structID := v.Interface().(Struct).ArfStructID()
	fields, err := encodableFieldsFromType(t)
	if err != nil {
		return nil, err
	}

	var data [][]byte
	for _, f := range fields {
		data = append(data, encodeUint64(uint64(f.index)))
		buf, err := Encode(v.FieldByIndex(f.field.Index).Interface())
		if err != nil {
			return nil, err
		}
		data = append(data, buf)
	}

	payload := bytes.Join(data, nil)

	return bytes.Join([][]byte{
		{byte(TypeStruct)},
		EncodeString(structID),
		encodeUint64(uint64(len(payload))),
		payload,
	}, nil), nil
}

type decodedStruct struct {
	id     string
	fields map[int]any
}

func decodeStruct(r io.Reader) (any, error) {
	t, b, err := readType(r)
	if err != nil {
		return nil, err
	}
	if t != TypeString {
		return nil, fmt.Errorf("cannot decode struct: expected string, found %s instead", t.String())
	}
	id, err := decodeString(b, r)
	if err != nil {
		return nil, err
	}
	bytesLen, err := decodeUint64(r)
	if err != nil {
		return nil, err
	}

	reader := io.LimitReader(r, int64(bytesLen))
	fields := map[int]any{}
	for {
		i, err := decodeUint64(reader)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		v, err := DecodeAny(reader)
		if err != nil {
			return nil, err
		}
		fields[int(i)] = v
	}

	return decodeIntoInstance(decodedStruct{
		id:     id,
		fields: fields,
	})
}

func decodeIntoInstance(d decodedStruct) (any, error) {
	t, ok := reg.structs[d.id]
	if !ok {
		return nil, fmt.Errorf("unknown message kind %s", d.id)
	}

	res := reflect.New(t.structType)
	inst := res.Elem()

	for i, v := range d.fields {
		f, ok := t.fields.fieldByIndex(i)
		if !ok {
			break
		}

		setValue(inst, f.field, v)
	}

	return res.Interface(), nil
}

func setStructField(into reflect.Value, fd reflect.StructField, value reflect.Value) {
	if !fd.IsExported() {
		fieldPtr := unsafe.Pointer(into.UnsafeAddr() + fd.Offset)
		fieldValuePtr := reflect.NewAt(fd.Type, fieldPtr)
		fieldValuePtr.Elem().Set(value)
	} else {
		into.FieldByIndex(fd.Index).Set(value)
	}
}

func setValue(into reflect.Value, fd reflect.StructField, value interface{}) {
	var rv reflect.Value
	if v, ok := value.(reflect.Value); ok {
		rv = v
	} else {
		rv = reflect.ValueOf(value)
	}

	switch {
	case fd.Type.Kind() == reflect.Pointer && !rv.IsValid():
		// nil for a pointer, there's not much to do here. This case is only
		// here to prevent the switch from going into the default case.
		// Feel free to rest at this bonfire, traveller.
		//                          xg,
		//                         1 I
		//                         #F`
		//                        ,#6
		//                        k#
		//                       ,k!
		//                       [E
		//                      ,N&
		//                  *""NR#==~`
		//                     [M}
		//                     W0
		//                   ,Q#!    **
		//               *   44H    **
		//               **   B0  ****
		//      *  *      **  [Q ****
		//     *** **    ***********
		//      ******  *********     **
		//      ******  *****HA     ****
		//        *****  ***;AD   *****
		//        ***** *** jN#  ***
		//         **   ****NN5 *** *
		//          *** ****N#** ***
		//           **  **]08*********
		//           ******##8** ****
		//             *** ## *****
		//             ***|&8 * **
		//         ^,E&***lNH****-m&
		//          ^mF***BM **y0*DQ
		//        ,p$&Ep_?MD **K&~~WE,
		//       y0&M 060~GURU`"~&&Q&WI,
		//      ~~'hNH7~ 2&mKk$KwK40Q$~*+
		//         #YmbdB##EMQGW&N6Nx
		//          *N&E&WB08NNH#6r6
		//               ^  ~~""^

	case fd.Type.Kind() != reflect.Pointer &&
		rv.Type().Kind() == reflect.Pointer &&
		rv.Elem().Type().ConvertibleTo(fd.Type):
		setStructField(into, fd, rv.Elem().Convert(fd.Type))

	case fd.Type.Kind() == reflect.Pointer &&
		rv.Type().Kind() != reflect.Pointer &&
		rv.Type().ConvertibleTo(fd.Type.Elem()):
		ptrVal := reflect.New(fd.Type.Elem())
		ptrVal.Elem().Set(rv.Convert(fd.Type.Elem()))
		setStructField(into, fd, ptrVal)

	case rv.Type().ConvertibleTo(fd.Type):
		setStructField(into, fd, rv.Convert(fd.Type))

	case rv.Type().Kind() == reflect.Slice && fd.Type.Kind() == reflect.Slice:
		// rv is []interface, f is specialised. Check if rv[i] can be
		// convertible to f[i]. Bear in mind that slices do not take optional
		// values, so no pointers here.
		ft := fd.Type.Elem()
		mustConvertPtrs := false
		if rv.Len() > 0 {
			mustConvertPtrs = rv.Index(0).Elem().Type().Kind() == reflect.Pointer
		}
		if !mustConvertPtrs {
			for i := 0; i < rv.Len(); i++ {
				if !rv.Index(i).Elem().Type().ConvertibleTo(ft) {
					return
				}
			}
		} else {
			for i := 0; i < rv.Len(); i++ {
				if !rv.Index(i).Elem().Elem().Type().ConvertibleTo(ft) {
					return
				}
			}
		}

		// All items can be converted. Initialise a new slice, fill it, and
		// set the field value.
		slice := reflect.MakeSlice(reflect.SliceOf(ft), rv.Len(), rv.Len())
		if mustConvertPtrs {
			for i := 0; i < rv.Len(); i++ {
				v := rv.Index(i).Elem().Elem()
				slice.Index(i).Set(v.Convert(ft))
			}
		} else {
			for i := 0; i < rv.Len(); i++ {
				v := rv.Index(i).Elem()
				slice.Index(i).Set(v.Convert(ft))
			}
		}
		setStructField(into, fd, slice)

	case fd.Type.Kind() == rv.Type().Kind():
		setStructField(into, fd, rv)

	case rv.Type().Kind() == reflect.Pointer &&
		fd.Type.Kind() == reflect.Map &&
		rv.Type() == reflectedMapValue:
		mv := rv.Interface().(*encodedMap)
		ok, mi := makeMap(mv, fd.Type)
		if !ok {
			return
		}
		setStructField(into, fd, mi)

	case rv.Type().Kind() == reflect.Pointer &&
		rv.Type().Elem().Kind() == reflect.Struct &&
		!rv.IsNil() &&
		fd.Type.Kind() == reflect.Struct:
		// rv is a pointer to struct coming from Decode, but we want a concrete
		// value.
		setValue(into, fd, rv.Elem())

	default:
		return
	}
}

func isConvertible(from, to reflect.Type) bool {
	if from.ConvertibleTo(to) {
		return true
	}

	if from.Kind() == reflect.Ptr && from.Elem().ConvertibleTo(to) {
		return true
	} else if to.Kind() == reflect.Ptr && from.ConvertibleTo(to.Elem()) {
		return true
	}

	return false
}

func pointerTo(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		return v
	}
	if !v.CanAddr() {
		ptr := reflect.New(v.Type()) // Create a pointer to a value of the same type
		ptr.Elem().Set(v)            // Set the value
		return ptr
	}
	return v.Addr() // If addressable, just return the address
}

func convertTo(from reflect.Value, to reflect.Type) reflect.Value {
	// If a direct conversion is possible, just do it
	if from.Type().ConvertibleTo(to) {
		return from.Convert(to)
	}
	if from.Type().Kind() == reflect.Ptr && to.Kind() != reflect.Ptr {
		return from.Elem()
	} else if from.Type().Kind() != reflect.Ptr && to.Kind() == reflect.Ptr {
		return pointerTo(from)
	}
	return from
}

func makeMap(v *encodedMap, mapType reflect.Type) (bool, reflect.Value) {
	mapKeyType := mapType.Key()
	mapValueType := mapType.Elem()
	var mi reflect.Value
	if v != nil {
		for _, v := range v.keys {
			if !isConvertible(reflect.TypeOf(v), mapKeyType) {
				return false, reflect.Value{}
			}
		}

		for _, v := range v.values {
			if !isConvertible(reflect.TypeOf(v), mapValueType) {
				return false, reflect.Value{}
			}
		}
		mi = reflect.MakeMapWithSize(mapType, len(v.keys))
		for i, rk := range v.keys {
			k := convertTo(reflect.ValueOf(rk), mapKeyType)
			v := convertTo(reflect.ValueOf(v.values[i]), mapValueType)
			mi.SetMapIndex(k, v)
		}
	} else {
		mi = reflect.MakeMapWithSize(mapType, 0)
	}

	return true, mi
}
