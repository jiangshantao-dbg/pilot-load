package xds

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/howardjohn/pilot-load/adsc"
	"github.com/howardjohn/pilot-load/pkg/simulation/model"
	"google.golang.org/grpc"
	meshconfig "istio.io/api/mesh/v1alpha1"
	istiomodel "istio.io/istio/pilot/pkg/model"
)

type Simulation struct {
	Labels    map[string]string
	Namespace string
	Name      string
	IP        string
	// Defaults to "Kubernetes"
	Cluster string
	PodType model.PodType

	GrpcOpts []grpc.DialOption

	cancel context.CancelFunc
	done   chan struct{}
}

func clone(m map[string]string) map[string]interface{} {
	n := map[string]interface{}{}
	for k, v := range m {
		n[k] = v
	}
	return n
}

func (x *Simulation) Run(ctx model.Context) error {
	c, cancel := context.WithCancel(ctx.Context)
	x.cancel = cancel
	x.done = make(chan struct{})
	cluster := x.Cluster
	if cluster == "" {
		cluster = "Kubernetes"
	}
	meta := clone(ctx.Args.Metadata)
	meta["ISTIO_VERSION"] = "1.20.0-pilot-load"
	meta["CLUSTER_ID"] = cluster
	meta["LABELS"] = x.Labels
	meta["NAMESPACE"] = x.Namespace
	meta["SDS"] = "true"
	meta["PROXY_CONFIG"] = (*istiomodel.NodeMetaProxyConfig)(&meshconfig.ProxyConfig{
		ConfigPath:               "./etc/istio/proxy",
		BinaryPath:               "/usr/local/bin/envoy",
		DrainDuration:            &types.Duration{Seconds: 45},
		ParentShutdownDuration:   &types.Duration{Seconds: 60},
		DiscoveryAddress:         "istiod-v0109.mt-istio-system.svc.cluster.local",
		ProxyAdminPort:           15000,
		ControlPlaneAuthPolicy:   meshconfig.AuthenticationPolicy_MUTUAL_TLS,
		Concurrency:              &types.Int32Value{Value: 2},
		StatNameLength:           189,
		Tracing:                  &meshconfig.Tracing{Tracer: &meshconfig.Tracing_Zipkin_{Zipkin: &meshconfig.Tracing_Zipkin{Address: "zipkin.mt-istio-system:9411"}}},
		ProxyMetadata:            map[string]string{"ISTIO_META_DNS_CAPTURE": "false", "ISTIO_META_PROXY_XDS_VIA_AGENT": "false"},
		StatusPort:               15020,
		TerminationDrainDuration: &types.Duration{Seconds: 5},
		MeshId:                   "cluster.local",
		ServiceCluster:           "istio-gateway-biz-common",
	})
	meta["ROUTER_MODE"] = "standard"
	meta["SERVICE_ACCOUNT"] = "istio-gateway-service-account"
	meta["WORKLOAD_NAME"] = "istio-gateway-biz-common"
	go func() {
		adsc.Connect(ctx.Args.PilotAddress, &adsc.Config{
			Namespace: x.Namespace,
			Workload:  x.Name + "-" + x.IP + "-" + strconv.FormatInt(time.Now().UnixNano(), 10),
			Meta:      meta,
			NodeType:  string(x.PodType),
			IP:        x.IP,
			Context:   c,
			GrpcOpts:  x.GrpcOpts,
		})
		close(x.done)
	}()
	return nil
}

func (x *Simulation) Cleanup(ctx model.Context) error {
	if x == nil {
		return nil
	}
	if x.cancel != nil {
		x.cancel()
	}
	if x.done != nil {
		<-x.done
	}
	return nil
}

var _ model.Simulation = &Simulation{}
