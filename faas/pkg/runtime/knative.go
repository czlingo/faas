package runtime

import (
	"context"

	faasv1alpha1 "github.com/czlingo/faas/faas/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type knativeRuntime struct {
	client.Client
	Scheme *runtime.Scheme
	svc    *faasv1alpha1.Serving
}

func NewDefaultKnativeRuntime(client client.Client, scheme *runtime.Scheme, svc *faasv1alpha1.Serving) Interface {
	return &knativeRuntime{
		Client: client,
		Scheme: scheme,
		svc:    svc,
	}
}

func (k *knativeRuntime) Serving(ctx context.Context) error {
	logger := log.FromContext(ctx)

	ks := &kservingv1.Service{}
	err := k.Get(ctx, client.ObjectKey{Namespace: k.svc.Namespace, Name: k.svc.Name}, ks)
	if err == nil || !errors.IsNotFound(err) {
		return err
	}

	ks = &kservingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: k.svc.Namespace,
			Name:      k.svc.Name,
		},
		Spec: kservingv1.ServiceSpec{
			ConfigurationSpec: kservingv1.ConfigurationSpec{
				Template: kservingv1.RevisionTemplateSpec{
					Spec: kservingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: *k.svc.Spec.Image,
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	k.svc.Status.State = &faasv1alpha1.State{
		Phase: faasv1alpha1.Running,
	}
	if err := k.Status().Update(ctx, k.svc); err != nil {
		logger.Error(err, "Failed to update serving state")
		return err
	}

	ks.SetOwnerReferences(nil)
	if err := controllerutil.SetControllerReference(k.svc, ks, k.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return err
	}

	if err := k.Create(ctx, ks); err != nil {
		logger.Error(err, "Failed to create servingRunner")
		return err
	}
	return nil
}

func (k *knativeRuntime) Result(ctx context.Context) (*Result, error) {
	ks := &kservingv1.Service{}
	if err := k.Get(ctx, client.ObjectKeyFromObject(k.svc), ks); err != nil {
		return nil, err
	}
	ks.IsReady()

	return &Result{}, nil
}
