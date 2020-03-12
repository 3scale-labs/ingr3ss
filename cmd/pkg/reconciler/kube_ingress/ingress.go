package kube_ingress

import (
	"context"
	"fmt"
	"github.com/3scale/ingr3ss/cmd/pkg/generator"
	"github.com/3scale/ingr3ss/cmd/pkg/kubernetes"
	"github.com/3scale/kourier/pkg/envoy"
	"go.uber.org/zap"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/networking/v1beta1"
	"k8s.io/kubernetes/staging/src/k8s.io/client-go/tools/cache"
	"reflect"
)

type Reconciler struct {
	IngressLister     v1beta1.IngressLister
	EnvoyXDSServer    *envoy.XdsServer
	kubeClient        kubeclient.Interface
	CurrentCaches     *generator.Caches
	statusManager     *StatusProber
	ingressTranslator *generator.IngressTranslator
	ExtAuthz          bool
	logger            *zap.SugaredLogger
}

func (reconciler *Reconciler) Reconcile(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	reconciler.logger.Infof("Got reconcile request for %s namespace: %s", name, namespace)

	original, err := reconciler.IngressLister.Ingresses(namespace).Get(name)
	if apierrors.IsNotFound(err) {
		return reconciler.deleteIngress(namespace, name)
	} else if err != nil {
		return err
	}

	ingress := original.DeepCopy()

	if err := reconciler.updateIngress(ingress); err != nil {
		return fmt.Errorf("failed to update ingress: %w", err)
	}

	return reconciler.updateStatus(original, ingress)
}

func (reconciler *Reconciler) deleteIngress(namespace, name string) error {
	reconciler.logger.Infof("Deleting Ingress %s namespace: %s", name, namespace)
	ingress := reconciler.CurrentCaches.GetIngress(name, namespace)

	// We need to check for ingress not being nil, because we can receive an event from an already
	// removed ingress, like for example, when the endpoints object for that ingress is updated/removed.
	if ingress != nil {
		reconciler.statusManager.CancelIngress(ingress)
	}

	err := reconciler.CurrentCaches.DeleteIngressInfo(name, namespace, reconciler.kubeClient)
	if err != nil {
		return err
	}

	snapshot, err := reconciler.CurrentCaches.ToEnvoySnapshot()
	if err != nil {
		return err
	}

	return reconciler.EnvoyXDSServer.SetSnapshot(&snapshot, nodeID)
}

func (reconciler *Reconciler) updateIngress(ingress *networkingv1beta1.Ingress) error {
	reconciler.logger.Infof("Updating Ingress %s namespace: %s", ingress.Name, ingress.Namespace)

	err := generator.UpdateInfoForIngress(
		reconciler.CurrentCaches, ingress, reconciler.kubeClient, reconciler.ingressTranslator, reconciler.logger, reconciler.ExtAuthz,
	)
	if err != nil {
		return err
	}

	snapshot, err := reconciler.CurrentCaches.ToEnvoySnapshot()
	if err != nil {
		return err
	}

	err = reconciler.EnvoyXDSServer.SetSnapshot(&snapshot, nodeID)
	if err != nil {
		return err
	}

	ready, err := reconciler.statusManager.IsReady(ingress)
	if err != nil {
		return err
	}

	if ready {
		kubernetes.MarkIngressReady(ingress)
	}
	return nil
}

func (reconciler *Reconciler) updateStatus(existing *networkingv1beta1.Ingress, desired *networkingv1beta1.Ingress) error {
	// If there's nothing to update, just return.
	if reflect.DeepEqual(existing.Status, desired.Status) {
		return nil
	}

	existing = existing.DeepCopy()
	existing.Status = desired.Status

	// TODO: update status
	//_, err := reconciler.knativeClient.NetworkingV1alpha1().Ingresses(existing.Namespace).UpdateStatus(existing)
	return nil
}
