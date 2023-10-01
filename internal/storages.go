package internal

import "fmt"

type MemStorage struct {
	m map[string]*Metric
	fmt.Stringer
}

func (r *MemStorage) String() string {
	s := ""
	for k, v := range r.m {
		s += fmt.Sprintf("key: %s -> value: %v\n", k, (*v).GetValue())
	}
	return s
}

var MapMetric = MemStorage{}
