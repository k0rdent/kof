package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/kube"
)

type IstioRemoteSecretsResponse struct {
	Secrets []Secret `json:"secrets"`
}

type Secret struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	SyncStatus  string `json:"syncStatus"`
	ClusterName string `json:"clusterName"`
}

func IstioRemoteSecretsHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()

	istioConfig, err := kube.NewCLIClient(k8s.LocalKubeClient.Config)
	if err != nil {
		res.Logger.Error(err, "failed to create istio client")
		http.Error(res.Writer, "Something went wrong", http.StatusInternalServerError)
		return
	}

	istioRes, err := istioConfig.AllDiscoveryDo(ctx, constants.IstioSystemNamespace, "debug/clusterz")
	if err != nil {
		res.Logger.Error(err, "failed to get istio clusterz")
		http.Error(res.Writer, "Something went wrong", http.StatusInternalServerError)
		return
	}

	result, err := ParseIstioSecretsStatus(res.Logger, istioRes)
	if err != nil {
		res.Logger.Error(err, "failed to parse istio secrets status")
		http.Error(res.Writer, "Something went wrong", http.StatusInternalServerError)
		return
	}

	res.SendObj(&IstioRemoteSecretsResponse{
		Secrets: result,
	}, http.StatusOK)
}

func ParseIstioSecretsStatus(log *logr.Logger, istioRes map[string][]byte) ([]Secret, error) {
	var result []Secret
	for _, bytes := range istioRes {
		var parsed []cluster.DebugInfo
		if err := json.Unmarshal(bytes, &parsed); err != nil {
			return result, fmt.Errorf("failed to parse istio cluster statuses: %v", err)
		}
		for _, info := range parsed {
			if utils.IsEmptyString(info.SecretName) {
				continue
			}

			nameParts := strings.SplitN(info.SecretName, "/", 2)
			if len(nameParts) != 2 {
				log.Error(fmt.Errorf("unexpected secret name format: %s", info.SecretName), "failed to parse secret name")
				continue
			}

			result = append(result, Secret{
				Namespace:   nameParts[0],
				Name:        nameParts[1],
				SyncStatus:  info.SyncStatus,
				ClusterName: info.ID.String(),
			})
		}
	}
	return result, nil
}
