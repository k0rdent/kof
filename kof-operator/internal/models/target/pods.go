package target

import (
	v1 "github.com/prometheus/prometheus/web/api/v1"
)

type Pods []*Pod

type Pod struct {
	Name     string       `json:"name"`
	Response *v1.Response `json:"response"`
}

func (p *Pods) Add(name string, response *v1.Response) {
	*p = append(*p, &Pod{Name: name,
		Response: response})
}
