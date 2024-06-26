package contractor

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// _k8s:openapi-gen=true
// _kubebuilder:validation:Type=object
// _kubebuilder:validation:Format=int-or-str
// _kubebuilder:validation:Schemaless
// _kubebuilder:pruning:PreserveUnknownFields
type ConfigValue struct {
	// +kubebuilder:validation:Required
	Type ConfigValueType `json:"type,omitempty"`
	// +kubebuilder:validation:Optional
	IntVal int64 `json:"int,omitempty"`
	// +kubebuilder:validation:Optional
	FloatVal float64 `json:"float,omitempty"`
	// +kubebuilder:validation:Optional
	StrVal string `json:"string,omitempty"`
	// +kubebuilder:validation:Optional
	ArrayVal []intstr.IntOrString `json:"array,omitempty"`
	// +kubebuilder:validation:Optional
	MapVal map[string]intstr.IntOrString `json:"map,omitempty"`
}

type ConfigValueType string

const (
	Int    ConfigValueType = "int"
	Float  ConfigValueType = "float"
	String ConfigValueType = "string"
	Array  ConfigValueType = "array"
	Map    ConfigValueType = "map"
)

// type ConfigValueType uint8

// const (
// 	Int    ConfigValueType = 1
// 	String ConfigValueType = 2
// 	Array  ConfigValueType = 3
// 	Map    ConfigValueType = 4
// )

// var ConfigValueType_name = map[uint8]string{
// 	1: "int",
// 	2: "string",
// 	3: "array",
// 	4: "map",
// }

// var ConfigValueType_value = map[string]uint8{
// 	"int":    1,
// 	"string": 2,
// 	"array":  3,
// 	"map":    4,
// }

// func (c ConfigValueType) String() string {
// 	return ConfigValueType_name[uint8(c)]
// }

// func (c ConfigValueType) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(c.String())
// }

// func (c *ConfigValueType) UnmarshalJSON(data []byte) (err error) {
// 	var name string
// 	if err := json.Unmarshal(data, &name); err != nil {
// 		return err
// 	}
// 	value, ok := ConfigValueType_value[name]
// 	if !ok {
// 		return fmt.Errorf("invalid type '%q'", name)
// 	}
// 	*c = ConfigValueType(value)
// 	return nil
// }

// FromInterface creates an ConfigValue from interface{}
func FromInterface(val interface{}) ConfigValue {
	switch v := val.(type) {
	case int:
		return ConfigValue{Type: Int, IntVal: int64(v)}
	case int32:
		return ConfigValue{Type: Int, IntVal: int64(v)}
	case int64:
		return ConfigValue{Type: Int, IntVal: v}
	case float32:
		return ConfigValue{Type: Float, FloatVal: float64(v)}
	case float64:
		return ConfigValue{Type: Float, FloatVal: v}
	case []intstr.IntOrString:
		return ConfigValue{Type: Array, ArrayVal: v}
	case map[string]intstr.IntOrString:
		return ConfigValue{Type: Map, MapVal: v}
	}

	return ConfigValue{Type: String, StrVal: val.(string)}
}

// FromInt64 creates an ConfigValue object with an int64 value.
func FromInt64(val int64) ConfigValue {
	return ConfigValue{Type: Int, IntVal: val}
}

// FromString creates an ConfigValue object with a string value.
func FromString(val string) ConfigValue {
	return ConfigValue{Type: String, StrVal: val}
}

// FromSlice creates an ConfigValue object with a slice value.
func FromSlice(val []intstr.IntOrString) ConfigValue {
	return ConfigValue{Type: Array, ArrayVal: val}
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (c *ConfigValue) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		c.Type = String
		return json.Unmarshal(value, &c.StrVal)
	} else if value[0] == '[' {
		c.Type = Array
		return json.Unmarshal(value, &c.ArrayVal)
	} else if value[0] == '{' {
		c.Type = Map
		return json.Unmarshal(value, &c.MapVal)
	}
	c.Type = Float
	if json.Unmarshal(value, &c.FloatVal) == nil {
		return nil
	}
	c.Type = Int
	return json.Unmarshal(value, &c.IntVal)
}

// MarshalJSON implements the json.Marshaller interface.
func (c ConfigValue) MarshalJSON() ([]byte, error) {
	switch c.Type {
	case Int:
		return json.Marshal(c.IntVal)
	case Float:
		return json.Marshal(c.FloatVal)
	case String:
		return json.Marshal(c.StrVal)
	case Array:
		return json.Marshal(c.ArrayVal)
	case Map:
		return json.Marshal(c.MapVal)
	default:
		return []byte{}, fmt.Errorf("impossible ConfigValue Type")
	}
}

// DeepCopy copys deeply
func (c ConfigValue) DeepCopy() *ConfigValue {
	copy := ConfigValue{Type: c.Type}
	switch c.Type {
	case Int:
		copy.IntVal = c.IntVal
	case Float:
		copy.FloatVal = c.FloatVal
	case String:
		copy.StrVal = c.StrVal
	case Array:
		copy.ArrayVal = make([]intstr.IntOrString, len(c.ArrayVal))
		for key, val := range c.ArrayVal {
			copy.ArrayVal[key] = val
		}
	case Map:
		copy.MapVal = make(map[string]intstr.IntOrString, len(c.MapVal))
		for key, val := range c.MapVal {
			copy.MapVal[key] = val
		}
	}

	return &copy
}

// // OpenAPIV3OneOfTypes is used by the kube-openapi generator when constructing
// // the OpenAPI v3 spec of this type.
// func (ConfigValue) OpenAPIV3OneOfTypes() []string {
// 	panic("Hello???")
// 	//return []string{"integer", "string", "array", "object"}
// }

/*
https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/1027-api-unions
https://groups.google.com/g/kubebuilder/c/ImZ5BFqV394?pli=1
https://stackoverflow.com/questions/46472543/specifying-multiple-types-for-additionalproperties-through-swagger-openapi
https://spec.openapis.org/oas/latest.html
https://book.kubebuilder.io/reference/markers/crd-validation
https://kubernetes.io/blog/2019/06/20/crd-structural-schema/
https://github.com/kubernetes/kubernetes/issues/91153

https://rotational.io/blog/marshaling-go-enums-to-and-from-json/
https://github.com/kubernetes-sigs/controller-tools/issues/477
https://github.com/kubernetes-sigs/controller-tools/issues/461
https://groups.google.com/g/kubebuilder/c/ImZ5BFqV394?pli=1
https://github.com/metal3-io/baremetal-operator/blob/main/apis/metal3.io/v1alpha1/hostfirmwaresettings_types.go

*/
