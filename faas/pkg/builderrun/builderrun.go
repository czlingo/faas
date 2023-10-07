package builderrun

import (
	"context"

	faas "github.com/czlingo/faas/faas/api/v1alpha1"
	tektoncdv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	envWorkspaceSourceKey   = "WORKSPACE_SOURCES"
	envWorkspaceSourceValue = "/workspace/source"

	workspaceSource = "source"

	paramAppImage   = "APP_IMAGE"
	paramSubPath    = "SOURCE_SUBPATH"
	paramBuildImage = "BUILDER_IMAGE"
	paramRunImage   = "RUN_IMAGE"
)

type Result struct {
	Phase   string
	Message string
	Reason  string
}

type BuilderRun interface {
	Start(ctx context.Context) error
	Result(ctx context.Context) (*Result, error)
}

type builderRun struct {
	client.Client
	Scheme *runtime.Scheme

	building      *faas.Building
	buildTaskName string
}

func New(building *faas.Building, client client.Client, scheme *runtime.Scheme) *builderRun {
	builderRun := &builderRun{
		Client:   client,
		Scheme:   scheme,
		building: building,
	}

	if building.Status.State != nil && building.Status.State.Builder != "" {
		builderRun.buildTaskName = building.Status.State.Builder
	}

	return builderRun
}

func (t *builderRun) Start(ctx context.Context) error {
	strategy := &faas.BuildStrategy{}
	if err := t.Get(ctx, client.ObjectKey{Namespace: t.building.Namespace, Name: string(t.building.Spec.Strategy)}, strategy); err != nil {
		return err
	}

	if err := t.createTask(ctx, strategy); err != nil {
		return err
	}

	return nil
}

func (t *builderRun) createTask(ctx context.Context, strategy *faas.BuildStrategy) error {
	taskSpec, err := t.generateTaskSpec(ctx, strategy)
	if err != nil {
		return err
	}

	task := &tektoncdv1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				faas.LabelName: t.building.Name,
			},
			Namespace:    t.building.Namespace,
			GenerateName: t.building.Name + "-",
		},
		Spec: tektoncdv1.TaskRunSpec{
			TaskSpec: taskSpec,
			Workspaces: []tektoncdv1.WorkspaceBinding{
				{
					Name:     workspaceSource,
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
	}

	params := tektoncdv1.Params{
		{
			Name: paramAppImage,
			Value: tektoncdv1.ParamValue{
				Type:      tektoncdv1.ParamTypeString,
				StringVal: t.building.Spec.Image,
			},
		},
		{
			Name: paramSubPath,
			Value: tektoncdv1.ParamValue{
				Type:      tektoncdv1.ParamTypeString,
				StringVal: t.building.Spec.SubPath,
			},
		},
		{
			Name: paramBuildImage,
			Value: tektoncdv1.ParamValue{
				Type:      tektoncdv1.ParamTypeString,
				StringVal: "czlingo/sample-builder",
			},
		},
		{
			Name: paramRunImage,
			Value: tektoncdv1.ParamValue{
				Type:      tektoncdv1.ParamTypeString,
				StringVal: "czlingo/faas-go-run",
			},
		},
	}
	task.Spec.Params = params

	credentialVolumes, credentialVolumesMount := t.credentialVolumes()
	task.Spec.TaskSpec.Volumes = append(task.Spec.TaskSpec.Volumes, credentialVolumes...)
	for i := range task.Spec.TaskSpec.Steps {
		task.Spec.TaskSpec.Steps[i].VolumeMounts = append(task.Spec.TaskSpec.Steps[i].VolumeMounts, credentialVolumesMount...)
	}

	task.SetOwnerReferences(nil)
	if err := controllerutil.SetControllerReference(t.building, task, t.Scheme); err != nil {
		return err
	}

	if err := t.Create(ctx, task); err != nil {
		return err
	}

	t.buildTaskName = task.ObjectMeta.Name
	t.building.Status.State = &faas.State{
		Phase:   faas.Created,
		Builder: task.ObjectMeta.Name,
	}

	if err := t.Status().Update(ctx, t.building); err != nil {
		return err
	}

	return nil
}

func (t *builderRun) credentialVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := make([]corev1.Volume, 0, 2)
	mount := make([]corev1.VolumeMount, 0, 2)

	if t.building.Spec.SSHCredential != nil {
		volumes = append(volumes, corev1.Volume{
			Name: "ssh",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: t.building.Spec.SSHCredential.Name,
				},
			},
		})
		mount = append(mount, corev1.VolumeMount{
			Name:      "ssh",
			MountPath: "/credential/ssh",
		})
	}

	if t.building.Spec.ImageCredential != nil {
		volumes = append(volumes, corev1.Volume{
			Name: "docker",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: t.building.Spec.ImageCredential.Name,
				},
			},
		})
		mount = append(mount, corev1.VolumeMount{
			Name:      "docker",
			MountPath: "/credential/docker",
		})
	}
	return volumes, mount
}

func (t *builderRun) generateTaskSpec(ctx context.Context, strategy *faas.BuildStrategy) (*tektoncdv1.TaskSpec, error) {
	spec := &tektoncdv1.TaskSpec{}
	for _, v := range strategy.Spec.Parameter {
		param := tektoncdv1.ParamSpec{
			Name:        v.Name,
			Description: v.Description,
		}
		if v.Default != nil {
			param.Default = &tektoncdv1.ParamValue{
				Type:      tektoncdv1.ParamTypeString,
				StringVal: *v.Default,
			}
		}
		if v.Defaults != nil {
			param.Default = &tektoncdv1.ParamValue{
				Type:     tektoncdv1.ParamTypeArray,
				ArrayVal: *v.Defaults,
			}
		}
		spec.Params = append(spec.Params, param)
	}

	for _, v := range strategy.Spec.Steps {
		spec.Steps = append(spec.Steps, tektoncdv1.Step{
			Name:            v.Name,
			Image:           v.Image,
			ImagePullPolicy: v.ImagePullPolicy,
			Env:             append(v.Env, t.generateEnv()...),
			Script:          v.Script,
			VolumeMounts:    v.VolumeMounts,
		})
	}

	for _, v := range strategy.Spec.Volumes {
		spec.Volumes = append(spec.Volumes, corev1.Volume{
			Name:         v.Name,
			VolumeSource: v.VolumeSource,
		})
	}

	if err := t.amendTaskSpecWithSources(spec); err != nil {
		return nil, err
	}

	return spec, nil
}

func (t *builderRun) amendTaskSpecWithSources(taskSpec *tektoncdv1.TaskSpec) error {
	script := `#!/usr/bin/env bash
cd $WORKSPACE_SOURCES

# if [ -r "/credential/ssh" ]; then
# fi

cp -r /credential/ssh ~/.ssh
chmod 700 ~/.ssh
chmod -R 400 ~/.ssh/*

# if [ "$(workspaces.credentials.bound)" == "true" ] ; then
# 	cp -r $(workspaces.credentials.path) "${USER_HOME}"/.ssh
# 	chmod 700 "${USER_HOME}"/.ssh
# 	chmod -R 400 "${USER_HOME}"/.ssh/*
# fi

`

	cmd := "git clone " + t.building.Spec.Repo
	if t.building.Spec.Revision != nil {
		cmd += " -b " + *t.building.Spec.Revision
	}
	cmd += " ."

	script += cmd

	step := tektoncdv1.Step{
		Name:   "git-clone",
		Image:  "bitnami/git",
		Env:    t.generateEnv(),
		Script: script,
	}

	taskSpec.Steps = append([]tektoncdv1.Step{step}, taskSpec.Steps...)
	return nil
}

func (t *builderRun) generateEnv() []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name:  envWorkspaceSourceKey,
			Value: envWorkspaceSourceValue,
		},
	}
	return envs
}

var ReasonPhaseMapping = map[tektoncdv1.TaskRunReason]string{
	tektoncdv1.TaskRunReasonStarted:    faas.Starting,
	tektoncdv1.TaskRunReasonRunning:    faas.Running,
	tektoncdv1.TaskRunReasonSuccessful: faas.Succeeded,
	tektoncdv1.TaskRunReasonFailed:     faas.Failed,
	tektoncdv1.TaskRunReasonCancelled:  faas.Failed,
	tektoncdv1.TaskRunReasonTimedOut:   faas.Failed,
}

func (t *builderRun) Result(ctx context.Context) (*Result, error) {
	buildTaskRun := &tektoncdv1.TaskRun{}
	if err := t.Client.Get(ctx, client.ObjectKey{Namespace: t.building.Namespace, Name: t.buildTaskName}, buildTaskRun); err != nil {
		return nil, err
	}

	if c := buildTaskRun.GetStatusCondition().GetCondition(apis.ConditionSucceeded); c != nil {
		reason := tektoncdv1.TaskRunReason(c.Reason)
		phase := ReasonPhaseMapping[reason]
		return &Result{
			Phase:   phase,
			Reason:  c.Reason,
			Message: c.Message,
		}, nil
	}

	return &Result{
		Phase: faas.Created,
	}, nil
}
