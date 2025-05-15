package v1

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/google/go-cmp/cmp"
)

type ConfigValues map[string]ConfigValue

func (cvs *ConfigValues) Value() map[string]any {
	value_map := make(map[string]any, len(*cvs))
	for k, v := range *cvs {
		value_map[k] = v.Value()
	}
	return value_map
}

func (cvs ConfigValues) String() string {
	value_map := make(map[string]any, len(cvs))
	for k, v := range cvs {
		value_map[k] = v.Value()
	}
	return fmt.Sprintf("%v", value_map)
}

func (cvs ConfigValues) Equal(cvs2 ConfigValues) bool {
	if (cvs == nil) || (cvs2 == nil) { // often we get the case where one is nil and the other is map[], we count that as equal
		return (len(cvs) == 0) == (len(cvs2) == 0) // in golang len(nil) == 0 , so we don't need to expilctally say len() == 0 and == nil
	}

	if len(cvs) != len(cvs2) {
		return false
	}

	cvs_keys := slices.Sorted(maps.Keys(cvs))
	cvs2_keys := slices.Sorted(maps.Keys(cvs2))
	for i := range len(cvs_keys) {
		if cvs_keys[i] != cvs2_keys[i] {
			return false
		}

		key := cvs_keys[i]
		if !cmp.Equal(cvs[key], cvs2[key]) {
			return false
		}
	}

	return true
}

func ConfigValuesFromContractor(values map[string]any) ConfigValues {
	if len(values) == 0 {
		return map[string]ConfigValue{}
	}

	cs := make(map[string]ConfigValue, len(values))

	for k, v := range values {
		cs[k] = ConfigValueFromContractor(v)
	}

	return cs
}

func (c *ConfigValues) ToContractor() map[string]any {
	result := map[string]any{}
	for k, v := range *c {
		result[k] = v.ToContractor()
	}
	return result
}

type ConfigValue struct {
	strVal   *string                `json:"-"`
	numVal   *float64               `json:"-"`
	boolVal  *bool                  `json:"-"`
	arrayVal []ConfigValue          `json:"-"`
	mapVal   map[string]ConfigValue `json:"-"`
}

func NewConfigValue(val any) ConfigValue {
	return ConfigValueFromContractor(val)
}

func (cv *ConfigValue) Value() any {
	if cv.numVal != nil {
		return *cv.numVal
	}

	if cv.boolVal != nil {
		return *cv.boolVal
	}

	if cv.strVal != nil {
		return *cv.strVal
	}

	if cv.arrayVal != nil {
		value_list := make([]any, len(cv.arrayVal))
		for k, v := range cv.arrayVal {
			value_list[k] = v.Value()
		}
		return value_list
	}

	if cv.mapVal != nil {
		value_map := make(map[string]any, len(cv.mapVal))
		for k, v := range cv.mapVal {
			value_map[k] = v.Value()
		}
		return value_map
	}

	return nil
}

func (cv ConfigValue) String() string {
	return fmt.Sprintf("%v", cv.Value())
}

// Custom unmarshaling logic
func (cv *ConfigValue) UnmarshalJSON(data []byte) error {
	var tmpFloat float64
	if err := json.Unmarshal(data, &tmpFloat); err == nil {
		cv.numVal = &tmpFloat
		return nil
	}

	var tmpBool bool
	if err := json.Unmarshal(data, &tmpBool); err == nil {
		cv.boolVal = &tmpBool
		return nil
	}

	var tmpSlice []ConfigValue
	if err := json.Unmarshal(data, &tmpSlice); err == nil {
		cv.arrayVal = tmpSlice
		return nil
	}

	var tmpMap map[string]ConfigValue
	if err := json.Unmarshal(data, &tmpMap); err == nil {
		cv.mapVal = tmpMap
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		cv.strVal = &str
		return nil
	}

	return fmt.Errorf("unmarshalable value for ConfigValue")
}

func (cv ConfigValue) MarshalJSON() ([]byte, error) {
	if cv.numVal != nil {
		return json.Marshal(*cv.numVal)
	}

	if cv.boolVal != nil {
		return json.Marshal(*cv.boolVal)
	}

	if cv.arrayVal != nil {
		return json.Marshal(cv.arrayVal)
	}

	if cv.mapVal != nil {
		return json.Marshal(cv.mapVal)
	}

	if cv.strVal != nil {
		return json.Marshal(cv.strVal)
	}

	return json.Marshal(nil)
}

func (cv ConfigValue) Equal(cv2 ConfigValue) bool {
	return cmp.Equal(cv.Value(), cv2.Value())
}

func ConfigValueFromContractor(value any) ConfigValue {
	switch v := value.(type) {
	case nil:
		return ConfigValue{}
	case bool:
		return ConfigValue{boolVal: &v}
	case string:
		return ConfigValue{strVal: &v}
	case int:
		tmp := float64(v)
		return ConfigValue{numVal: &tmp}
	case int32:
		tmp := float64(v)
		return ConfigValue{numVal: &tmp}
	case int64:
		tmp := float64(v)
		return ConfigValue{numVal: &tmp}
	case float32:
		tmp := float64(v)
		return ConfigValue{numVal: &tmp}
	case float64:
		return ConfigValue{numVal: &v}
	case []any:
		value_list := make([]ConfigValue, len(v))
		for k, v := range v {
			value_list[k] = ConfigValueFromContractor(v)
		}
		return ConfigValue{arrayVal: value_list}
	case map[string]any:
		value_map := make(map[string]ConfigValue, len(v))
		for k, v := range v {
			value_map[k] = ConfigValueFromContractor(v)
		}
		return ConfigValue{mapVal: value_map}
	}

	tmp := fmt.Sprintf("%+v", value)
	return ConfigValue{strVal: &tmp}
}

func (cv *ConfigValue) ToContractor() any {
	return cv.Value()
}
