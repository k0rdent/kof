package controller

import "gopkg.in/yaml.v3"

type IdentityRef struct {
	Region string `yaml:"region"`
}

type VSphere struct {
	Datacenter string `yaml:"datacenter"`
}

type ClusterDeploymentConfig struct {
	ClusterAnnotations map[string]string `yaml:"clusterAnnotations"`
	Region             string            `yaml:"region"`
	Location           string            `yaml:"location"`
	IdentityRef        IdentityRef       `yaml:"identityRef"`
	VSphere            VSphere           `yaml:"vsphere"`
}

func ReadClusterDeploymentConfig(configYaml []byte) (*ClusterDeploymentConfig, error) {
	config := &ClusterDeploymentConfig{}
	err := yaml.Unmarshal(configYaml, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
