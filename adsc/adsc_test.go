package adsc_test

import (
	"bytes"
	"encoding/json"
	"testing"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/jsonpb"
	structpb "github.com/golang/protobuf/ptypes/struct"
	meshconfig "istio.io/api/mesh/v1alpha1"
	istiomodel "istio.io/istio/pilot/pkg/model"
)

func TestMetadata(t *testing.T) {
	meta := make(map[string]interface{})
	meta["ISTIO_VERSION"] = "1.9.1"
	meta["CLUSTER_ID"] = "cluster1"
	meta["LABELS"] = map[string]string{"a": "b"}
	meta["NAMESPACE"] = "namespace"
	meta["SDS"] = "true"
	meta["PROXY_CONFIG"] = istiomodel.NodeMetaProxyConfig{
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
	}
	meta["ROUTER_MODE"] = "standard"
	meta["SERVICE_ACCOUNT"] = "istio-gateway-service-account"
	meta["WORKLOAD_NAME"] = "istio-gateway-biz-common"

	n := &core.Node{
		Id: "",
	}
	js, err := json.Marshal(meta)
	if err != nil {
		panic("invalid metadata " + err.Error())
	}

	metadata := &structpb.Struct{}
	err = jsonpb.UnmarshalString(string(js), metadata)
	if err != nil {
		panic("invalid metadata " + err.Error())
	}

	n.Metadata = metadata

	buf := &bytes.Buffer{}
	if err := (&jsonpb.Marshaler{OrigName: true}).Marshal(buf, n.Metadata); err != nil {
		t.Logf("failed to read node metadata %v: %v", metadata, err)
	}
	nodeMeta := &istiomodel.BootstrapNodeMetadata{}
	if err := json.Unmarshal(buf.Bytes(), nodeMeta); err != nil {
		t.Logf("failed to unmarshal node metadata (%v): %v", buf.String(), err)
	}
	t.Logf("%+v", nodeMeta.NodeMetadata)
}
