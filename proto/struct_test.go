package proto

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type SampleStruct struct {
	A uint8                `arf:"0"`
	B uint16               `arf:"1"`
	C uint32               `arf:"2"`
	D uint64               `arf:"3"`
	E int8                 `arf:"4"`
	F int16                `arf:"5"`
	G int32                `arf:"6"`
	H int64                `arf:"7"`
	I float32              `arf:"8"`
	J float64              `arf:"9"`
	K bool                 `arf:"10"`
	L map[string]string    `arf:"11"`
	M string               `arf:"12"`
	N []byte               `arf:"13"`
	O []string             `arf:"14"`
	P *string              `arf:"15"`
	Q *bool                `arf:"16"`
	R SubStruct            `arf:"17"`
	W map[string]SubStruct `arf:"18"`
	X []SubStruct          `arf:"19"`
}

func (SampleStruct) ArfStructID() string { return "org.example.test/SampleStruct" }

type SubStruct struct {
	A string `arf:"0"`
}

func (SubStruct) ArfStructID() string { return "org.example.test/SubStruct" }

func TestStruct(t *testing.T) {
	resetRegistry()
	RegisterMessage(SampleStruct{})
	RegisterMessage(SubStruct{})

	v := "ptr string"
	s := &SampleStruct{
		A: 0,
		B: 1,
		C: 2,
		D: 3,
		E: 4,
		F: 5,
		G: 6,
		H: 7,
		I: 8,
		J: 9,
		K: true,
		L: map[string]string{
			"hello": "arf!",
		},
		M: "Sample String",
		N: []byte{0x01, 0x02, 0x03},
		O: []string{"hello", "world"},
		P: &v,
		Q: nil,
		R: SubStruct{
			A: "SubStruct value",
		},
		W: map[string]SubStruct{
			"test": {A: "substruct in map"},
		},
		X: []SubStruct{
			{A: "substruct in array 1"}, {A: "substruct in array 2"},
		},
	}
	b, err := Encode(s)
	require.NoError(t, err)
	fmt.Println(hex.EncodeToString(b))
	d, err := DecodeAny(bytes.NewReader(b))
	require.NoError(t, err)

	assert.Equal(t, s, d)
}
