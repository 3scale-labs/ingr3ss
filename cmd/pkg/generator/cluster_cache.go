package generator

import (
	"strings"
	"time"

	"go.uber.org/zap"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
	gocache "github.com/patrickmn/go-cache"
)

const (
	clusterExpiration      = 15 * time.Second
	defaultExpiration      = gocache.NoExpiration
	defaultCleanupInterval = 1 * time.Minute
)

type ClustersCache struct {
	clusters *gocache.Cache
	logger   *zap.SugaredLogger
}

func newClustersCache(logger *zap.SugaredLogger) *ClustersCache {
	goCache := gocache.New(defaultExpiration, defaultCleanupInterval)
	return &ClustersCache{clusters: goCache, logger: logger}
}

func newClustersCacheWithExpAndCleanupIntervals(expiration time.Duration, cleanupInterval time.Duration,
	logger *zap.SugaredLogger) *ClustersCache {
	goCache := gocache.New(expiration, cleanupInterval)
	return &ClustersCache{clusters: goCache, logger: logger}
}

func (cc *ClustersCache) set(cluster *v2.Cluster, ingressName string, ingressNamespace string) {
	key := key(cluster.Name, ingressName, ingressNamespace)
	cc.clusters.Set(key, cluster, defaultExpiration)
}

func (cc *ClustersCache) setExpiration(clusterName string, ingressName string, ingressNamespace string) {
	key := key(clusterName, ingressName, ingressNamespace)
	if cluster, ok := cc.clusters.Get(key); ok {
		cc.clusters.Set(key, cluster, clusterExpiration)
	}
}

func (cc *ClustersCache) list() []envoycache.Resource {
	var res []envoycache.Resource
	cc.logger.Debug("listing clusters")

	for _, cluster := range cc.clusters.Items() {
		cc.logger.Debugf("listing cluster %#v", cluster.Object.(*v2.Cluster))
		res = append(res, cluster.Object.(envoycache.Resource))
	}

	return res
}

// Using only the cluster name is not enough to ensure uniqueness, that's why we
// use also the ingress info.
func key(clusterName, ingressName, ingressNamespace string) string {
	return strings.Join([]string{clusterName, ingressName, ingressNamespace}, ":")
}
