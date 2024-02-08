package runtime

import (
	"context"

	faasv1alpha1 "github.com/czlingo/faas/faas/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Result struct{}

type Interface interface {
	Serving(ctx context.Context) error
	Result(ctx context.Context) (*Result, error)
}

type constructor func(client.Client, *runtime.Scheme, *faasv1alpha1.Serving) Interface

type facotry struct {
	d map[faasv1alpha1.Runtime]constructor
}

func newFactory() *facotry {
	return &facotry{
		d: map[faasv1alpha1.Runtime]constructor{},
	}
}

func (f *facotry) Get(runtime faasv1alpha1.Runtime, client client.Client, sheme *runtime.Scheme, svc *faasv1alpha1.Serving) Interface {
	construct := f.d[runtime]
	if construct == nil {
		return nil
	}
	return construct(client, sheme, svc)
}

func (f *facotry) Register(runtime faasv1alpha1.Runtime, construct constructor) {
	f.d[runtime] = construct
}

var ServeFactory *facotry

func init() {
	ServeFactory = newFactory()

	ServeFactory.Register(faasv1alpha1.DefaultRuntime, NewDefaultRuntime)
	ServeFactory.Register(faasv1alpha1.KnativeRuntime, NewDefaultKnativeRuntime)
}

type defaultRuntime struct {
	client.Client
	Scheme *runtime.Scheme
	svc    *faasv1alpha1.Serving
}

func NewDefaultRuntime(client client.Client, scheme *runtime.Scheme, svc *faasv1alpha1.Serving) Interface {
	return &defaultRuntime{
		Client: client,
		Scheme: scheme,
		svc:    svc,
	}
}

func (d *defaultRuntime) Serving(ctx context.Context) error {
	logger := log.FromContext(ctx)

	var defaultReplicas int32 = 1

	labels := map[string]string{
		"fn-svc-pod": d.svc.Namespace + "-" + d.svc.Name,
	}

	servingRun := &appsv1.Deployment{}
	// FIXME: check serving state
	// avoid repeat to create deployment
	err := d.Client.Get(ctx, client.ObjectKey{Namespace: d.svc.Namespace, Name: d.svc.Name}, servingRun)
	if err == nil || !errors.IsNotFound(err) {
		return err
	}

	servingRun = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: d.svc.Namespace,
			Name:      d.svc.Name,
			Labels: map[string]string{
				faasv1alpha1.LabelName: d.svc.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &defaultReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  d.svc.Name,
							Image: *d.svc.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: d.svc.Spec.ImageCredentials.Name,
						},
					},
				},
			},
		},
	}

	d.svc.Status.State = &faasv1alpha1.State{
		Phase: faasv1alpha1.Running,
	}
	if err := d.Status().Update(ctx, d.svc); err != nil {
		logger.Error(err, "Failed to update serving state")
		return err
	}

	servingRun.SetOwnerReferences(nil)
	if err := controllerutil.SetControllerReference(d.svc, servingRun, d.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return err
	}

	if err := d.Create(ctx, servingRun); err != nil {
		logger.Error(err, "Failed to create servingRunner")
		return err
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: d.svc.Namespace,
			Name:      d.svc.Name,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 80},
				},
			},
		},
	}

	service.SetOwnerReferences(nil)
	if err := controllerutil.SetControllerReference(d.svc, service, d.Scheme); err != nil {
		return err
	}

	if err := d.Create(ctx, service); err != nil {
		return err
	}

	return nil
}

func (s *defaultRuntime) Result(ctx context.Context) (*Result, error) {
	return nil, nil
}
