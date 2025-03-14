package controller

import "gopkg.in/yaml.v3"

type ClusterDeploymentConfig struct {
	Region      string
	Location    string
	IdentityRef struct {
		Region string
	} `yaml:"identityRef"`
	VSphere struct {
		Datacenter string
	}
}

func ReadClusterDeploymentConfig(configYaml []byte) (*ClusterDeploymentConfig, error) {
	config := &ClusterDeploymentConfig{}
	err := yaml.Unmarshal(configYaml, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
