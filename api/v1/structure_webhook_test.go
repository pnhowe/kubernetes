/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"fmt"
	"strconv"
	"time"

	cinp "github.com/cinp/go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	contractorClient "github.com/t3kton/contractor_goclient"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"t3kton.com/pkg/contractor"
	"t3kton.com/pkg/contractor/test_contractor"
)

var _ = Describe("Structure Webhook", func() {
	const (
		resourceName  = "test-structure"
		namespaceName = "default"
	)

	var (
		mockCtrl                                                                     *gomock.Controller
		mockCINP                                                                     *test_contractor.MockCInPClient
		mockStructure                                                                *contractorClient.BuildingStructure
		mockFoundation                                                               *contractorClient.BuildingFoundation
		mockJob                                                                      *contractorClient.ForemanStructureJob
		mockStructureBluePrint                                                       *contractorClient.BlueprintStructureBluePrint
		mockJobScriptName                                                            string
		mockStructureState                                                           string
		mockJobID                                                                    int
		uri                                                                          *cinp.URI
		doGetStructure, doGetFoudation, doGetJob, doFindJob, doGetStructureBluePrint *gomock.Call
		doGetInvalidStructure, doGetInvalidStructureBluePrint                        *gomock.Call
	)

	// typeNamespacedName := types.NamespacedName{
	// 	Name:      resourceName,
	// 	Namespace: namespaceName,
	// }

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCINP = test_contractor.NewMockCInPClient(mockCtrl)
		Expect(contractor.SetupTestingFactory(ctx, mockCINP)).NotTo(HaveOccurred())

		client := contractor.GetClient(ctx)

		mockStructureState = "planned"

		mockStructure = client.BuildingStructureNewWithID(123)
		mockStructure.ID = cinp.IntAddr(123)
		mockStructure.Hostname = cinp.StringAddr("testing")
		mockStructure.State = &mockStructureState
		mockStructure.Blueprint = cinp.StringAddr("/api/v1/BluePrint/StructureBluePrint:test-structure-base:")
		mockStructure.Foundation = cinp.StringAddr("/api/v1/Building/Foundation:test:")
		mockStructure.ConfigValues = &map[string]interface{}{"a": "asdf", "b": 12, "c": 2.1}

		mockFoundation = client.BuildingFoundationNewWithID("test")
		mockFoundation.Locator = cinp.StringAddr("test")
		mockFoundation.Blueprint = cinp.StringAddr("/api/v1/BluePrint/FoundationBluePrint:test-foundation-base:")

		mockJobID = 37
		mockJobScriptName = "Create"

		mockJob = client.ForemanStructureJobNewWithID(mockJobID)
		mockJob.ID = &mockJobID
		mockJob.Status = cinp.StringAddr("magic")
		mockJob.State = cinp.StringAddr("waiting")
		mockJob.ScriptName = &mockJobScriptName
		mockJob.Message = cinp.StringAddr("Just doing the thing")
		mockJob.CanStart = cinp.StringAddr("true")
		mockJob.Created = TimeAddr(time.Now())
		mockJob.Updated = TimeAddr(time.Now())

		mockStructureBluePrint = client.BlueprintStructureBluePrintNewWithID("test-structure-base")
		mockStructureBluePrint.Name = cinp.StringAddr("test-structure-base")

		var err error
		uri, err = cinp.NewURI("/api/v1/")
		Expect(err).NotTo(HaveOccurred())

		mockCINP.EXPECT().GetURI().Return(uri).AnyTimes()

		// testing Get Structure
		doGetStructure = mockCINP.EXPECT().
			Get(gomock.Any(), gomock.Eq("/api/v1/Building/Structure:123:")).
			DoAndReturn(func(_ context.Context, _ string) (*cinp.Object, error) {
				result := cinp.Object(mockStructure)
				return &result, nil
			})

		doGetInvalidStructure = mockCINP.EXPECT().
			Get(gomock.Any(), gomock.Eq("/api/v1/Building/Structure:54321:")).
			DoAndReturn(func(_ context.Context, _ string) (*cinp.Object, error) {
				return nil, fmt.Errorf("Not found")
			})

		// testing Get Foundation
		doGetFoudation = mockCINP.EXPECT().
			Get(gomock.Any(), gomock.Eq("/api/v1/Building/Foundation:testing:")).
			DoAndReturn(func(_ context.Context, _ string) (*cinp.Object, error) {
				result := cinp.Object(mockFoundation)
				return &result, nil
			})

		// testing Get Job
		doGetJob = mockCINP.EXPECT().
			Get(gomock.Any(), gomock.AnyOf("/api/v1/Foreman/StructureJob:37:", "/api/v1/Foreman/StructureJob:38:")).
			DoAndReturn(func(_ context.Context, _ string) (*cinp.Object, error) {
				result := cinp.Object(mockJob)
				return &result, nil
			})

		// testing Get Structure Blueprint
		doGetStructureBluePrint = mockCINP.EXPECT().
			Get(gomock.Any(), gomock.Eq("/api/v1/BluePrint/StructureBluePrint:test-structure-base:")).
			DoAndReturn(func(_ context.Context, _ string) (*cinp.Object, error) {
				result := cinp.Object(mockStructureBluePrint)
				return &result, nil
			})

		doGetInvalidStructureBluePrint = mockCINP.EXPECT().
			Get(gomock.Any(), gomock.Eq("/api/v1/BluePrint/StructureBluePrint:not-right:")).
			DoAndReturn(func(_ context.Context, _ string) (*cinp.Object, error) {
				return nil, fmt.Errorf("Not found")
			})

		// testing Get Job
		doFindJob = mockCINP.EXPECT().
			Call(gomock.Any(), gomock.Eq("/api/v1/Building/Structure:42:(getJob)"), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ *map[string]interface{}, result *string) error {
				if mockJobID > 0 {
					*result = *cinp.StringAddr("/api/v1/Foreman/StructureJob:" + strconv.Itoa(mockJobID) + ":")
				} else {
					*result = ""
				}
				return nil
			})

	})

	Context("When creating Structure under Defaulting Webhook", func() {
		It("Should deal with missing id", func() {
			By("Defaulter Setup")
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{},
			}

			doGetStructure.Times(0)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(0)
			doGetInvalidStructure.Times(0)
			doGetInvalidStructureBluePrint.Times(0)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(0))
			Expect(structure.Spec.BluePrint).To(Equal(""))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))

			By("Call Default")
			structure.Default()

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(0))
			Expect(structure.Spec.BluePrint).To(Equal(""))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))
		})

		It("Should deal with invalid id", func() {
			By("Defaulter Setup")
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{ID: 54321},
			}

			doGetStructure.Times(0)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(0)
			doGetInvalidStructure.Times(1)
			doGetInvalidStructureBluePrint.Times(0)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(54321))
			Expect(structure.Spec.BluePrint).To(Equal(""))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))

			By("Call Default")
			structure.Default()

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(54321))
			Expect(structure.Spec.BluePrint).To(Equal(""))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))
		})

		It("Should fill in the default value if a required field is empty", func() {
			By("Defaulter Setup")
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{ID: 123},
			}

			doGetStructure.Times(1)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(0)
			doGetInvalidStructure.Times(0)
			doGetInvalidStructureBluePrint.Times(0)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(123))
			Expect(structure.Spec.BluePrint).To(Equal(""))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))

			By("Call Default")
			structure.Default()

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(123))
			Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(3))
			Expect(structure.Spec.ConfigValues["a"]).To(Equal(FromString("asdf")))
			Expect(structure.Spec.ConfigValues["b"]).To(Equal(FromInt64(12)))
			Expect(structure.Spec.ConfigValues["c"]).To(Equal(FromFloat64(2.1)))
			Expect(structure.Spec.State).To(Equal("planned"))
		})

		It("Should fall through if state, blueprint, configValues are set", func() {
			By("Defaulter Setup")
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{
					ID:           123,
					State:        "built",
					BluePrint:    "structure-non-base",
					ConfigValues: map[string]ConfigValue{"2": FromString("Bob")},
				},
			}

			doGetStructure.Times(0)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(0)
			doGetInvalidStructure.Times(0)
			doGetInvalidStructureBluePrint.Times(0)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(123))
			Expect(structure.Spec.BluePrint).To(Equal("structure-non-base"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(1))
			Expect(structure.Spec.ConfigValues["2"]).To(Equal(FromString("Bob")))
			Expect(structure.Spec.State).To(Equal("built"))

			By("Call Default")
			structure.Default()

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(123))
			Expect(structure.Spec.BluePrint).To(Equal("structure-non-base"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(1))
			Expect(structure.Spec.ConfigValues["2"]).To(Equal(FromString("Bob")))
			Expect(structure.Spec.State).To(Equal("built"))

		})
	})

	Context("When creating Structure under Validating Webhook", func() {
		It("Should deal with missing values", func() {
			By("ValidateCreate Setup")
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{},
			}

			doGetStructure.Times(0)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(0)
			doGetInvalidStructure.Times(0)
			doGetInvalidStructureBluePrint.Times(0)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(0))
			Expect(structure.Spec.BluePrint).To(Equal(""))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))

			By("Call ValidateCreate")
			warn, err := structure.ValidateCreate()
			Expect(warn).To(BeNil())
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("[ID not specified, blueprint not specified]"))

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(0))
			Expect(structure.Spec.BluePrint).To(Equal(""))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))
		})

		It("Should deal with invalid values", func() {
			By("ValidateCreate Setup")
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{
					ID:        54321,
					BluePrint: "not-right",
				},
			}

			doGetStructure.Times(0)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(0)
			doGetInvalidStructure.Times(1)
			doGetInvalidStructureBluePrint.Times(1)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(54321))
			Expect(structure.Spec.BluePrint).To(Equal("not-right"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))

			By("Call ValidateCreate")
			warn, err := structure.ValidateCreate()
			Expect(warn).To(BeNil())
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("[structure not found, blueprint not found]"))

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(54321))
			Expect(structure.Spec.BluePrint).To(Equal("not-right"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))
		})

		It("Should deal with invalid config values", func() {
			By("ValidateCreate Setup")
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{
					ID:        123,
					BluePrint: "test-structure-base",
					ConfigValues: map[string]ConfigValue{
						"a:>test": FromInt64(1),
					},
				},
			}

			doGetStructure.Times(1)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(1)
			doGetInvalidStructure.Times(0)
			doGetInvalidStructureBluePrint.Times(0)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(123))
			Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(1))
			Expect(structure.Spec.ConfigValues["a:>test"]).To(Equal(FromInt64(1)))
			Expect(structure.Spec.State).To(Equal(""))

			By("Call ValidateCreate")
			warn, err := structure.ValidateCreate()
			Expect(warn).To(BeNil())
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("invalid configuration value name 'a:>test'"))

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(123))
			Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(1))
			Expect(structure.Spec.ConfigValues["a:>test"]).To(Equal(FromInt64(1)))
			Expect(structure.Spec.State).To(Equal(""))
		})

		It("Should deal all valid values", func() {
			By("ValidateCreate Setup")
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{
					ID:        123,
					BluePrint: "test-structure-base",
					ConfigValues: map[string]ConfigValue{
						"a":      FromInt64(1),
						"a:test": FromInt64(1),
						">test":  FromInt64(1),
						"stuff":  FromInt64(1),
					},
				},
			}

			doGetStructure.Times(1)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(1)
			doGetInvalidStructure.Times(0)
			doGetInvalidStructureBluePrint.Times(0)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(123))
			Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(4))
			Expect(structure.Spec.ConfigValues["a"]).To(Equal(FromInt64(1)))
			Expect(structure.Spec.ConfigValues["a:test"]).To(Equal(FromInt64(1)))
			Expect(structure.Spec.ConfigValues[">test"]).To(Equal(FromInt64(1)))
			Expect(structure.Spec.ConfigValues["stuff"]).To(Equal(FromInt64(1)))
			Expect(structure.Spec.State).To(Equal(""))

			By("Call ValidateCreate")
			warn, err := structure.ValidateCreate()
			Expect(warn).To(BeNil())
			Expect(err).To(BeNil())

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(123))
			Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(4))
			Expect(structure.Spec.ConfigValues["a"]).To(Equal(FromInt64(1)))
			Expect(structure.Spec.ConfigValues["a:test"]).To(Equal(FromInt64(1)))
			Expect(structure.Spec.ConfigValues[">test"]).To(Equal(FromInt64(1)))
			Expect(structure.Spec.ConfigValues["stuff"]).To(Equal(FromInt64(1)))
			Expect(structure.Spec.State).To(Equal(""))
		})
	})

	Context("When changing Structure under Validating Webhook", func() {
		It("Should fall through with valid values, and nothing changing", func() {
			By("ValidateUpdate Setup")
			oldStructure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{
					ID:        123,
					BluePrint: "test-structure-base",
				},
			}
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{
					ID:        123,
					BluePrint: "test-structure-base",
				},
			}

			doGetStructure.Times(1)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(1)
			doGetInvalidStructure.Times(0)
			doGetInvalidStructureBluePrint.Times(0)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(123))
			Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))

			By("Call ValidateUpdate")
			warn, err := structure.ValidateUpdate(oldStructure)
			Expect(warn).To(BeNil())
			Expect(err).To(BeNil())

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(123))
			Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))
		})

		It("Should handle when required values are removed", func() {
			By("ValidateUpdate Setup")
			oldStructure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{
					ID:        123,
					BluePrint: "test-structure-base",
				},
			}
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{},
			}

			doGetStructure.Times(0)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(0)
			doGetInvalidStructure.Times(0)
			doGetInvalidStructureBluePrint.Times(0)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(0))
			Expect(structure.Spec.BluePrint).To(Equal(""))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))

			By("Call ValidateUpdate")
			warn, err := structure.ValidateUpdate(oldStructure)
			Expect(warn).To(BeNil())
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("[ID not specified, blueprint not specified, can not change the ID, can not change the BluePrint while not in 'Planned' State]"))

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(0))
			Expect(structure.Spec.BluePrint).To(Equal(""))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))
		})

		It("Should handle when required values are invalid", func() {
			By("ValidateUpdate Setup")
			oldStructure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{
					ID:        123,
					BluePrint: "test-structure-base",
				},
			}
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{
					ID:        54321,
					BluePrint: "not-right",
				},
			}

			doGetStructure.Times(0)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(0)
			doGetInvalidStructure.Times(1)
			doGetInvalidStructureBluePrint.Times(1)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(54321))
			Expect(structure.Spec.BluePrint).To(Equal("not-right"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))

			By("Call ValidateUpdate")
			warn, err := structure.ValidateUpdate(oldStructure)
			Expect(warn).To(BeNil())
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("[structure not found, blueprint not found, can not change the ID, can not change the BluePrint while not in 'Planned' State]"))

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(54321))
			Expect(structure.Spec.BluePrint).To(Equal("not-right"))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))
		})
	})

	It("Can only change blueprint when state is planned", func() {
		By("ValidateUpdate Setup")
		oldStructure := &Structure{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespaceName,
			},
			Spec: StructureSpec{
				ID:        123,
				BluePrint: "old-test-structure-base",
			},
			Status: StructureStatus{
				State: "planned",
			},
		}
		structure := &Structure{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespaceName,
			},
			Spec: StructureSpec{
				ID:        123,
				BluePrint: "test-structure-base",
			},
			Status: StructureStatus{
				State: "planned",
			},
		}

		doGetStructure.Times(9)
		doGetFoudation.Times(0)
		doGetJob.Times(0)
		doFindJob.Times(0)
		doGetStructureBluePrint.Times(9)
		doGetInvalidStructure.Times(0)
		doGetInvalidStructureBluePrint.Times(0)

		By("Checking Spec Before")
		Expect(structure.Spec.ID).To(Equal(123))
		Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
		Expect(structure.Spec.ConfigValues).To(HaveLen(0))
		Expect(structure.Spec.State).To(Equal(""))

		By("Call ValidateUpdate")
		warn, err := structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the BluePrint while not in 'Planned' State"))

		By("Setting new Spec State to planned")
		structure.Spec.State = "planned"
		oldStructure.Spec.State = "built"

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the BluePrint while not in 'Planned' State"))

		By("Setting old Spec State to planned")
		structure.Spec.State = "built"
		oldStructure.Spec.State = "planned"

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the BluePrint while not in 'Planned' State"))

		By("Setting both Spec State to planned")
		structure.Spec.State = "planned"
		oldStructure.Spec.State = "planned"

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).To(BeNil())

		By("Setting both Spec State to built")
		structure.Spec.State = "built"
		oldStructure.Spec.State = "built"

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the BluePrint while not in 'Planned' State"))

		By("Setting old Status State to built")
		structure.Spec.State = "planned"
		structure.Status.State = "planned"
		oldStructure.Spec.State = "planned"
		oldStructure.Status.State = "built"

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the BluePrint while not in 'Planned' State"))

		By("Setting new Status State to built")
		structure.Spec.State = "planned"
		structure.Status.State = "built"
		oldStructure.Spec.State = "planned"
		oldStructure.Status.State = "planned"

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the BluePrint while not in 'Planned' State"))

		By("Setting both Status State to built")
		structure.Spec.State = "planned"
		structure.Status.State = "built"
		oldStructure.Spec.State = "planned"
		oldStructure.Status.State = "built"

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the BluePrint while not in 'Planned' State"))

		By("Setting both Status State to planned")
		structure.Spec.State = "planned"
		structure.Status.State = "planned"
		oldStructure.Spec.State = "planned"
		oldStructure.Status.State = "planned"

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).To(BeNil())

		By("Checking Spec After")
		Expect(structure.Spec.ID).To(Equal(123))
		Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
		Expect(structure.Spec.ConfigValues).To(HaveLen(0))
		Expect(structure.Spec.State).To(Equal("planned"))
	})

	It("Can only change blueprint when there is no job", func() {
		By("ValidateUpdate Setup")
		oldStructure := &Structure{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespaceName,
			},
			Spec: StructureSpec{
				ID:        123,
				BluePrint: "old-test-structure-base",
				State:     "planned",
			},
			Status: StructureStatus{
				State: "planned",
			},
		}
		structure := &Structure{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespaceName,
			},
			Spec: StructureSpec{
				ID:        123,
				BluePrint: "test-structure-base",
				State:     "planned",
			},
			Status: StructureStatus{
				State: "planned",
			},
		}

		doGetStructure.Times(4)
		doGetFoudation.Times(0)
		doGetJob.Times(0)
		doFindJob.Times(0)
		doGetStructureBluePrint.Times(4)
		doGetInvalidStructure.Times(0)
		doGetInvalidStructureBluePrint.Times(0)

		By("Checking Spec Before")
		Expect(structure.Spec.ID).To(Equal(123))
		Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
		Expect(structure.Spec.ConfigValues).To(HaveLen(0))
		Expect(structure.Spec.State).To(Equal("planned"))

		By("Call ValidateUpdate")
		warn, err := structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).To(BeNil())

		structure.Status.Job = &JobStatus{}

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the BluePrint while there is a Job"))

		structure.Status.Job = &JobStatus{}
		oldStructure.Status.Job = &JobStatus{}

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the BluePrint while there is a Job"))

		structure.Status.Job = nil

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the BluePrint while there is a Job"))

		By("Checking Spec After")
		Expect(structure.Spec.ID).To(Equal(123))
		Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
		Expect(structure.Spec.ConfigValues).To(HaveLen(0))
		Expect(structure.Spec.State).To(Equal("planned"))
	})

	It("Can only change requested state when there is no job", func() {
		By("ValidateUpdate Setup")
		oldStructure := &Structure{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespaceName,
			},
			Spec: StructureSpec{
				ID:        123,
				BluePrint: "test-structure-base",
				State:     "planned",
			},
			Status: StructureStatus{
				State: "planned",
			},
		}
		structure := &Structure{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespaceName,
			},
			Spec: StructureSpec{
				ID:        123,
				BluePrint: "test-structure-base",
				State:     "built",
			},
			Status: StructureStatus{
				State: "planned",
			},
		}

		doGetStructure.Times(8)
		doGetFoudation.Times(0)
		doGetJob.Times(0)
		doFindJob.Times(0)
		doGetStructureBluePrint.Times(8)
		doGetInvalidStructure.Times(0)
		doGetInvalidStructureBluePrint.Times(0)

		By("Checking Spec Before")
		Expect(structure.Spec.ID).To(Equal(123))
		Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
		Expect(structure.Spec.ConfigValues).To(HaveLen(0))
		Expect(structure.Spec.State).To(Equal("built"))

		By("Call ValidateUpdate")
		warn, err := structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).To(BeNil())

		structure.Status.Job = &JobStatus{}

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the State while there is a Job"))

		structure.Status.Job = &JobStatus{}
		oldStructure.Status.Job = &JobStatus{}

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the State while there is a Job"))

		structure.Status.Job = nil

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("can not change the State while there is a Job"))

		// The Status State should not be affected buy the job
		structure.Spec.State = "planned"
		structure.Status.State = "built"

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).To(BeNil())

		structure.Status.Job = &JobStatus{}

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).To(BeNil())

		structure.Status.Job = &JobStatus{}
		oldStructure.Status.Job = &JobStatus{}

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).To(BeNil())

		structure.Status.Job = nil

		By("Call ValidateUpdate")
		warn, err = structure.ValidateUpdate(oldStructure)
		Expect(warn).To(BeNil())
		Expect(err).To(BeNil())

		By("Checking Spec After")
		Expect(structure.Spec.ID).To(Equal(123))
		Expect(structure.Spec.BluePrint).To(Equal("test-structure-base"))
		Expect(structure.Spec.ConfigValues).To(HaveLen(0))
		Expect(structure.Spec.State).To(Equal("planned"))
	})

	Context("When deleting strusture", func() {
		It("Just fall through for now", func() {
			By("ValidateDelete Setup")
			structure := &Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: StructureSpec{},
			}

			doGetStructure.Times(0)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doGetStructureBluePrint.Times(0)
			doGetInvalidStructure.Times(0)
			doGetInvalidStructureBluePrint.Times(0)

			By("Checking Spec Before")
			Expect(structure.Spec.ID).To(Equal(0))
			Expect(structure.Spec.BluePrint).To(Equal(""))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))

			By("Call ValidateDelete")
			warn, err := structure.ValidateDelete()
			Expect(warn).To(BeNil())
			Expect(err).To(BeNil())

			By("Checking Spec After")
			Expect(structure.Spec.ID).To(Equal(0))
			Expect(structure.Spec.BluePrint).To(Equal(""))
			Expect(structure.Spec.ConfigValues).To(HaveLen(0))
			Expect(structure.Spec.State).To(Equal(""))
		})
	})
})

func TimeAddr(v time.Time) *time.Time {
	return &v
}
