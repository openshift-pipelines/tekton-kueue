/*
Copyright 2026.

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

	"github.com/konflux-ci/tekton-kueue/pkg/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tekv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	kapi "knative.dev/pkg/apis"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var _ = Describe("pipelineRunStatusReconciler", func() {
	var (
		reconciler *pipelineRunStatusReconciler
		s          *runtime.Scheme
		nsName     types.NamespacedName
	)

	BeforeEach(func() {
		s = runtime.NewScheme()
		Expect(tekv1.AddToScheme(s)).To(Succeed())

		nsName = types.NamespacedName{
			Name:      "test-plr",
			Namespace: "default",
		}
	})

	Describe("Reconcile", func() {
		It("should return success when PipelineRun is not found", func(ctx context.Context) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				Build()
			reconciler = &pipelineRunStatusReconciler{client: fakeClient}

			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: nsName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("should skip PipelineRun that is not pending", func(ctx context.Context) {
			plr := &tekv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plr",
					Namespace: "default",
				},
				Spec: tekv1.PipelineRunSpec{
					ManagedBy: ptr.To(common.ManagedByMultiKueueLabel),
				},
			}
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithStatusSubresource(&tekv1.PipelineRun{}).
				WithObjects(plr).
				Build()
			reconciler = &pipelineRunStatusReconciler{client: fakeClient}

			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: nsName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			var updated tekv1.PipelineRun
			Expect(fakeClient.Get(ctx, nsName, &updated)).To(Succeed())
			Expect(updated.Status.Conditions).To(BeEmpty())
		})

		It("should skip PipelineRun that already has a Succeeded condition", func(ctx context.Context) {
			plr := &tekv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plr",
					Namespace: "default",
				},
				Spec: tekv1.PipelineRunSpec{
					Status:    tekv1.PipelineRunSpecStatusPending,
					ManagedBy: ptr.To(common.ManagedByMultiKueueLabel),
				},
			}
			plr.Status.Conditions = []kapi.Condition{
				{
					Type:   kapi.ConditionSucceeded,
					Status: corev1.ConditionUnknown,
					Reason: "Running",
				},
			}
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithStatusSubresource(&tekv1.PipelineRun{}).
				WithObjects(plr).
				Build()
			reconciler = &pipelineRunStatusReconciler{client: fakeClient}

			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: nsName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			var updated tekv1.PipelineRun
			Expect(fakeClient.Get(ctx, nsName, &updated)).To(Succeed())
			Expect(updated.Status.GetCondition(kapi.ConditionSucceeded).Reason).To(Equal("Running"))
		})

		It("should skip pending PipelineRun without managedBy", func(ctx context.Context) {
			plr := &tekv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plr",
					Namespace: "default",
				},
				Spec: tekv1.PipelineRunSpec{
					Status: tekv1.PipelineRunSpecStatusPending,
				},
			}
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithStatusSubresource(&tekv1.PipelineRun{}).
				WithObjects(plr).
				Build()
			reconciler = &pipelineRunStatusReconciler{client: fakeClient}

			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: nsName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			var updated tekv1.PipelineRun
			Expect(fakeClient.Get(ctx, nsName, &updated)).To(Succeed())
			Expect(updated.Status.Conditions).To(BeEmpty())
		})

		It("should skip pending PipelineRun with non-MultiKueue managedBy", func(ctx context.Context) {
			plr := &tekv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plr",
					Namespace: "default",
				},
				Spec: tekv1.PipelineRunSpec{
					Status:    tekv1.PipelineRunSpecStatusPending,
					ManagedBy: ptr.To("some-other-controller"),
				},
			}
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithStatusSubresource(&tekv1.PipelineRun{}).
				WithObjects(plr).
				Build()
			reconciler = &pipelineRunStatusReconciler{client: fakeClient}

			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: nsName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			var updated tekv1.PipelineRun
			Expect(fakeClient.Get(ctx, nsName, &updated)).To(Succeed())
			Expect(updated.Status.Conditions).To(BeEmpty())
		})

		It("should set PipelineRunPending condition on MultiKueue-managed PipelineRun with no conditions", func(ctx context.Context) {
			plr := &tekv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plr",
					Namespace: "default",
				},
				Spec: tekv1.PipelineRunSpec{
					Status:    tekv1.PipelineRunSpecStatusPending,
					ManagedBy: ptr.To(common.ManagedByMultiKueueLabel),
				},
			}
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithStatusSubresource(&tekv1.PipelineRun{}).
				WithObjects(plr).
				Build()
			reconciler = &pipelineRunStatusReconciler{client: fakeClient}

			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: nsName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			var updated tekv1.PipelineRun
			Expect(fakeClient.Get(ctx, nsName, &updated)).To(Succeed())

			condition := updated.Status.GetCondition(kapi.ConditionSucceeded)
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(corev1.ConditionUnknown))
			Expect(condition.Reason).To(Equal("PipelineRunPending"))
			Expect(condition.Message).To(ContainSubstring("pending"))
		})

		It("should return error when status update fails", func(ctx context.Context) {
			plr := &tekv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-plr",
					Namespace: "default",
				},
				Spec: tekv1.PipelineRunSpec{
					Status:    tekv1.PipelineRunSpecStatusPending,
					ManagedBy: ptr.To(common.ManagedByMultiKueueLabel),
				},
			}
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithStatusSubresource(&tekv1.PipelineRun{}).
				WithObjects(plr).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourceUpdate: func(_ context.Context, _ client.Client, _ string, _ client.Object, _ ...client.SubResourceUpdateOption) error {
						return fmt.Errorf("conflict")
					},
				}).
				Build()
			reconciler = &pipelineRunStatusReconciler{client: fakeClient}

			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: nsName})
			Expect(err).To(MatchError("conflict"))
		})
	})
})

var _ = Describe("needsPendingStatusCondition", func() {
	It("should return false for non-PipelineRun objects", func() {
		pod := &corev1.Pod{}
		Expect(needsPendingStatusCondition(pod)).To(BeFalse())
	})

	It("should return false when managedBy is nil", func() {
		plr := &tekv1.PipelineRun{
			Spec: tekv1.PipelineRunSpec{
				Status: tekv1.PipelineRunSpecStatusPending,
			},
		}
		Expect(needsPendingStatusCondition(plr)).To(BeFalse())
	})

	It("should return false when managedBy is not MultiKueue", func() {
		plr := &tekv1.PipelineRun{
			Spec: tekv1.PipelineRunSpec{
				Status:    tekv1.PipelineRunSpecStatusPending,
				ManagedBy: ptr.To("some-other-controller"),
			},
		}
		Expect(needsPendingStatusCondition(plr)).To(BeFalse())
	})

	It("should return false when not pending", func() {
		plr := &tekv1.PipelineRun{
			Spec: tekv1.PipelineRunSpec{
				ManagedBy: ptr.To(common.ManagedByMultiKueueLabel),
			},
		}
		Expect(needsPendingStatusCondition(plr)).To(BeFalse())
	})

	It("should return true when MultiKueue-managed and pending", func() {
		plr := &tekv1.PipelineRun{
			Spec: tekv1.PipelineRunSpec{
				Status:    tekv1.PipelineRunSpecStatusPending,
				ManagedBy: ptr.To(common.ManagedByMultiKueueLabel),
			},
		}
		Expect(needsPendingStatusCondition(plr)).To(BeTrue())
	})
})
