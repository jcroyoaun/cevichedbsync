/*
Copyright 2025.

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	migrationsv1alpha1 "cevichedbsync-operator/api/v1alpha1"
)

var _ = Describe("PostgresSync Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		postgressync := &migrationsv1alpha1.PostgresSync{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind PostgresSync")
			err := k8sClient.Get(ctx, typeNamespacedName, postgressync)
			if err != nil && errors.IsNotFound(err) {
				resource := &migrationsv1alpha1.PostgresSync{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: migrationsv1alpha1.PostgresSyncSpec{
						RepositoryURL: "https://github.com/example/repo.git",
						GitCredentials: migrationsv1alpha1.CredentialReference{
							SecretName: "git-credentials",
						},
						DatabaseCredentials: migrationsv1alpha1.CredentialReference{
							SecretName: "db-credentials",
						},
						StatefulSetRef: migrationsv1alpha1.StatefulSetReference{
							Name: "postgres",
						},
						DatabaseService: migrationsv1alpha1.DatabaseServiceReference{
							Name: "postgres-service",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			} else if err != nil {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			// Try to clean up the resource instance
			resource := &migrationsv1alpha1.PostgresSync{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &PostgresSyncReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			// We don't expect an error because the controller handles missing StatefulSet gracefully
			Expect(err).NotTo(HaveOccurred())

			// The controller should requeue when StatefulSet is missing
			Expect(result.Requeue || result.RequeueAfter > 0).To(BeTrue())

			// After reconciliation, we should still be able to fetch our resource
			fetchedResource := &migrationsv1alpha1.PostgresSync{}
			err = k8sClient.Get(ctx, typeNamespacedName, fetchedResource)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
