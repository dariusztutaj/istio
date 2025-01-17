package test

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const restartAnnotationName = "istio-operator.kyma-project.io/restartedAt"

func (s *scenario) aRestartHappens(sidecarImage string) error {
	warnings, err := sidecars.ProxyReset(context.TODO(),
		s.Client,
		pods.SidecarImage{Repository: "istio/proxyv2", Tag: sidecarImage},
		s.cniEnabled,
		&s.logger)
	s.restartWarnings = warnings
	return err
}

func (s *scenario) allRequiredResourcesAreDeleted() error {
	for _, v := range s.ToBeDeletedObjects {
		obj := v
		err := s.Client.Get(context.TODO(), types.NamespacedName{Name: v.GetName(), Namespace: v.GetNamespace()}, obj)
		if err == nil {
			return fmt.Errorf("the pod %s/%s should have been deleted, but was not deleted", v.GetNamespace(), v.GetName())
		}

		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (s *scenario) allRequiredResourcesAreRestarted() error {
	for _, v := range s.ToBeRestartedObjects {
		obj := v
		err := s.Client.Get(context.TODO(), types.NamespacedName{Name: v.GetName(), Namespace: v.GetNamespace()}, obj)
		if err != nil {
			return err
		}
		switch obj.GetObjectKind().GroupVersionKind().Kind {
		case "DaemonSet":
			ds := obj.(*appsv1.DaemonSet)
			if _, ok := ds.Spec.Template.Annotations[restartAnnotationName]; !ok {
				return fmt.Errorf("the annotation %s wasn't applied for %s %s/%s", restartAnnotationName, ds.GetObjectKind().GroupVersionKind().Kind, ds.GetNamespace(), ds.GetName())
			}

		case "Deployment":
			dep := obj.(*appsv1.Deployment)
			if _, ok := dep.Spec.Template.Annotations[restartAnnotationName]; !ok {
				return fmt.Errorf("the annotation %s wasn't applied for %s %s/%s", restartAnnotationName, dep.GetObjectKind().GroupVersionKind().Kind, dep.GetNamespace(), dep.GetName())
			}

		case "ReplicaSet":
			rs := obj.(*appsv1.ReplicaSet)
			if _, ok := rs.Spec.Template.Annotations[restartAnnotationName]; !ok {
				return fmt.Errorf("the annotation %s wasn't applied for %s %s/%s", restartAnnotationName, rs.GetObjectKind().GroupVersionKind().Kind, rs.GetNamespace(), rs.GetName())
			}

		case "StatefulSet":
			ss := obj.(*appsv1.StatefulSet)
			if _, ok := ss.Spec.Template.Annotations[restartAnnotationName]; !ok {
				return fmt.Errorf("the annotation %s wasn't applied for %s %s/%s", restartAnnotationName, ss.GetObjectKind().GroupVersionKind().Kind, ss.GetNamespace(), ss.GetName())
			}

		default:
			return fmt.Errorf("kind %s is not supported for rollout", obj.GetObjectKind().GroupVersionKind().Kind)
		}

	}
	return nil
}

func (s *scenario) onlyRequiredResourcesAreDeleted() error {
	for _, v := range s.NotToBeDeletedObjects {
		obj := v
		err := s.Client.Get(context.TODO(), types.NamespacedName{Name: v.GetName(), Namespace: v.GetNamespace()}, obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *scenario) onlyRequiredresourcesAreRestarted() error {
	for _, v := range s.NotToBeRestartedObjects {
		obj := v
		err := s.Client.Get(context.TODO(), types.NamespacedName{Name: v.GetName(), Namespace: v.GetNamespace()}, obj)
		if err != nil {
			return err
		}

		if _, ok := obj.GetAnnotations()[restartAnnotationName]; ok {
			return fmt.Errorf("the annotation %s was applied for %s %s/%s but shouldn't", restartAnnotationName, obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName())
		}
	}
	return nil
}

func (s *scenario) WithConfig(istioVersion, injection, cni string) error {
	s.istioVersion = istioVersion
	s.cniEnabled = cni == "true"
	if injection == "true" {
		s.injectionNamespaceSelector = SidecarEnabledAndDefault
	}
	return nil
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	var s scenario

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		scen, err := newScenario()
		s = *scen
		return ctx, err
	})

	ctx.Step(`^there is a cluster with Istio "([^"]*)", default injection == "([^"]*)" and CNI enabled == "([^"]*)"$`, s.WithConfig)
	ctx.Step(`^a restart happens with target Istio "([^"]*)"`, s.aRestartHappens)
	ctx.Step(`^all required resources are deleted$`, s.allRequiredResourcesAreDeleted)
	ctx.Step(`^all required resources are restarted$`, s.allRequiredResourcesAreRestarted)
	ctx.Step(`^there are Pods missing sidecar`, s.WithPodsMissingSidecar)
	ctx.Step(`^there are not ready Pods$`, s.WithNotReadyPods)
	ctx.Step(`^there are Pods with Istio "([^"]*)" sidecar$`, s.WithSidecarInVersionXPods)
	ctx.Step(`^no resource that is not supposed to be deleted is deleted$`, s.onlyRequiredResourcesAreDeleted)
	ctx.Step(`^no resource that is not supposed to be restarted is restarted$`, s.onlyRequiredresourcesAreRestarted)
}
