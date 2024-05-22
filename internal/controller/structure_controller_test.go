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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	contractorv1 "t3kton.com/api/v1"
	"t3kton.com/pkg/contractor"
	"t3kton.com/pkg/contractor/test_contractor"
)

var _ = Describe("Structure Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName  = "test-structure"
			namespaceName = "default"
		)

		var mockCtrl *gomock.Controller
		var mockCINP *test_contractor.MockCInPClient

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespaceName,
		}

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockCINP = test_contractor.NewMockCInPClient(mockCtrl)
		})

		It("should successfully reconcile from planned to built", func() {
			By("Setup Contractor Client Factory")
			Expect(contractor.SetupTestingFactory(ctx, mockCINP)).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind Structure")
			structure := &contractorv1.Structure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
				},
				Spec: contractorv1.StructureSpec{},
			}
			Expect(k8sClient.Create(ctx, structure)).To(Succeed())
			defer func() {
				By("Cleanup the specific resource instance Structure")
				Expect(k8sClient.Delete(ctx, structure)).To(Succeed())
			}()

			By("Reconciling the created resource")
			controllerReconciler := &StructureReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(MatchError("ID Not Specified"))
			Expect(result.IsZero()).To(Equal(true))

			structure.Spec = contractorv1.StructureSpec{ID: 42}
			Expect(k8sClient.Update(ctx, structure)).NotTo(HaveOccurred())

			result, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(Equal(false))
			Expect(result.RequeueAfter).To(Equal(time.Second * 30))

			structure.Spec.State = "planned"
			structure.Spec.BluePrint = "test-structure-base"
			Expect(k8sClient.Update(ctx, structure)).NotTo(HaveOccurred())

			result, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(Equal(true))

		})

	})
})
