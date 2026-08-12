package controller

import (
	"context"

	"github.com/konflux-ci/tekton-kueue/pkg/common"
	tekv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kapi "knative.dev/pkg/apis"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// +kubebuilder:rbac:groups="tekton.dev",resources=pipelineruns/status,verbs=update;patch

// pipelineRunStatusReconciler watches PipelineRuns and sets a Succeeded=Unknown
// condition with reason "PipelineRunPending" on runs that are suspended
// (spec.status=PipelineRunPending) but have no status.conditions yet. Tekton's
// controller skips PipelineRuns with spec.managedBy set, so without this
// reconciler the status stays blank in CLI and UI tools until MultiKueue admits
// the workload and dispatches it to a spoke cluster.
type pipelineRunStatusReconciler struct {
	client client.Client
}

func (r *pipelineRunStatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var plr tekv1.PipelineRun
	if err := r.client.Get(ctx, req.NamespacedName, &plr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if plr.Spec.ManagedBy == nil || *plr.Spec.ManagedBy != common.ManagedByMultiKueueLabel {
		return ctrl.Result{}, nil
	}

	if plr.Spec.Status != tekv1.PipelineRunSpecStatusPending {
		return ctrl.Result{}, nil
	}

	if plr.Status.GetCondition(kapi.ConditionSucceeded) != nil {
		return ctrl.Result{}, nil
	}

	plr.Status.SetCondition(&kapi.Condition{
		Type:               kapi.ConditionSucceeded,
		Status:             corev1.ConditionUnknown,
		Reason:             "PipelineRunPending",
		Message:            "PipelineRun is pending, waiting for MultiKueue admission",
		LastTransitionTime: kapi.VolatileTime{Inner: metav1.Now()},
	})

	log.V(2).Info("Setting PipelineRunPending status condition")
	return ctrl.Result{}, r.client.Status().Update(ctx, &plr)
}

// needsPendingStatusCondition returns true for PipelineRuns that are
// MultiKueue-managed, pending, and likely missing a status condition.
func needsPendingStatusCondition(o client.Object) bool {
	plr, ok := o.(*tekv1.PipelineRun)
	if !ok {
		return false
	}
	return plr.Spec.ManagedBy != nil &&
		*plr.Spec.ManagedBy == common.ManagedByMultiKueueLabel &&
		plr.Spec.Status == tekv1.PipelineRunSpecStatusPending
}

// SetupPendingStatusWithManager registers the pending-status reconciler with the manager.
func SetupPendingStatusWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("PipelineRunStatus").
		For(&tekv1.PipelineRun{}, builder.WithPredicates(
			predicate.NewPredicateFuncs(needsPendingStatusCondition),
		)).
		Complete(&pipelineRunStatusReconciler{client: mgr.GetClient()})
}
