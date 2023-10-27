package server

import (
	"fmt"

	"github.com/akashipov/MetricCollector/internal/general"
)

type MemStorage struct {
	m map[string]general.Metric
}

func (r *MemStorage) String() string {
	s := ""
	for k, v := range r.m {
		s += fmt.Sprintf("key: %s -> value: %v\n", k, v.GetValue())
	}
	return s
}

func NewStorage(vMap map[string]general.Metric) *MemStorage {
	if vMap == nil {
		return &MemStorage{map[string]general.Metric{}}
	}
	return &MemStorage{vMap}
}

var MapMetric = NewStorage(nil)
