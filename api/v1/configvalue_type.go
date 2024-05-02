package v1

import (
	"encoding/json"
	"fmt"
)

// +protobuf=true
// +protobuf.options.(gogoproto.goproto_stringer)=false
// +k8s:openapi-gen=true
type ConfigValue struct {
	Type   ConfigValueType `protobuf:"varint,1,opt,name=type,casttype=Type"`
	IntVal int64           `protobuf:"varint,2,opt,name=intVal"`
	StrVal string          `protobuf:"bytes,3,opt,name=strVal"`
	// ListVal []ConfigValue          `protobuf:"bytes,4,opt,name=listVal"`
	// MapVal  map[string]ConfigValue `protobuf:"bytes,5,opt,name=mapVal"`
}

type ConfigValueType int64

const (
	Int ConfigValueType = iota
	String
	List
	Map
)

// UnmarshalJSON implements the json.Unmarshaller interface.
func (c *ConfigValue) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		c.Type = String
		return json.Unmarshal(value, &c.StrVal)
		// } else if value[0] == '[' {
		// 	c.Type = List
		// 	return json.Unmarshal(value, &c.ListVal)
		// } else if value[0] == '{' {
		// 	c.Type = Map
		// 	return json.Unmarshal(value, &c.MapVal)
	}
	c.Type = Int
	return json.Unmarshal(value, &c.IntVal)
}

// MarshalJSON implements the json.Marshaller interface.
func (c ConfigValue) MarshalJSON() ([]byte, error) {
	switch c.Type {
	case Int:
		return json.Marshal(c.IntVal)
	case String:
		return json.Marshal(c.StrVal)
	// case List:
	// 	return json.Marshal(c.ListVal)
	// case Map:
	// 	return json.Marshal(c.MapVal)
	default:
		return []byte{}, fmt.Errorf("impossible ConfigValue Type")
	}
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (ConfigValue) OpenAPISchemaType() []string { return []string{"string"} }

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (ConfigValue) OpenAPISchemaFormat() string { return "int-or-string" }

// OpenAPIV3OneOfTypes is used by the kube-openapi generator when constructing
// the OpenAPI v3 spec of this type.
func (ConfigValue) OpenAPIV3OneOfTypes() []string {
	return []string{"integer", "string"}
}
