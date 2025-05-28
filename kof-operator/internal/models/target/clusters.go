package target

import (
	"slices"
)

type Clusters []*Cluster

type Cluster struct {
	Name  string `json:"name"`
	Nodes `json:"nodes"`
}

func (c *Clusters) FindOrCreate(name string) *Cluster {
	cluster := c.Find(name)

	if cluster == nil {
		cluster = c.Create(name)
	}
	return cluster
}

func (c *Clusters) Find(name string) *Cluster {
	i := slices.IndexFunc(*c, func(c *Cluster) bool {
		return c.Name == name
	})

	if i >= 0 {
		return (*c)[i]
	}
	return nil
}

func (c *Clusters) Create(name string) *Cluster {
	*c = append(*c, &Cluster{
		Name:  name,
		Nodes: []*Node{},
	})
	return (*c)[len(*c)-1]
}
