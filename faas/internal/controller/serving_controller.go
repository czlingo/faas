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
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	faasv1alpha1 "github.com/czlingo/faas/faas/api/v1alpha1"
	svcruntime "github.com/czlingo/faas/faas/pkg/runtime"
	"github.com/czlingo/faas/faas/pkg/util"
)

// ServingReconciler reconciles a Serving object
type ServingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=faas.czlingo.io,resources=servings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=faas.czlingo.io,resources=servings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=faas.czlingo.io,resources=servings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Serving object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *ServingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithName("controllers").WithName("Serving")
	log.WithValues("Serving", req.NamespacedName)

	svc := &faasv1alpha1.Serving{}
	if err := r.Get(ctx, req.NamespacedName, svc); err != nil {
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	serve := svcruntime.ServeFactory.Get(svc.Spec.Runtime, r.Client, r.Scheme, svc)

	if svc.Status.State != nil {
		return ctrl.Result{}, r.updateStatus(ctx, svc, serve)
	}

	if err := serve.Serving(ctx); err != nil {
		log.Error(err, "Failed to serving function")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.updateStatus(ctx, svc, serve)
}

func (r *ServingReconciler) updateStatus(ctx context.Context, serving *faasv1alpha1.Serving, servingRun svcruntime.Interface) error {
	servingRun.Result(ctx)

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := kservingv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&faasv1alpha1.Serving{}).
		Owns(&kservingv1.Service{},
			builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
				_, ok := object.GetAnnotations()[faasv1alpha1.LabelName]
				return ok
			}))).
		Complete(r)
}
