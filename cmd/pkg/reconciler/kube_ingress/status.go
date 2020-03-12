package kube_ingress

import (
	"k8s.io/api/networking/v1beta1"
)

// TODO
type StatusProber struct {

}

// TODO
func (m *StatusProber) CancelIngress(ingress *v1beta1.Ingress) {

}

// TODO
func (m *StatusProber) IsReady(ingress *v1beta1.Ingress) (bool, error) {
	return true, nil
}
