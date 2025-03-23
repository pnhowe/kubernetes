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

// add Bool and nil tests

var _ = Describe("Testing Configuration Values", func() {
	Context("When Building Simple Configuration Values", func() {
		It("String Type", func() {
			test := FromString("your string")
			Expect(test.Type).To(Equal(String))
			Expect(test.StringVal).To(Equal("your string"))
			Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"string\",\"string\":\"your string\"}")))
			Expect(test.ToContractor()).To(Equal("your string"))

			test = FromContractor("my string")
			Expect(test.Type).To(Equal(String))
			Expect(test.StringVal).To(Equal("my string"))
			Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"string\",\"string\":\"my string\"}")))
			Expect(test.ToContractor()).To(Equal("my string"))

			var test2 ConfigValue
			Expect(json.Unmarshal([]byte("{\"type\":\"string\",\"string\":\"more strings\"}"), &test2)).To(Succeed())
			Expect(test2.Type).To(Equal(String))
			Expect(test2.StringVal).To(Equal("more strings"))
			Expect(json.Marshal(test2)).To(Equal([]byte("{\"type\":\"string\",\"string\":\"more strings\"}")))
			Expect(test2.ToContractor()).To(Equal("more strings"))

			test3 := test.DeepCopy()
			Expect(test3.Type).To(Equal(String))
			Expect(test3.StringVal).To(Equal("my string"))
			Expect(json.Marshal(test3)).To(Equal([]byte("{\"type\":\"string\",\"string\":\"my string\"}")))
			Expect(test3.ToContractor()).To(Equal("my string"))
		})

		// 	It("Int Type", func() {
		// 		test := FromInt64(21)
		// 		Expect(test.Type).To(Equal(Integer))
		// 		Expect(test.IntegerVal).To(Equal(int64(21)))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"int\",\"int\":21}")))

		// 		test = FromInterface(432)
		// 		Expect(test.Type).To(Equal(Integer))
		// 		Expect(test.IntegerVal).To(Equal(int64(432)))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"int\",\"int\":432}")))

		// 		test = FromInterface(int(15))
		// 		Expect(test.Type).To(Equal(Integer))
		// 		Expect(test.IntegerVal).To(Equal(int64(15)))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"int\",\"int\":15}")))

		// 		test = FromInterface(int64(32))
		// 		Expect(test.Type).To(Equal(Integer))
		// 		Expect(test.IntegerVal).To(Equal(int64(32)))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"int\",\"int\":32}")))

		// 		test = FromInterface(int32(321))
		// 		Expect(test.Type).To(Equal(Integer))
		// 		Expect(test.IntegerVal).To(Equal(int64(321)))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"int\",\"int\":321}")))

		// 		var test2 ConfigValue
		// 		Expect(json.Unmarshal([]byte("{\"type\":\"int\",\"int\":43}"), &test2)).To(Succeed())
		// 		Expect(test2.Type).To(Equal(Integer))
		// 		Expect(test2.IntegerVal).To(Equal(int64(43)))
		// 		Expect(json.Marshal(test2)).To(Equal([]byte("{\"type\":\"int\",\"int\":43}")))

		// 		test3 := test.DeepCopy()
		// 		Expect(test3.Type).To(Equal(Integer))
		// 		Expect(test3.IntegerVal).To(Equal(int64(321)))
		// 		Expect(json.Marshal(test3)).To(Equal([]byte("{\"type\":\"int\",\"int\":321}")))
		// 	})

		// 	It("Float Type", func() {
		// 		test := FromFloat64(2.2)
		// 		Expect(test.Type).To(Equal(Float))
		// 		Expect(test.FloatVal).To(Equal("2.2"))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"float\",\"float\":\"2.2\"}")))

		// 		test = FromFloat32(42.24)
		// 		Expect(test.Type).To(Equal(Float))
		// 		Expect(test.FloatVal).To(Equal("42.24"))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"float\",\"float\":\"42.24\"}")))

		// 		test = FromInterface(3.14)
		// 		Expect(test.Type).To(Equal(Float))
		// 		Expect(test.FloatVal).To(Equal("3.14"))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"float\",\"float\":\"3.14\"}")))

		// 		test = FromInterface(float64(5.3))
		// 		Expect(test.Type).To(Equal(Float))
		// 		Expect(test.FloatVal).To(Equal("5.3"))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"float\",\"float\":\"5.3\"}")))

		// 		test = FromInterface(float32(123.5))
		// 		Expect(test.Type).To(Equal(Float))
		// 		Expect(test.FloatVal).To(Equal("123.5"))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"float\",\"float\":\"123.5\"}")))

		// 		var test2 ConfigValue
		// 		Expect(json.Unmarshal([]byte("{\"type\":\"float\",\"float\":\"1.8\"}"), &test2)).To(Succeed())
		// 		Expect(test2.Type).To(Equal(Float))
		// 		Expect(test2.FloatVal).To(Equal("1.8"))
		// 		Expect(json.Marshal(test2)).To(Equal([]byte("{\"type\":\"float\",\"float\":\"1.8\"}")))

		// 		test3 := test.DeepCopy()
		// 		Expect(test3.Type).To(Equal(Float))
		// 		Expect(test3.FloatVal).To(Equal("123.5"))
		// 		Expect(json.Marshal(test3)).To(Equal([]byte("{\"type\":\"float\",\"float\":\"123.5\"}")))
		// 	})
		// })

		// Context("When Building Complex Configuration Values", func() {
		// 	It("Slice Type", func() {
		// 		test := FromInterface([]ConfigValue{})
		// 		Expect(test.Type).To(Equal(Array))
		// 		Expect(test.ArrayVal).To(Equal([]ConfigValue{}))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"array\"}")))

		// 		tar := make([]ConfigValue, 3)
		// 		tar[0] = FromInt64(52)
		// 		tar[1] = FromString("sdf")
		// 		tar[2] = FromInt64(20)
		// 		test = FromSlice(tar)
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"array\",\"array\":[{\"type\":\"int\",\"int\":52},{\"type\":\"string\",\"string\":\"sdf\"},{\"type\":\"int\",\"int\":20}]}")))
		// 		Expect(test.Type).To(Equal(Array))
		// 		ref := make([]ConfigValue, 3)
		// 		ref[0] = FromInt64(52)
		// 		ref[1] = FromString("sdf")
		// 		ref[2] = FromInt64(20)
		// 		Expect(test.ArrayVal).To(Equal(ref))

		// 		test = FromInterface([]ConfigValue{FromInt(1), FromFloat64(2.1), FromString("sdf")})
		// 		Expect(test.Type).To(Equal(Array))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"array\",\"array\":[{\"type\":\"int\",\"int\":1},{\"type\":\"float\",\"float\":\"2.1\"},{\"type\":\"string\",\"string\":\"sdf\"}]}")))
		// 		ref = make([]ConfigValue, 3)
		// 		ref[0] = FromInt64(1)
		// 		ref[1] = FromFloat64(2.1)
		// 		ref[2] = FromString("sdf")
		// 		Expect(test.ArrayVal).To(Equal(ref))

		// 		var test2 ConfigValue
		// 		Expect(json.Unmarshal([]byte("{\"type\":\"array\",\"array\":[{\"type\":\"int\",\"int\":12},{\"type\":\"float\",\"float\":\"2\"}]}"), &test2)).To(Succeed())
		// 		Expect(test2.Type).To(Equal(Array))
		// 		Expect(json.Marshal(test2)).To(Equal([]byte("{\"type\":\"array\",\"array\":[{\"type\":\"int\",\"int\":12},{\"type\":\"float\",\"float\":\"2\"}]}")))
		// 		ref = make([]ConfigValue, 2)
		// 		ref[0] = FromInt64(12)
		// 		ref[1] = FromFloat64(2.0)
		// 		Expect(test2.ArrayVal).To(Equal(ref))

		// 		test3 := test.DeepCopy()
		// 		Expect(test3.Type).To(Equal(Array))
		// 		ref = make([]ConfigValue, 3)
		// 		ref[0] = FromInt64(1)
		// 		ref[1] = FromFloat64(2.1)
		// 		ref[2] = FromString("sdf")
		// 		Expect(test3.ArrayVal).To(Equal(ref))
		// 		Expect(json.Marshal(test3)).To(Equal([]byte("{\"type\":\"array\",\"array\":[{\"type\":\"int\",\"int\":1},{\"type\":\"float\",\"float\":\"2.1\"},{\"type\":\"string\",\"string\":\"sdf\"}]}")))
		// 	})

		// 	It("Map Type", func() {
		// 		test := FromInterface(map[string]ConfigValue{})
		// 		Expect(test.Type).To(Equal(Map))
		// 		Expect(test.MapVal).To(Equal(map[string]ConfigValue{}))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"map\"}")))

		// 		test = FromInterface(map[string]ConfigValue{"a": FromInt64(34), "f": FromInt64(2), "d": FromString("goodie")})
		// 		Expect(test.Type).To(Equal(Map))
		// 		Expect(json.Marshal(test)).To(Equal([]byte("{\"type\":\"map\",\"map\":{\"a\":{\"type\":\"int\",\"int\":34},\"d\":{\"type\":\"string\",\"string\":\"goodie\"},\"f\":{\"type\":\"int\",\"int\":2}}}")))
		// 		ref := make(map[string]ConfigValue, 3)
		// 		ref["a"] = FromInt64(34)
		// 		ref["f"] = FromInt64(2)
		// 		ref["d"] = FromString("goodie")
		// 		Expect(test.MapVal).To(Equal(ref))

		// 		var test2 ConfigValue
		// 		Expect(json.Unmarshal([]byte("{\"type\":\"map\",\"map\":{\"1\":{\"type\":\"float\",\"float\":\"34\"},\"e\":{\"type\":\"int\",\"int\":11223344},\"world\":{\"type\":\"string\",\"string\":\"hello\"}}}"), &test2)).To(Succeed())
		// 		Expect(test2.Type).To(Equal(Map))
		// 		Expect(json.Marshal(test2)).To(Equal([]byte("{\"type\":\"map\",\"map\":{\"1\":{\"type\":\"float\",\"float\":\"34\"},\"e\":{\"type\":\"int\",\"int\":11223344},\"world\":{\"type\":\"string\",\"string\":\"hello\"}}}")))
		// 		ref = make(map[string]ConfigValue, 3)
		// 		ref["1"] = FromFloat64(34)
		// 		ref["e"] = FromInt64(11223344)
		// 		ref["world"] = FromString("hello")
		// 		Expect(test2.MapVal).To(Equal(ref))

		// 		test3 := test.DeepCopy()
		// 		Expect(test3.Type).To(Equal(Map))
		// 		Expect(json.Marshal(test3)).To(Equal([]byte("{\"type\":\"map\",\"map\":{\"a\":{\"type\":\"int\",\"int\":34},\"d\":{\"type\":\"string\",\"string\":\"goodie\"},\"f\":{\"type\":\"int\",\"int\":2}}}")))
		// 		ref = make(map[string]ConfigValue, 3)
		// 		ref["a"] = FromInt64(34)
		// 		ref["f"] = FromInt64(2)
		// 		ref["d"] = FromString("goodie")
		// 		Expect(test3.MapVal).To(Equal(ref))
		// 	})
	})
})

var _ = Describe("Test Job Handeling", func() {
	// change in state and no existing job
	// not when state does not change
	// not shen already has job
})
