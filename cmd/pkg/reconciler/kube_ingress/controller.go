package kube_ingress

import "time"

const (
	gatewayLabelKey   = "app"
	gatewayLabelValue = "3scale-ingr3ss-gateway"

	globalResyncPeriod = 30 * time.Second
	controllerName     = "Ingr3ssController"
	nodeID             = "3scale-ingr3ss-gateway"
	gatewayPort        = 19001
	managementPort     = 18000
)

// TODO
