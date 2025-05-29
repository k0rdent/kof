package target

type Nodes map[string]*Node

type Node struct {
	Pods `json:"pods"`
}

func (n Nodes) FindOrCreate(name string) *Node {
	if node := n.Find(name); node != nil {
		return node
	}
	return n.Create(name)
}

func (n Nodes) Find(name string) *Node {
	if node, ok := n[name]; ok {
		return node
	}
	return nil
}

func (n Nodes) Create(name string) *Node {
	n[name] = &Node{
		Pods: make(Pods),
	}
	return n[name]
}
