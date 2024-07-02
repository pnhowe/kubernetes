package v1

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestStructureSpecValidation(t *testing.T) {
	BeforeEach(func() {
	})

	Context("When Building Simple Configuration Values", func() {
		It("String Type", func() {
			test := FromString("your string")
			Expect(test.Type).To(Equal(String))
			Expect(test.StrVal).To(Equal("your string"))
			Expect(test.MarshalJSON()).To(Equal([]byte("\"your string\"")))

			test = FromInterface("my string")
			Expect(test.Type).To(Equal(String))
			Expect(test.StrVal).To(Equal("my string"))
			Expect(test.MarshalJSON()).To(Equal([]byte("\"my string\"")))

			var test2 ConfigValue
			Expect(test2.UnmarshalJSON([]byte("\"more strings\""))).To(Succeed())
			Expect(test2.Type).To(Equal(String))
			Expect(test2.StrVal).To(Equal("more strings"))
			Expect(test2.MarshalJSON()).To(Equal([]byte("\"more strings\"")))

			test3 := test.DeepCopy()
			Expect(test3.Type).To(Equal(String))
			Expect(test3.StrVal).To(Equal("my string"))
			Expect(test3.MarshalJSON()).To(Equal([]byte("\"my string\"")))
		})

		It("Int Type", func() {
			test := FromInt64(21)
			Expect(test.Type).To(Equal(Int))
			Expect(test.IntVal).To(Equal(int64(21)))
			Expect(test.MarshalJSON()).To(Equal([]byte("21")))

			test = FromInterface(int(15))
			Expect(test.Type).To(Equal(Int))
			Expect(test.IntVal).To(Equal(int64(15)))
			Expect(test.MarshalJSON()).To(Equal([]byte("15")))

			test = FromInterface(int64(32))
			Expect(test.Type).To(Equal(Int))
			Expect(test.IntVal).To(Equal(int64(32)))
			Expect(test.MarshalJSON()).To(Equal([]byte("32")))

			test = FromInterface(int32(321))
			Expect(test.Type).To(Equal(Int))
			Expect(test.IntVal).To(Equal(int64(321)))
			Expect(test.MarshalJSON()).To(Equal([]byte("321")))

			var test2 ConfigValue
			Expect(test2.UnmarshalJSON([]byte("43"))).To(Succeed())
			Expect(test2.Type).To(Equal(Int))
			Expect(test2.IntVal).To(Equal(int64(43)))
			Expect(test2.MarshalJSON()).To(Equal([]byte("43")))

			test3 := test.DeepCopy()
			Expect(test3.Type).To(Equal(Int))
			Expect(test3.IntVal).To(Equal(int64(321)))
			Expect(test3.MarshalJSON()).To(Equal([]byte("321")))
		})

		It("Float Type", func() {
			test := FromFloat64(2.2)
			Expect(test.Type).To(Equal(Float))
			Expect(test.FloatVal).To(Equal(float64(2.2)))
			Expect(test.MarshalJSON()).To(Equal([]byte("2.2")))

			test = FromInterface(float64(5.3))
			Expect(test.Type).To(Equal(Float))
			Expect(test.FloatVal).To(Equal(float64(5.3)))
			Expect(test.MarshalJSON()).To(Equal([]byte("5.3")))

			test = FromInterface(float32(123.5))
			Expect(test.Type).To(Equal(Float))
			Expect(test.FloatVal).To(Equal(float64(123.5)))
			Expect(test.MarshalJSON()).To(Equal([]byte("123.5")))

			var test2 ConfigValue
			Expect(test2.UnmarshalJSON([]byte("1.8"))).To(Succeed())
			Expect(test2.Type).To(Equal(Float))
			Expect(test2.FloatVal).To(Equal(float64(1.8)))
			Expect(test2.MarshalJSON()).To(Equal([]byte("1.8")))

			test3 := test.DeepCopy()
			Expect(test3.Type).To(Equal(Float))
			Expect(test3.FloatVal).To(Equal(float64(123.5)))
			Expect(test3.MarshalJSON()).To(Equal([]byte("123.5")))
		})
	})

	Context("When Building Somplex Configuration Values", func() {
		It("Slice Type", func() {
			test := FromInterface([]intstr.IntOrString{})
			Expect(test.Type).To(Equal(Array))
			Expect(test.ArrayVal).To(Equal([]intstr.IntOrString{}))
			Expect(test.MarshalJSON()).To(Equal([]byte("[]")))

			tar := make([]ConfigValue, 3)
			tar[0] = FromInt64(52)
			tar[1] = FromString("sdf")
			tar[2] = FromInt64(20)
			test = FromSlice(tar)
			Expect(test.Type).To(Equal(Array))
			ref := make([]intstr.IntOrString, 3)
			ref[0] = intstr.FromInt(52)
			ref[1] = intstr.FromString("sdf")
			ref[2] = intstr.FromInt(20)
			Expect(test.ArrayVal).To(Equal(ref))

			var test2 ConfigValue
			Expect(test2.UnmarshalJSON([]byte("[1, 2, \"sdf\"]"))).To(Succeed())
			Expect(test2.Type).To(Equal(Array))
			ref = make([]intstr.IntOrString, 3)
			ref[0] = intstr.FromInt(1)
			ref[1] = intstr.FromInt(2)
			ref[2] = intstr.FromString("sdf")
			Expect(test2.ArrayVal).To(Equal(ref))
			Expect(test2.MarshalJSON()).To(Equal([]byte("[1,2,\"sdf\"]")))

			test3 := test2.DeepCopy()
			Expect(test3.Type).To(Equal(Array))
			ref = make([]intstr.IntOrString, 3)
			ref[0] = intstr.FromInt(1)
			ref[1] = intstr.FromInt(2)
			ref[2] = intstr.FromString("sdf")
			Expect(test3.ArrayVal).To(Equal(ref))
			Expect(test3.MarshalJSON()).To(Equal([]byte("[1,2,\"sdf\"]")))
		})

		It("Map Type", func() {
			test := FromInterface(map[string]intstr.IntOrString{})
			Expect(test.Type).To(Equal(Map))
			Expect(test.MapVal).To(Equal(map[string]intstr.IntOrString{}))
			Expect(test.MarshalJSON()).To(Equal([]byte("{}")))

			var test2 ConfigValue
			Expect(test2.UnmarshalJSON([]byte("{\"a\":34, \"f\":2, \"d\":\"goodie\"}"))).To(Succeed())
			Expect(test2.Type).To(Equal(Map))
			ref := make(map[string]intstr.IntOrString, 3)
			ref["a"] = intstr.FromInt(34)
			ref["f"] = intstr.FromInt(2)
			ref["d"] = intstr.FromString("goodie")
			Expect(test2.MapVal).To(Equal(ref))
			Expect(test2.MarshalJSON()).To(Equal([]byte("{\"a\":34,\"d\":\"goodie\",\"f\":2}")))

			test3 := test2.DeepCopy()
			Expect(test3.Type).To(Equal(Map))
			ref = make(map[string]intstr.IntOrString, 3)
			ref["a"] = intstr.FromInt(34)
			ref["f"] = intstr.FromInt(2)
			ref["d"] = intstr.FromString("goodie")
			Expect(test3.MapVal).To(Equal(ref))
			Expect(test3.MarshalJSON()).To(Equal([]byte("{\"a\":34,\"d\":\"goodie\",\"f\":2}")))
		})
	})
}

func TestStructureNeedsJob(t *testing.T) {
	// change in state and no existing job
	// not when state does not change
	// not shen already has job
}
