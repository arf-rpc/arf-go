package rpc

import (
	"bytes"
	proto2 "github.com/arf-rpc/arf/proto"
	"io"
	"slices"
)

func MetadataFromMap(m map[string][]byte) Metadata {
	meta := Metadata{}
	for k, v := range m {
		meta.Add(k, v)
	}
	return meta
}

func MetadataFromStringMap(m map[string]string) Metadata {
	meta := Metadata{}
	for k, v := range m {
		meta.AddString(k, v)
	}
	return meta
}

type MetadataPair struct {
	Key   string
	Value []byte
}

type Metadata []MetadataPair

func (m Metadata) Lookup(name string) (v []byte, ok bool) {
	for i := len(m) - 1; i >= 0; i-- {
		if m[i].Key == name {
			return m[i].Value, true
		}
	}
	return
}

func (m Metadata) LookupString(name string) (v string, ok bool) {
	vv, ok := m.Lookup(name)
	return string(vv), ok
}
func (m Metadata) Get(name string) []byte {
	v, _ := m.Lookup(name)
	return v
}
func (m Metadata) GetString(name string) string {
	return string(m.Get(name))
}
func (m Metadata) GetAll(name string) (v [][]byte) {
	for i := len(m) - 1; i >= 0; i-- {
		if m[i].Key == name {
			v = append(v, m[i].Value)
		}
	}
	return
}
func (m Metadata) GetAllString(name string) []string {
	allBytes := m.GetAll(name)
	allString := make([]string, len(allBytes))
	for i, b := range allBytes {
		allString[i] = string(b)
	}
	return allString
}
func (m *Metadata) Add(name string, value []byte) {
	*m = append(*m, MetadataPair{name, value})
}
func (m *Metadata) AddString(name string, value string) {
	m.Add(name, []byte(value))
}
func (m *Metadata) Set(name string, value []byte) {
	clone := slices.DeleteFunc(*m, func(s MetadataPair) bool { return s.Key == name })
	clone.Add(name, value)
	*m = clone
}
func (m *Metadata) SetString(name string, value string) {
	m.Set(name, []byte(value))
}

func (m Metadata) Encode() []byte {
	mlen := len(m)
	keys := make([][]byte, mlen)
	values := make([][]byte, mlen)
	for i, v := range m {
		keys[i] = proto2.EncodeString(v.Key)
		values[i] = proto2.EncodeBytes(v.Value)
	}
	keysArr := bytes.Join(keys, nil)
	valuesArr := bytes.Join(values, nil)

	return bytes.Join([][]byte{
		encodeUint16(uint16(mlen)),
		keysArr,
		valuesArr,
	}, nil)
}

func MetadataFromReader(r io.Reader) (Metadata, error) {
	size, err := decodeUint16FromReader(r)
	if err != nil {
		return nil, err
	}
	if size == 0 {
		return Metadata{}, nil
	}

	keys := make([]string, size)
	values := make([][]byte, size)

	for i := range size {
		k, err := proto2.DecodeString(r)
		if err != nil {
			return nil, err
		}
		keys[i] = k
	}

	for i := range size {
		v, err := proto2.DecodeBytes(r)
		if err != nil {
			return nil, err
		}
		values[i] = v
	}

	pairs := make([]MetadataPair, size)

	for i := int(size - 1); i >= 0; i-- {
		pairs[i] = MetadataPair{keys[i], values[i]}
	}

	return pairs, nil
}
