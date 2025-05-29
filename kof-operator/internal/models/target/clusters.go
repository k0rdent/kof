package target

type Clusters map[string]*Cluster

type Cluster struct {
	Nodes `json:"nodes"`
}

func (c Clusters) FindOrCreate(name string) *Cluster {
	if cluster := c.Find(name); cluster != nil {
		return cluster
	}
	return c.Create(name)
}

func (c Clusters) Find(name string) *Cluster {
	if cluster, ok := c[name]; ok {
		return cluster
	}
	return nil
}

func (c Clusters) Create(name string) *Cluster {
	c[name] = &Cluster{
		Nodes: make(Nodes),
	}
	return c[name]
}

func (c Clusters) Add(name string, cluster *Cluster) {
	c[name] = cluster
}
