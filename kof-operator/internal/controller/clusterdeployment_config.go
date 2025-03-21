package controller

import "gopkg.in/yaml.v3"

type ClusterDeploymentConfig struct {
	ClusterAnnotations map[string]string `yaml:"clusterAnnotations"`
	Region             string            `yaml:"region"`
	Location           string            `yaml:"location"`
	IdentityRef        struct {
		Region string `yaml:"region"`
	} `yaml:"identityRef"`
	VSphere struct {
		Datacenter string `yaml:"datacenter"`
	} `yaml:"vsphere"`
}

func ReadClusterDeploymentConfig(configYaml []byte) (*ClusterDeploymentConfig, error) {
	config := &ClusterDeploymentConfig{}
	err := yaml.Unmarshal(configYaml, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
