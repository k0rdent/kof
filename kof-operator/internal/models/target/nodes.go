package target

import "slices"

type Nodes []*Node

type Node struct {
	Name string `json:"name"`
	Pods `json:"pods"`
}

func (n *Nodes) FindOrCreate(name string) *Node {
	node := n.Find(name)

	if node == nil {
		node = n.Create(name)
	}
	return node
}

func (n *Nodes) Find(name string) *Node {
	i := slices.IndexFunc(*n, func(c *Node) bool {
		return c.Name == name
	})

	if i >= 0 {
		return (*n)[i]
	}
	return nil
}

func (n *Nodes) Create(name string) *Node {
	*n = append(*n, &Node{
		Name: name,
		Pods: []*Pod{},
	})
	return (*n)[len(*n)-1]
}
