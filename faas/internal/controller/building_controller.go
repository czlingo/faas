/*
Copyright 2023.

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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	faas "github.com/czlingo/faas/faas/api/v1alpha1"
	"github.com/czlingo/faas/faas/pkg/builderrun"
	tektoncdv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

// BuildingReconciler reconciles a Building object
type BuildingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=faas.czlingo.io,resources=buildings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=faas.czlingo.io,resources=buildings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=faas.czlingo.io,resources=buildings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Building object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *BuildingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("BuildingController").
		WithValues("Function", req.String())
	ctx = log.IntoContext(ctx, logger)

	building := &faas.Building{}
	if err := r.Get(ctx, req.NamespacedName, building); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	builderRun := builderrun.New(building, r.Client, r.Scheme)

	if building.Status.State != nil {
		return ctrl.Result{}, r.updateStatus(ctx, building, builderRun)
	}

	if err := builderRun.Start(ctx); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.updateStatus(ctx, building, builderRun)
}

func (r *BuildingReconciler) updateStatus(ctx context.Context, building *faas.Building, builderRun builderrun.BuilderRun) error {
	logger := log.FromContext(ctx)

	br, err := builderRun.Result(ctx)
	if err != nil {
		return err
	}

	if building.Status.State == nil {
		building.Status.State = &faas.State{}
	}

	if building.Status.State.Phase != br.Phase ||
		building.Status.State.Message != br.Message {
		building.Status.State.Phase = br.Phase
		building.Status.State.Message = br.Message
		if err := r.Status().Update(ctx, building); err != nil {
			logger.Error(err, "Failed to update building state")
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	tektoncdv1.AddToScheme(mgr.GetScheme())
	// mgr.GetScheme().AddKnownTypes(tektoncdv1.SchemeGroupVersion, &tektoncdv1.TaskRun{}, &tektoncdv1.TaskRunList{})
	return ctrl.NewControllerManagedBy(mgr).
		For(&faas.Building{}).
		Owns(
			&tektoncdv1.TaskRun{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(
					func(object client.Object) bool {
						_, ok := object.GetLabels()[faas.LabelName]
						return ok
					},
				),
			),
		).Complete(r)
}
