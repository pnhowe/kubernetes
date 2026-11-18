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

package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	cinp "github.com/cinp/go/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	contractorClient "github.com/t3kton/contractor_goclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	contractorv1 "t3kton.com/api/v1"
	"t3kton.com/pkg/contractor"
	"t3kton.com/pkg/contractor/test_contractor"
)

type buildingStructureMatcher struct {
	//contractorClient.BuildingStructure
	uri string
}

func BuildingStructureMatcher(uri string) gomock.Matcher {
	fmt.Println("Setting up Macher for", uri)
	return &buildingStructureMatcher{uri}
}

func (b *buildingStructureMatcher) Matches(x any) bool {
	switch y := x.(type) {
	case *contractorClient.BuildingStructure:
		return y.GetURI() == b.uri
	}
	return false
}

func (b *buildingStructureMatcher) String() string {
	return "with uri " + b.uri
}

const (
	scriptCreate  = "Create"
	scriptDestroy = "Destroy"
)

func TimeAddr(v time.Time) *time.Time {
	return &v
}

var _ = Describe("Structure Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName  = "test-structure"
			namespaceName = "default"
		)

		var (
			mockCtrl                                          *gomock.Controller
			mockCINP                                          *test_contractor.MockCInPClient
			mockStructure                                     *contractorClient.BuildingStructure
			mockFoundation                                    *contractorClient.BuildingFoundation
			mockJob                                           *contractorClient.ForemanStructureJob
			mockJobScriptName                                 string
			mockStructureState                                string
			mockJobID                                         int
			uri                                               *cinp.URI
			doGetStructure, doUpdateStructure, doGetFoudation *gomock.Call
			doCreateCall, doDestroyCall, doGetJob, doFindJob  *gomock.Call
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespaceName,
		}

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockCINP = test_contractor.NewMockCInPClient(mockCtrl)
			Expect(contractor.SetupTestingFactory(ctx, mockCINP)).NotTo(HaveOccurred())

			client, err := contractor.GetClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			mockStructureState = contractorv1.StatePlanned

			mockStructure = client.BuildingStructureNewWithID(42)
			mockStructure.ID = cinp.IntAddr(42)
			mockStructure.Hostname = cinp.StringAddr("testing")
			mockStructure.State = &mockStructureState
			mockStructure.Blueprint = cinp.StringAddr("/api/v1/BluePrint/StructureBluePrint:test-structure-base:")
			mockStructure.Foundation = cinp.StringAddr("/api/v1/Building/Foundation:test:")
			mockStructure.ConfigValues = &map[string]interface{}{}

			mockFoundation = client.BuildingFoundationNewWithID("test")
			mockFoundation.Locator = cinp.StringAddr("test")
			mockFoundation.Blueprint = cinp.StringAddr("/api/v1/BluePrint/FoundationBluePrint:test-foundation-base:")

			mockJobID = 37
			mockJobScriptName = scriptCreate

			mockJob = client.ForemanStructureJobNewWithID(mockJobID)
			mockJob.ID = &mockJobID
			mockJob.Status = &[]map[string]any{{"percent": 12.5, "operation": "Scope", "parameters": map[string]any{"time_elapsed": "00:02", "time_remaining": "00:07"}}}
			mockJob.State = cinp.StringAddr("waiting")
			mockJob.ScriptName = &mockJobScriptName
			mockJob.Message = cinp.StringAddr("Just doing the thing")
			mockJob.CanStart = cinp.StringAddr("true")
			mockJob.Created = TimeAddr(time.Now())
			mockJob.Updated = TimeAddr(time.Now())

			uri, err = cinp.NewURI("/api/v1/")
			Expect(err).NotTo(HaveOccurred())

			mockCINP.EXPECT().GetURI().Return(uri).AnyTimes()

			// testing Get Structure
			doGetStructure = mockCINP.EXPECT().
				Get(gomock.Any(), gomock.Eq("/api/v1/Building/Structure:42:")).
				DoAndReturn(func(_ context.Context, _ string) (*cinp.Object, error) {
					result := cinp.Object(mockStructure)
					return &result, nil
				})

			// testing Update
			doUpdateStructure = mockCINP.EXPECT().
				Update(gomock.Any(), BuildingStructureMatcher("/api/v1/Building/Structure:42:")).
				DoAndReturn(func(_ context.Context, _ *contractorClient.BuildingStructure) error {
					return nil
				})

			// testing Get Foundation
			doGetFoudation = mockCINP.EXPECT().
				Get(gomock.Any(), gomock.Eq("/api/v1/Building/Foundation:test:")).
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

			// testing DoCreate
			doCreateCall = mockCINP.EXPECT().
				Call(gomock.Any(), gomock.Eq("/api/v1/Building/Structure:42:(doCreate)"), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, _ *map[string]interface{}, result *int) error {
					*result = 37
					mockJobID = 37
					mockJobScriptName = scriptCreate
					return nil
				})

			// testing doDestroy
			doDestroyCall = mockCINP.EXPECT().
				Call(gomock.Any(), gomock.Eq("/api/v1/Building/Structure:42:(doDestroy)"), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, _ *map[string]interface{}, result *int) error {
					*result = 38
					mockJobID = 38
					mockJobScriptName = scriptDestroy
					return nil
				})
		})

		It("should successfully handle incomplete information", func() {
			By("creating the custom resource for the Kind Structure")
			req := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}

			controllerReconciler := &StructureReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: &events.FakeRecorder{},
			}

			By("Reconciling the before structure is made resource")
			result, err := controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(Equal(true))

			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: contractorv1.StructureSpec{ID: 42},
			}
			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				By("Cleanup the specific resource instance Structure")
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			doGetStructure.Times(0)
			doUpdateStructure.Times(0)
			doGetFoudation.Times(0)
			doGetJob.Times(0)
			doFindJob.Times(0)
			doCreateCall.Times(0)
			doDestroyCall.Times(0)

			// For now this is not testable b/c the structure spec does not allow ID to be 0
			// By("Reconciling the empty resource")
			// result, err = controllerReconciler.Reconcile(ctx, req)
			// Expect(err).To(MatchError("ID Not Specified"))
			// Expect(result.IsZero()).To(Equal(true))

			By("Reconciling the created resource with only ID")
			// still missing target state and blueprint
			structure.Spec = contractorv1.StructureSpec{ID: 42}
			Expect(k8sClient.Update(ctx, structure)).NotTo(HaveOccurred())

			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).To(MatchError("structure is not fully defined"))
			Expect(result.IsZero()).To(Equal(true))
		})

		It("fall through when it is already in the correct state(planned) and blueprint", func() {
			// this will just pull in the updated status
			By("creating the custom resource for the Kind Structure")
			var structure2 contractorv1.Structure
			req := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: contractorv1.StructureSpec{
					ID:        42,
					State:     contractorv1.StatePlanned,
					BluePrint: "test-structure-base",
				},
			}
			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				By("Cleanup the specific resource instance Structure")
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			controllerReconciler := &StructureReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: &events.FakeRecorder{},
			}

			mockJobID = 0

			doGetStructure.Times(2)
			doUpdateStructure.Times(0)
			doGetFoudation.Times(2)
			doGetJob.Times(0)
			doFindJob.Times(2)
			doCreateCall.Times(0)
			doDestroyCall.Times(0)

			By("Checking Status Before")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(BeZero())
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling") // this will fill in the status
			result, err := controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling Again") // should just fall through
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(Equal(true))

			By("Checking Status After")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job).To(BeNil())

		})

		It("fall through when it is already in the correct state(built) and blueprint", func() {
			// this will just pull in the updated status
			By("creating the custom resource for the Kind Structure")
			var structure2 contractorv1.Structure
			req := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: contractorv1.StructureSpec{
					ID:        42,
					State:     contractorv1.StateBuilt,
					BluePrint: "test-structure-base",
				},
			}
			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				By("Cleanup the specific resource instance Structure")
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			controllerReconciler := &StructureReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: &events.FakeRecorder{},
			}

			mockStructure.State = cinp.StringAddr(contractorv1.StateBuilt)

			mockJobID = 0

			doGetStructure.Times(2)
			doUpdateStructure.Times(0)
			doGetFoudation.Times(2)
			doGetJob.Times(0)
			doFindJob.Times(2)
			doCreateCall.Times(0)
			doDestroyCall.Times(0)

			By("Checking Status Before")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(BeZero())
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling") // this will fill in the status
			result, err := controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status After")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling Again") // should just fall through
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(Equal(true))

			By("Checking Status After")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job).To(BeNil())
		})

		It("creating the job when going from planned to built, no existing job", func() {
			// this will call create job, then we will make sure the job status get's filled in, then the job "finishes"
			By("creating the custom resource for the Kind Structure")
			var structure2 contractorv1.Structure
			req := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: contractorv1.StructureSpec{
					ID:        42,
					State:     contractorv1.StateBuilt,
					BluePrint: "test-structure-base",
				},
			}

			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				By("Cleanup the specific resource instance Structure")
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			structure.Status = contractorv1.StructureStatus{
				State:               contractorv1.StatePlanned,
				BluePrint:           "test-structure-base",
				Hostname:            "testing",
				Foundation:          "test",
				FoundationBluePrint: "test-foundation-base",
			}
			Expect(k8sClient.Status().Update(ctx, structure)).To(Succeed())

			controllerReconciler := &StructureReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: &events.FakeRecorder{},
			}

			mockStructureState = contractorv1.StatePlanned
			mockJobID = 0
			mockJobScriptName = ""

			doGetStructure.Times(6)
			doUpdateStructure.Times(0)
			doGetFoudation.Times(6)
			doGetJob.Times(2)
			doFindJob.Times(6)
			doCreateCall.Times(1)
			doDestroyCall.Times(0)

			By("Checking Status Before")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling") // should just create job
			result, err := controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))
			Expect(mockJobID).To(Equal(37))

			By("Checking Status Without Job")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling") // now we get the job status
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status With Job")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job.Script).To(Equal(scriptCreate))

			By("Reconciling") // now we get told to requeue in 30 seconds, letting the job run
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			By("Checking Status Waiting for Job")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job.Script).To(Equal(scriptCreate))

			mockJobID = 0

			By("Reconciling") // now the job is done, will update status to remove the job
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status Job is Gone")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job).To(BeNil())

			mockStructureState = contractorv1.StateBuilt

			By("Reconciling") // now the status is built and it will be done
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status With new State")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling") // now we should not get any requeuing
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(Equal(true))

			By("Checking Status After")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job).To(BeNil())
		})

		It("creating the job when going from built to planned, no existing job", func() {
			// this will call destroy job, then we will make sure the job status get's filled in
			By("creating the custom resource for the Kind Structure")
			var structure2 contractorv1.Structure
			req := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: contractorv1.StructureSpec{
					ID:        42,
					State:     contractorv1.StatePlanned,
					BluePrint: "test-structure-base",
				},
			}

			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				By("Cleanup the specific resource instance Structure")
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			structure.Status = contractorv1.StructureStatus{
				State:               contractorv1.StateBuilt,
				BluePrint:           "test-structure-base",
				Hostname:            "testing",
				Foundation:          "test",
				FoundationBluePrint: "test-foundation-base",
			}
			Expect(k8sClient.Status().Update(ctx, structure)).To(Succeed())

			controllerReconciler := &StructureReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: &events.FakeRecorder{},
			}

			mockStructureState = contractorv1.StateBuilt
			mockJobID = 0
			mockJobScriptName = ""

			doGetStructure.Times(6)
			doUpdateStructure.Times(0)
			doGetFoudation.Times(6)
			doGetJob.Times(2)
			doFindJob.Times(6)
			doCreateCall.Times(0)
			doDestroyCall.Times(1)

			By("Checking Status Before")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling") // should just create job
			result, err := controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))
			Expect(mockJobID).To(Equal(38))

			By("Checking Status Without Job")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling") // now we get the job status
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status With Job")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job.Script).To(Equal(scriptDestroy))

			By("Reconciling") // now we get told to requeue in 30 seconds, letting the job run
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			By("Checking Status Waiting for Job")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job.Script).To(Equal(scriptDestroy))

			mockJobID = 0

			By("Reconciling") // now the job is done, will update status to remove the job
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status Job is Gone")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job).To(BeNil())

			mockStructureState = contractorv1.StatePlanned

			By("Reconciling") // now the status is built and it will be done
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status With new State")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling") // now we should not get any requeuing
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(Equal(true))

			By("Checking Status After")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job).To(BeNil())
		})

		It("creating the job when going from planned to built, existing jobs", func() {
			// make sure we don't create a new job, both when the job is the expected job and not
			By("creating the custom resource for the Kind Structure")
			var structure2 contractorv1.Structure
			req := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: contractorv1.StructureSpec{
					ID:        42,
					State:     contractorv1.StateBuilt,
					BluePrint: "test-structure-base",
				},
			}

			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				By("Cleanup the specific resource instance Structure")
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			structure.Status = contractorv1.StructureStatus{
				State:               contractorv1.StatePlanned,
				BluePrint:           "test-structure-base",
				Hostname:            "testing",
				Foundation:          "test",
				FoundationBluePrint: "test-foundation-base",
			}
			Expect(k8sClient.Status().Update(ctx, structure)).To(Succeed())

			controllerReconciler := &StructureReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: &events.FakeRecorder{},
			}

			mockStructureState = contractorv1.StatePlanned
			mockJobID = 37

			doGetStructure.Times(4)
			doUpdateStructure.Times(0)
			doGetFoudation.Times(4)
			doGetJob.Times(4)
			doFindJob.Times(4)
			doCreateCall.Times(0)
			doDestroyCall.Times(0)

			By("Checking Status Before")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling") // we should get the job status
			result, err := controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status With Job (Create Job)")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job.Script).To(Equal(scriptCreate))

			By("Reconciling") // now we get told to requeue in 30 seconds, letting the job run
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			By("Checking Status After (Create Job)")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job.Script).To(Equal(scriptCreate))

			mockJobScriptName = scriptDestroy

			By("Reconciling") // should update the job status
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status With Job (Destroy Job)")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job.Script).To(Equal(scriptDestroy))

			By("Reconciling") // now we get told to requeue in 30 seconds, letting the job run
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			By("Checking Status After (Destroy Job)")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job.Script).To(Equal(scriptDestroy))
		})

		It("creating the job when going from built to planned, existing jobs", func() {
			// make sure we don't create a new job, both when the job is the expected job and not
			By("creating the custom resource for the Kind Structure")
			var structure2 contractorv1.Structure
			req := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: contractorv1.StructureSpec{
					ID:        42,
					State:     contractorv1.StatePlanned,
					BluePrint: "test-structure-base",
				},
			}

			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				By("Cleanup the specific resource instance Structure")
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			structure.Status = contractorv1.StructureStatus{
				State:               contractorv1.StateBuilt,
				BluePrint:           "test-structure-base",
				Hostname:            "testing",
				Foundation:          "test",
				FoundationBluePrint: "test-foundation-base",
			}
			Expect(k8sClient.Status().Update(ctx, structure)).To(Succeed())

			controllerReconciler := &StructureReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: &events.FakeRecorder{},
			}

			mockStructureState = contractorv1.StateBuilt
			mockJobID = 37
			mockJobScriptName = scriptDestroy

			doGetStructure.Times(4)
			doUpdateStructure.Times(0)
			doGetFoudation.Times(4)
			doGetJob.Times(4)
			doFindJob.Times(4)
			doCreateCall.Times(0)
			doDestroyCall.Times(0)

			By("Checking Status Before")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job).To(BeNil())

			By("Reconciling") // we should get the job status
			result, err := controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status With Job (Destroy Job)")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job.Script).To(Equal(scriptDestroy))

			By("Reconciling") // now we get told to requeue in 30 seconds, letting the job run
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			By("Checking Status After (Destroy Job)")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job.Script).To(Equal(scriptDestroy))

			mockJobScriptName = scriptCreate

			By("Reconciling") // should update the job status
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status With Job (Create Job)")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job.Script).To(Equal(scriptCreate))

			By("Reconciling") // now we get told to requeue in 30 seconds, letting the job run
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			By("Checking Status After (Create Job)")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StateBuilt))
			Expect(structure2.Status.Job.Script).To(Equal(scriptCreate))
		})

		// update blueprint

		It("should update configuration values when state is not changing", func() {
			//
			By("creating the custom resource for the Kind Structure")
			var structure2 contractorv1.Structure
			req := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}
			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: contractorv1.StructureSpec{
					ID:        42,
					State:     contractorv1.StatePlanned,
					BluePrint: "test-structure-base",
				},
			}
			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				By("Cleanup the specific resource instance Structure")
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			structure.Status = contractorv1.StructureStatus{
				State:               contractorv1.StatePlanned,
				BluePrint:           "test-structure-base",
				Hostname:            "testing",
				Foundation:          "test",
				FoundationBluePrint: "test-foundation-base",
			}
			Expect(k8sClient.Status().Update(ctx, structure)).To(Succeed())

			controllerReconciler := &StructureReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: &events.FakeRecorder{},
			}

			mockStructureState = contractorv1.StatePlanned
			mockJobID = 0

			doGetStructure.Times(2)
			doUpdateStructure.Times(1)
			doGetFoudation.Times(2)
			doGetJob.Times(0)
			doFindJob.Times(2)
			doCreateCall.Times(0)
			doDestroyCall.Times(0)

			By("Checking Status Before")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job).To(BeNil())
			Expect(structure2.Spec.ConfigValues).To(BeNil())

			By("Reconciling") // should just fall through
			result, err := controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(Equal(true))

			By("Checking Status Before Setting Values")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job).To(BeNil())
			Expect(structure2.Spec.ConfigValues).To(BeNil())

			By("Setting Config values")
			structure.Spec.ConfigValues = contractorv1.ConfigValues{}
			structure.Spec.ConfigValues["test"] = contractorv1.NewConfigValue("asdf")
			structure.Spec.ConfigValues["test2"] = contractorv1.NewConfigValue(42)
			Expect(k8sClient.Update(ctx, structure)).To(Succeed())

			By("Reconciling") // update config values
			result, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(requeueSoon))

			By("Checking Status After Setting Values")
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Status.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Status.Job).To(BeNil())
			Expect(structure2.Spec.ConfigValues).ToNot(BeNil())
		})
		It("follow mode updates the spec from contractor and never changes contractor", func() {
			req := reconcile.Request{NamespacedName: typeNamespacedName}
			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespaceName},
				Spec: contractorv1.StructureSpec{
					ID:           42,
					State:        contractorv1.StateBuilt,
					BluePrint:    "some-other-blueprint",
					ConfigValues: contractorv1.ConfigValues{"mine": contractorv1.NewConfigValue("gone")},
				},
			}
			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			mockStructureState = contractorv1.StatePlanned
			mockJobID = 0
			mockStructure.ConfigValues = &map[string]interface{}{"fromcontractor": 7.0}

			doGetStructure.AnyTimes()
			doGetFoudation.AnyTimes()
			doFindJob.AnyTimes()
			doGetJob.AnyTimes()
			doUpdateStructure.Times(0)
			doCreateCall.Times(0)
			doDestroyCall.Times(0)

			controllerReconciler := &StructureReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				Recorder:       &events.FakeRecorder{},
				SyncMode:       SyncModeFollow,
				ResyncInterval: 5 * time.Minute,
			}

			var result reconcile.Result
			var err error
			for i := 0; i < 10; i++ { // status, then state, then blueprint+config, each requeues
				result, err = controllerReconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				if result.RequeueAfter != requeueSoon {
					break
				}
			}
			Expect(result.RequeueAfter).To(Equal(5 * time.Minute))

			var structure2 contractorv1.Structure
			Expect(k8sClient.Get(ctx, typeNamespacedName, &structure2)).NotTo(HaveOccurred())
			Expect(structure2.Spec.State).To(Equal(contractorv1.StatePlanned))
			Expect(structure2.Spec.BluePrint).To(Equal("test-structure-base"))
			Expect(structure2.Spec.ConfigValues).To(HaveLen(1))
			Expect(structure2.Spec.ConfigValues).To(HaveKey("fromcontractor"))
		})

		It("destroys a built structure when the blueprint changes, instead of trying to create it", func() {
			req := reconcile.Request{NamespacedName: typeNamespacedName}
			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespaceName},
				Spec: contractorv1.StructureSpec{
					ID:           42,
					State:        contractorv1.StateBuilt,
					BluePrint:    "some-other-blueprint",
					ConfigValues: contractorv1.ConfigValues{},
				},
			}
			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			mockStructureState = contractorv1.StateBuilt
			mockJobID = 0

			doGetStructure.AnyTimes()
			doGetFoudation.AnyTimes()
			doFindJob.AnyTimes()
			doGetJob.AnyTimes()
			doUpdateStructure.Times(0)
			doCreateCall.Times(0)
			doDestroyCall.Times(1)

			controllerReconciler := &StructureReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: &events.FakeRecorder{},
			}

			for i := 0; i < 3 && mockJobID == 0; i++ { // first one fills in the status
				_, err := controllerReconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(mockJobID).To(Equal(38))
		})

		It("schedules the next resync when idle", func() {
			req := reconcile.Request{NamespacedName: typeNamespacedName}
			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespaceName},
				Spec: contractorv1.StructureSpec{
					ID:           42,
					State:        contractorv1.StatePlanned,
					BluePrint:    "test-structure-base",
					ConfigValues: contractorv1.ConfigValues{},
				},
			}
			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			mockJobID = 0
			doGetStructure.AnyTimes()
			doGetFoudation.AnyTimes()
			doFindJob.AnyTimes()
			doGetJob.AnyTimes()
			doUpdateStructure.Times(0)
			doCreateCall.Times(0)
			doDestroyCall.Times(0)

			controllerReconciler := &StructureReconciler{
				Client:         k8sClient,
				Scheme:         k8sClient.Scheme(),
				Recorder:       &events.FakeRecorder{},
				ResyncInterval: 5 * time.Minute,
			}

			var result reconcile.Result
			var err error
			for i := 0; i < 5; i++ {
				result, err = controllerReconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				if result.RequeueAfter != requeueSoon {
					break
				}
			}
			Expect(result.RequeueAfter).To(Equal(5 * time.Minute))
		})
	})
})
