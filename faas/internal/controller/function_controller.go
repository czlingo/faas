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
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	faas "github.com/czlingo/faas/faas/api/v1alpha1"
	"github.com/czlingo/faas/faas/pkg/util"
)

// FunctionReconciler reconciles a Function object
type FunctionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=faas.czlingo.io,resources=functions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=faas.czlingo.io,resources=functions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=faas.czlingo.io,resources=functions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Function object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *FunctionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("FunctionController").
		WithValues("Function", req.String())
	ctx = log.IntoContext(ctx, logger)

	fn := &faas.Function{}
	if err := r.Get(ctx, req.NamespacedName, fn); err != nil {
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	if err := r.building(ctx, fn); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.serving(ctx, fn); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *FunctionReconciler) building(ctx context.Context, fn *faas.Function) error {
	logger := log.FromContext(ctx).WithValues("Phase", "building")

	if !r.needCreateBuilding(fn) {
		logger.Info("No need to create building")
		if fn.Spec.Build != nil {
			if err := r.updateFunctionBuildState(ctx, fn); err != nil {
				logger.Error(err, "Failed to update function build state")
				return err
			}
		}
		return nil
	}

	builder := &faas.Building{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: fn.Namespace,
			Name:      fn.Name,
		},
		Spec: faas.BuildingSpec{
			Strategy:        faas.BuildStrategyBuildpacks,
			Image:           *fn.Spec.Image,
			ImageCredential: fn.Spec.ImageCredential,
			SSHCredential:   fn.Spec.SSHCredential,
			BuildingImpl:    *fn.Spec.Build,
		},
	}

	builder.SetOwnerReferences(nil)
	if err := controllerutil.SetControllerReference(fn, builder, r.Scheme); err != nil {
		logger.Error(err, "Failed to SetOwnerReferences for builder")
		return err
	}

	fn.Status.Build = &faas.State{
		Phase: faas.Created,
	}

	if err := r.Status().Update(ctx, fn); err != nil {
		logger.Error(err, "Failed to update function build state")
		return err
	}

	if err := r.Create(ctx, builder); err != nil {
		logger.Error(err, "Failed to create builder")
		return err
	}
	return nil
}

func (r *FunctionReconciler) needCreateBuilding(fn *faas.Function) bool {
	// TODO:
	if fn.Spec.Build != nil {
		return fn.Status.Build == nil
	}
	return false
}

func (r *FunctionReconciler) updateFunctionBuildState(ctx context.Context, fn *faas.Function) error {
	builder := &faas.Building{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(fn), builder); err != nil {
		return err
	}

	if builder.Status.State != nil && fn.Status.Build.Phase != builder.Status.State.Phase {
		log.FromContext(ctx).Info(fmt.Sprintf("current: %+v, to: %+v", fn.Status.Build, builder.Status.State))

		fn.Status.Build = builder.Status.State

		if err := r.Status().Update(ctx, fn); err != nil {
			return err
		}
	}
	return nil
}

func (r *FunctionReconciler) serving(ctx context.Context, fn *faas.Function) error {
	logger := log.FromContext(ctx).WithValues("Phase", "serving")

	if !r.needCreateSeving(fn) {
		logger.Info("No need to create serving")
		return nil
	}

	svc := &faas.Serving{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: fn.Namespace,
			Name:      fn.Name,
		},
		Spec: faas.ServingSpec{
			Image: fn.Spec.Image,
			// Runtime: faas.DefaultRuntime,
			Runtime: faas.KnativeRuntime,
			ImageCredentials: &v1.LocalObjectReference{
				Name: fn.Spec.ImageCredential.Name,
			},
		},
	}

	svc.SetOwnerReferences(nil)
	if err := controllerutil.SetControllerReference(fn, svc, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, svc); err != nil {
		return err
	}

	fn.Status.Serving = &faas.State{
		Phase: faas.Created,
		Image: *svc.Spec.Image,
	}
	return r.Status().Update(ctx, fn)
}

func (r *FunctionReconciler) needCreateSeving(fn *faas.Function) bool {
	if fn.Status.Serving == nil && fn.Spec.Image != nil {
		if fn.Spec.Build != nil && fn.Status.Build != nil &&
			(fn.Status.Build.Phase == faas.Succeeded ||
				fn.Status.Build.Phase == faas.Skipped) {
			return true
		}

		if fn.Spec.Build == nil {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *FunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&faas.Function{}).
		Owns(&faas.Building{}).
		Owns(&faas.Serving{}).
		Complete(r)
}
