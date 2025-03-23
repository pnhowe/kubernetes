package v1

import (
	"fmt"
	"strconv"
	"strings"
)

type ConfigValues map[string]ConfigValue

func (c *ConfigValues) ToContractor() map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range *c {
		result[k] = v.ToContractor()
	}
	return result
}

func (c *ConfigValues) Clean() {
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~ Clean it!!!")
	for _, v := range *c {
		v.Clean()
	}
}

// ValuesFromContractor
func ValuesFromContractor(val map[string]interface{}) ConfigValues {
	if len(val) == 0 {
		return map[string]ConfigValue{}
	}

	cs := make(map[string]ConfigValue, len(val))

	for k, v := range val {
		cs[k] = FromContractor(v)
	}

	return cs
}

// keep an eye on https://github.com/kubernetes-sigs/controller-tools/issues/461, a union type would make the configValue less awarkward in the yaml
// and https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/1027-api-unions/README.md
// +kubebuilder:object:generate=false
type ConfigValue struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=nil
	// +kubebuilder:validation:Enum=nil;number;bool;string;array;map
	Type ConfigValueType `json:"type"`
	// +kubebuilder:validation:Optional
	BooleanVal bool `json:"boolean,omitempty"`
	// +kubebuilder:validation:Optional
	NumberVal string `json:"number,omitempty"`
	// +kubebuilder:validation:Optional
	StringVal string `json:"string,omitempty"`
	// // +kubebuilder:validation:Schemaless
	// // +kubebuilder:validation:Type=array
	// // +kubebuilder:validation:items:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Optional
	ArrayVal []ConfigValue `json:"array,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Optional
	MapVal map[string]ConfigValue `json:"map,omitempty"`
}

// // +kubebuilder:validation:Type=object
// // +kubebuilder:validation:Format=string
// // +kubebuilder:object:generate=false
// type ConfigValue struct {
// 	Type     ConfigValueType
// 	IntVal   int64
// 	FloatVal float64
// 	StrVal   string
// 	ArrayVal []ConfigValue
// 	MapVal   map[string]ConfigValue
// }

type ConfigValueType string

const (
	Nil     ConfigValueType = "nil"
	Number  ConfigValueType = "number"
	Boolean ConfigValueType = "bool"
	String  ConfigValueType = "string"
	Array   ConfigValueType = "array"
	Map     ConfigValueType = "map"
)

// Clean make sure the struct dosen't have extra junk in it, tends to mess up the reconcile loop
func (c *ConfigValue) Clean() {
	switch c.Type {
	case Nil:
		c.BooleanVal = false
		c.NumberVal = ""
		c.StringVal = ""
		c.ArrayVal = nil
		c.MapVal = nil
	case Boolean:
		c.NumberVal = ""
		c.StringVal = ""
		c.ArrayVal = nil
		c.MapVal = nil
	case Number:
		c.BooleanVal = false
		c.StringVal = ""
		c.ArrayVal = nil
		c.MapVal = nil
	case String:
		c.BooleanVal = false
		c.NumberVal = ""
		c.ArrayVal = nil
		c.MapVal = nil
	case Array:
		c.BooleanVal = false
		c.NumberVal = ""
		c.StringVal = ""
		c.MapVal = nil
		for i := range c.ArrayVal {
			c.ArrayVal[i].Clean()
		}
	case Map:
		c.BooleanVal = false
		c.NumberVal = ""
		c.StringVal = ""
		c.ArrayVal = nil
		for _, v := range c.MapVal {
			v.Clean()
		}
	}
}

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

// FromContractor creates an ConfigValue from the interface{} from the JSON decode from contractor
func FromContractor(val interface{}) ConfigValue {
	switch v := val.(type) {
	case nil:
		return ConfigValue{Type: Nil}
	case bool:
		return ConfigValue{Type: Boolean, BooleanVal: v}
	case int:
		return ConfigValue{Type: Number, NumberVal: strconv.FormatInt(int64(v), 10)}
	case int32:
		return ConfigValue{Type: Number, NumberVal: strconv.FormatInt(int64(v), 10)}
	case int64:
		return ConfigValue{Type: Number, NumberVal: strconv.FormatInt(v, 10)}
	case string:
		return ConfigValue{Type: String, StringVal: v}
	case float32:
		return ConfigValue{Type: Number, NumberVal: strconv.FormatFloat(float64(v), 'g', -1, 32)}
	case float64:
		return ConfigValue{Type: Number, NumberVal: strconv.FormatFloat(v, 'g', -1, 64)}
	case []interface{}:
		fmt.Println("{{{{{{{{{{{{{{{{{{{[[[]]]}}}}}}}}}}}}}}}}}}}")
		fmt.Printf("%+v\n", val)
		arrayval := make([]ConfigValue, len(v))
		for i, v2 := range v {
			arrayval[i] = FromContractor(v2)
		}
		fmt.Printf("%+v\n", arrayval)
		return ConfigValue{Type: Array, ArrayVal: arrayval}
	case map[string]interface{}:
		mapval := make(map[string]ConfigValue, len(v))
		for k, v2 := range v {
			mapval[k] = FromContractor(v2)
		}
		return ConfigValue{Type: Map, MapVal: mapval}
	}

	return ConfigValue{Type: Nil}
	// strval := val.(string)

	// // JSON dosen't know the difference between int and float, we will have to figure it out
	// intval, err := strconv.ParseInt(strval, 10, 64)
	// if err == nil {
	// 	fmt.Printf("int %+v\n", intval)
	// 	return ConfigValue{Type: Number, NumberVal: strconv.FormatInt(intval, 10)}
	// }

	// floatval, err := strconv.ParseFloat(strval, 64)
	// if err == nil {
	// 	fmt.Printf("float %+v\n", floatval)
	// 	return ConfigValue{Type: Number, NumberVal: strconv.FormatFloat(floatval, 'g', -1, 64)}
	// }

	// return ConfigValue{Type: String, StringVal: strval}
}

// ToContractor formats a return value sutible for JSON encoding to Contractor
func (c *ConfigValue) ToContractor() interface{} {
	if c == nil {
		return nil
	}

	switch c.Type {
	case Nil:
		return nil
	case Boolean:
		return c.BooleanVal
	case Number:
		if v, err := strconv.ParseFloat(c.NumberVal, 64); err == nil {
			return v
		} else {
			return 0
		}
	case String:
		return c.StringVal
	case Array:
		fmt.Println("}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}")
		fmt.Printf("%+v\n", c)
		arrayval := make([]interface{}, len(c.ArrayVal))
		for k, v := range c.ArrayVal {
			arrayval[k] = v.ToContractor()
		}
		fmt.Printf("%+v\n", arrayval)
		return arrayval
	case Map:
		interface_list := make(map[string]interface{}, len(c.MapVal))
		for k, v := range c.MapVal {
			interface_list[k] = v.ToContractor()
		}
		return interface_list
	}

	return nil
}

// FromNil creates an ConfigValue object with an float value.
func FromNil() ConfigValue {
	return ConfigValue{Type: Nil}
}

// FromBoolean creates an ConfigValue object with an int value.
func FromBoolean(val bool) ConfigValue {
	return ConfigValue{Type: Boolean, BooleanVal: val}
}

// FromInt64 creates an ConfigValue object with an int value.
func FromInt64(val int64) ConfigValue {
	return ConfigValue{Type: Number, NumberVal: strconv.FormatInt(val, 10)}
}

// FromFloat64 creates an ConfigValue object with an float value.
func FromFloat64(val float64) ConfigValue {
	return ConfigValue{Type: Number, NumberVal: strconv.FormatFloat(val, 'g', -1, 64)}
}

// FromString creates an ConfigValue object with a string value.
func FromString(val string) ConfigValue {
	return ConfigValue{Type: String, StringVal: val}
}

// FromSlice creates an ConfigValue object with a slice value.
func FromSlice(val []ConfigValue) ConfigValue {
	return ConfigValue{Type: Array, ArrayVal: val}
}

// // Unmarshal implements the yaml.Unmarshaller interface.
// func (c *ConfigValue) Unmarshal(value []byte) error {
// 	if (value[0] == 'n') && (value[1] == 'u') && (value[2] == 'l') && (value[3] == 'l') {
// 		c.Type = Nil
// 		return nil
// 	}

// 	if value[0] == '"' {
// 		c.Type = String
// 		return yaml.Unmarshal(value, &c.StrVal)
// 	}

// 	if value[0] == '[' {
// 		c.Type = Array
// 		return yaml.Unmarshal(value, &c.ArrayVal)
// 	}
// 	if value[0] == '{' {
// 		c.Type = Map
// 		return yaml.Unmarshal(value, &c.MapVal)
// 	}

// 	if yaml.Unmarshal(value, &c.IntVal) == nil {
// 		c.Type = Int
// 		return nil
// 	}

// 	if json.Unmarshal(value, &c.FloatVal) == nil {
// 		c.Type = Float
// 		return nil
// 	}

// 	return fmt.Errorf("unable to Unmarshal YAML")
// }

// // Marshal implements the json.Marshaller interface.
// func (c ConfigValue) Marshal() ([]byte, error) {
// 	switch c.Type {
// 	case Nil:
// 		return []byte("null"), nil
// 	case Int:
// 		return yaml.Marshal(c.IntVal)
// 	case Float:
// 		return yaml.Marshal(c.FloatVal)
// 	case String:
// 		return yaml.Marshal(c.StrVal)
// 	case Array:
// 		return yaml.Marshal(c.ArrayVal)
// 	case Map:
// 		return yaml.Marshal(c.MapVal)
// 	default:
// 		return []byte{}, fmt.Errorf("impossible configValue type")
// 	}
// }

// // UnmarshalJSON implements the json.Unmarshaller interface.
// func (c *ConfigValue) UnmarshalJSON(value []byte) error {
// 	if (value[0] == 'n') && (value[1] == 'u') && (value[2] == 'l') && (value[3] == 'l') {
// 		c.Type = Nil
// 		return nil
// 	}

// 	if value[0] == '"' {
// 		c.Type = String
// 		return json.Unmarshal(value, &c.StrVal)
// 	}

// 	if value[0] == '[' {
// 		c.Type = Array
// 		return json.Unmarshal(value, &c.ArrayVal)
// 	}

// 	if value[0] == '{' {
// 		c.Type = Map
// 		return json.Unmarshal(value, &c.MapVal)
// 	}

// 	if json.Unmarshal(value, &c.IntVal) == nil {
// 		c.Type = Int
// 		return nil
// 	}

// 	if json.Unmarshal(value, &c.FloatVal) == nil {
// 		c.Type = Float
// 		return nil
// 	}

// 	return fmt.Errorf("unable to Unmarshal JSON")
// }

// // MarshalJSON implements the json.Marshaller interface.
// func (c ConfigValue) MarshalJSON() ([]byte, error) {
// 	switch c.Type {
// 	case Nil:
// 		return []byte("null"), nil
// 	case Int:
// 		return json.Marshal(c.IntVal)
// 	case Float:
// 		return json.Marshal(c.FloatVal)
// 	case String:
// 		return json.Marshal(c.StrVal)
// 	case Array:
// 		return json.Marshal(c.ArrayVal)
// 	case Map:
// 		return json.Marshal(c.MapVal)
// 	default:
// 		return []byte{}, fmt.Errorf("impossible configValue type")
// 	}
// }

// func (c ConfigValue) MarshalText() (text []byte, err error) {
// 	return []byte{}, fmt.Errorf("Not implemented")
// }

// String returns the string value, or the Itoa of the int value.
func (c *ConfigValue) String() string {
	if c == nil {
		return "<nil>"
	}

	switch c.Type {
	case Nil:
		return ""
	case Boolean:
		return strconv.FormatBool(c.BooleanVal)
	case Number:
		return c.NumberVal
	case String:
		return c.StringVal
	case Array:
		string_list := make([]string, len(c.ArrayVal))
		for k, v := range c.ArrayVal {
			string_list[k] = v.String()
		}
		return strings.Join(string_list, ", ")
	case Map:
		string_list := make([]string, len(c.MapVal))
		i := 0
		for k, v := range c.MapVal {
			string_list[i] = k + ": " + v.String()
			i += 1
		}
		return strings.Join(string_list, ", ")
	}
	return "<invalid>"
}

// DeepCopy copys deeply
func (c ConfigValue) DeepCopy() *ConfigValue {
	copy := ConfigValue{Type: c.Type}
	switch c.Type {
	case Nil:
		//
	case Boolean:
		copy.BooleanVal = c.BooleanVal
	case Number:
		copy.NumberVal = c.NumberVal
	case String:
		copy.StringVal = c.StringVal
	case Array:
		copy.ArrayVal = make([]ConfigValue, len(c.ArrayVal))
		for key, val := range c.ArrayVal {
			copy.ArrayVal[key] = val
		}
	case Map:
		copy.MapVal = make(map[string]ConfigValue, len(c.MapVal))
		for key, val := range c.MapVal {
			copy.MapVal[key] = val
		}
	}

	return &copy
}

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
