package server

import "fmt"

type MemStorage struct {
	m map[string]Metric
}

func (r *MemStorage) String() string {
	s := ""
	for k, v := range r.m {
		s += fmt.Sprintf("key: %s -> value: %v\n", k, v.GetValue())
	}
	return s
}

func NewStorage(vMap map[string]Metric) *MemStorage {
	if vMap == nil {
		return &MemStorage{map[string]Metric{}}
	}
	return &MemStorage{vMap}
}

var MapMetric = NewStorage(nil)
