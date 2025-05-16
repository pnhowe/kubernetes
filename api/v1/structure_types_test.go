package v1

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStructureTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Structure Types")
}

// TODO: add Bool and nil tests

var _ = Describe("Testing Configuration Values", func() {
	Context("When Building Simple Configuration Values", func() {
		It("String Type", func() {
			test := ConfigValueFromContractor("your string")
			Expect(*test.strVal).To(Equal("your string"))
			Expect(test.Value()).To(Equal("your string"))
			Expect(json.Marshal(test)).To(Equal([]byte("\"your string\"")))
			Expect(test.ToContractor()).To(Equal("your string"))

			test = ConfigValueFromContractor("my string")
			Expect(*test.strVal).To(Equal("my string"))
			Expect(test.Value()).To(Equal("my string"))
			Expect(json.Marshal(test)).To(Equal([]byte("\"my string\"")))
			Expect(test.ToContractor()).To(Equal("my string"))

			var test2 ConfigValue
			Expect(json.Unmarshal([]byte("\"more strings\""), &test2)).To(Succeed())
			Expect(*test2.strVal).To(Equal("more strings"))
			Expect(test2.Value()).To(Equal("more strings"))
			Expect(json.Marshal(test2)).To(Equal([]byte("\"more strings\"")))
			Expect(test2.ToContractor()).To(Equal("more strings"))

			test3 := test.DeepCopy()
			Expect(*test3.strVal).To(Equal("my string"))
			Expect(test3.Value()).To(Equal("my string"))
			Expect(json.Marshal(test3)).To(Equal([]byte("\"my string\"")))
			Expect(test3.ToContractor()).To(Equal("my string"))
		})

		It("Number Type", func() {
			test := ConfigValueFromContractor(21)
			Expect(*test.numVal).To(Equal(float64(21)))
			Expect(test.Value()).To(Equal(float64(21)))
			Expect(json.Marshal(test)).To(Equal([]byte("21")))

			test = ConfigValueFromContractor(int64(432))
			Expect(*test.numVal).To(Equal(float64(432)))
			Expect(test.Value()).To(Equal(float64(432)))
			Expect(json.Marshal(test)).To(Equal([]byte("432")))

			test = ConfigValueFromContractor(int32(321))
			Expect(*test.numVal).To(Equal(float64(321)))
			Expect(test.Value()).To(Equal(float64(321)))
			Expect(json.Marshal(test)).To(Equal([]byte("321")))

			var test2 ConfigValue
			Expect(json.Unmarshal([]byte("43"), &test2)).To(Succeed())
			Expect(*test2.numVal).To(Equal(float64(43)))
			Expect(test2.Value()).To(Equal(float64(43)))
			Expect(json.Marshal(test2)).To(Equal([]byte("43")))

			test3 := test.DeepCopy()
			Expect(*test3.numVal).To(Equal(float64(321)))
			Expect(test3.Value()).To(Equal(float64(321)))
			Expect(json.Marshal(test3)).To(Equal([]byte("321")))

			test = ConfigValueFromContractor(float64(2.2))
			Expect(*test.numVal).To(Equal(float64(2.2)))
			Expect(test.Value()).To(Equal(float64(2.2)))
			Expect(json.Marshal(test)).To(Equal([]byte("2.2")))

			test = ConfigValueFromContractor(float64(5.3))
			Expect(*test.numVal).To(Equal(float64(5.3)))
			Expect(test.Value()).To(Equal(float64(5.3)))
			Expect(json.Marshal(test)).To(Equal([]byte("5.3")))

			test = ConfigValueFromContractor(float32(123.5))
			Expect(*test.numVal).To(Equal(float64(123.5)))
			Expect(test.Value()).To(Equal(float64(123.5)))
			Expect(json.Marshal(test)).To(Equal([]byte("123.5")))

			Expect(json.Unmarshal([]byte("1.8"), &test2)).To(Succeed())
			Expect(*test2.numVal).To(Equal(float64(1.8)))
			Expect(test2.Value()).To(Equal(float64(1.8)))
			Expect(json.Marshal(test2)).To(Equal([]byte("1.8")))

			test3 = test.DeepCopy()
			Expect(*test3.numVal).To(Equal(float64(123.5)))
			Expect(test3.Value()).To(Equal(float64(123.5)))
			Expect(json.Marshal(test3)).To(Equal([]byte("123.5")))
		})
	})

	Context("When Building Complex Configuration Values", func() {
		It("Slice Type", func() {
			test := ConfigValueFromContractor([]any{})
			Expect(test.arrayVal).To(Equal([]ConfigValue{}))
			Expect(test.Value()).To(Equal([]any{}))
			Expect(json.Marshal(test)).To(Equal([]byte("[]")))

			tar := make([]any, 3)
			tar[0] = 52
			tar[1] = "sdf"
			tar[2] = 20
			test = ConfigValueFromContractor(tar)
			Expect(test.arrayVal).To(Equal([]ConfigValue{ConfigValueFromContractor(52), ConfigValueFromContractor("sdf"), ConfigValueFromContractor(20)}))
			Expect(test.Value()).To(Equal([]any{float64(52), "sdf", float64(20)}))
			Expect(json.Marshal(test)).To(Equal([]byte("[52,\"sdf\",20]")))

			test = ConfigValueFromContractor([]any{1, 2.1, "aaabbbccc"})
			Expect(test.arrayVal).To(Equal([]ConfigValue{ConfigValueFromContractor(1), ConfigValueFromContractor(2.1), ConfigValueFromContractor("aaabbbccc")}))
			Expect(test.Value()).To(Equal([]any{float64(1), float64(2.1), "aaabbbccc"}))
			Expect(json.Marshal(test)).To(Equal([]byte("[1,2.1,\"aaabbbccc\"]")))

			var test2 ConfigValue
			Expect(json.Unmarshal([]byte("[12, 2.0]"), &test2)).To(Succeed())
			Expect(test2.arrayVal).To(Equal([]ConfigValue{ConfigValueFromContractor(12), ConfigValueFromContractor(2)}))
			Expect(test2.Value()).To(Equal([]any{float64(12), float64(2)}))
			Expect(json.Marshal(test2)).To(Equal([]byte("[12,2]")))

			test3 := test.DeepCopy()
			Expect(test.arrayVal).To(Equal([]ConfigValue{ConfigValueFromContractor(1), ConfigValueFromContractor(2.1), ConfigValueFromContractor("aaabbbccc")}))
			Expect(json.Marshal(test3)).To(Equal([]byte("[1,2.1,\"aaabbbccc\"]")))
		})

		It("Map Type", func() {
			test := ConfigValueFromContractor(map[string]any{})
			Expect(test.mapVal).To(Equal(map[string]ConfigValue{}))
			Expect(test.Value()).To(Equal(map[string]any{}))
			Expect(json.Marshal(test)).To(Equal([]byte("{}")))

			test = ConfigValueFromContractor(map[string]any{"a": 34, "f": 2, "d": "goodie"})
			Expect(test.mapVal).To(Equal(map[string]ConfigValue{"a": ConfigValueFromContractor(34), "f": ConfigValueFromContractor(2), "d": ConfigValueFromContractor("goodie")}))
			Expect(test.Value()).To(Equal(map[string]any{"a": float64(34), "f": float64(2), "d": "goodie"}))
			Expect(json.Marshal(test)).To(Equal([]byte("{\"a\":34,\"d\":\"goodie\",\"f\":2}")))

			var test2 ConfigValue
			Expect(json.Unmarshal([]byte("{\"1\":\"34\",\"e\":11223344,\"world\":\"hello\"}"), &test2)).To(Succeed())
			Expect(test2.mapVal).To(Equal(map[string]ConfigValue{"1": ConfigValueFromContractor("34"), "e": ConfigValueFromContractor(11223344), "world": ConfigValueFromContractor("hello")}))
			Expect(test2.Value()).To(Equal(map[string]any{"1": "34", "e": float64(11223344), "world": "hello"}))
			Expect(json.Marshal(test2)).To(Equal([]byte("{\"1\":\"34\",\"e\":11223344,\"world\":\"hello\"}")))

			test3 := test.DeepCopy()
			Expect(test3.mapVal).To(Equal(map[string]ConfigValue{"a": ConfigValueFromContractor(34), "f": ConfigValueFromContractor(2), "d": ConfigValueFromContractor("goodie")}))
			Expect(test3.Value()).To(Equal(map[string]any{"a": float64(34), "f": float64(2), "d": "goodie"}))
			Expect(json.Marshal(test3)).To(Equal([]byte("{\"a\":34,\"d\":\"goodie\",\"f\":2}")))
		})
	})
})

var _ = Describe("Test Job Handeling", func() {
	// change in state and no existing job
	// not when state does not change
	// not shen already has job
})
