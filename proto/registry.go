package proto

import (
	"fmt"
	"reflect"
)

type encodableFieldSet []encodableField

func (e encodableFieldSet) fieldByIndex(idx int) (*encodableField, bool) {
	for _, v := range e {
		if v.index == idx {
			return &v, true
		}
	}
	return nil, false
}

type knownStructType struct {
	id         string
	structType reflect.Type
	fields     encodableFieldSet
}

type registry struct {
	structs map[string]knownStructType
}

func RegisterMessage(s Struct) {
	id := s.ArfStructID()
	t := reflect.TypeOf(s)
	f, err := encodableFieldsFromType(t)
	if err != nil {
		panic(fmt.Sprintf("Failed to register arf struct: %s", err.Error()))
	}

	reg.structs[id] = knownStructType{
		id:         id,
		structType: t,
		fields:     f,
	}
}

func resetRegistry() {
	reg.structs = map[string]knownStructType{}
}
