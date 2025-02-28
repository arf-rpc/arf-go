package proto

var reg *registry

func init() {
	reg = &registry{
		structs: map[string]knownStructType{},
	}
}
