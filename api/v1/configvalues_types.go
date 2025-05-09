package v1

import (
	"fmt"
	"strconv"
)

type ConfigValues map[string]ConfigValue

func ConfigValuesFromContractor(values map[string]interface{}) ConfigValues {
	fmt.Printf("*** From: '%+v'\n", values)

	if len(values) == 0 {
		return map[string]ConfigValue{}
	}

	cs := make(map[string]ConfigValue, len(values))

	for k, v := range values {
		cs[k] = ConfigValueFromContractor(v)
	}

	return cs
}

func (c *ConfigValues) ToContractor() map[string]interface{} {
	fmt.Printf("*** to: '%+v'\n", c)
	result := map[string]interface{}{}
	for k, v := range *c {
		result[k] = v.ToContractor()
	}
	return result
}

// https://github.com/kubernetes-sigs/controller-tools/issues/461
// https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/1027-api-unions/README.md
type ConfigValue struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:default=nil
	// +kubebuilder:validation:Enum=nil;number;bool;string;array;map
	Type  ConfigValueType `json:"type"`
	Value string          `json:"value"`
}

type ConfigValueType string

const (
	Nil     ConfigValueType = "nil"
	Number  ConfigValueType = "number"
	Boolean ConfigValueType = "bool"
	String  ConfigValueType = "string"
	Array   ConfigValueType = "array"
	Map     ConfigValueType = "map"
)

func ConfigValueFromContractor(value interface{}) ConfigValue {
	switch v := value.(type) {
	case nil:
		return ConfigValue{Type: Nil, Value: ""}
	case bool:
		return ConfigValue{Type: Boolean, Value: strconv.FormatBool(v)}
	case int:
		return ConfigValue{Type: Number, Value: strconv.FormatInt(int64(v), 10)}
	case int32:
		return ConfigValue{Type: Number, Value: strconv.FormatInt(int64(v), 10)}
	case int64:
		return ConfigValue{Type: Number, Value: strconv.FormatInt(v, 10)}
	case string:
		return ConfigValue{Type: String, Value: v}
	case float32:
		return ConfigValue{Type: Number, Value: strconv.FormatFloat(float64(v), 'g', -1, 32)}
	case float64:
		return ConfigValue{Type: Number, Value: strconv.FormatFloat(v, 'g', -1, 64)}
	}

	return ConfigValue{Type: String, Value: fmt.Sprintf("%+v", value)}
}

func (c *ConfigValue) ToContractor() interface{} {
	return ""
}

func init() {
	SchemeBuilder.Register(&Structure{}, &StructureList{})
}
