package target

import (
	v1 "github.com/prometheus/prometheus/web/api/v1"
)

type Pods map[string]*v1.Response

func (p Pods) Add(name string, response *v1.Response) {
	p[name] = response
}
