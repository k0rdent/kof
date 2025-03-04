package controller

import "gopkg.in/yaml.v3"

type ClusterDeploymentConfig struct {
	ClusterLabels map[string]string `yaml:"clusterLabels"`
}

func ReadClusterDeploymentConfig(configYaml []byte) (*ClusterDeploymentConfig, error) {
	config := &ClusterDeploymentConfig{}
	err := yaml.Unmarshal(configYaml, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
