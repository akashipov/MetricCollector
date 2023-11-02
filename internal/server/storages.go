package server

import (
	"fmt"

	"github.com/akashipov/MetricCollector/internal/general"
)

type MemStorage struct {
	MetricList map[string]*general.Metrics `json:"metrics"`
}

func (r *MemStorage) Get(metricName string) *general.Metrics {
	val, ok := r.MetricList[metricName]
	if ok {
		return val
	}
	return nil
}

func (r *MemStorage) String() string {
	s := ""
	for _, v := range r.MetricList {
		if v.Delta != nil {
			s += fmt.Sprintf("key: %s -> value: %d\n", v.ID, *v.Delta)
		} else {
			s += fmt.Sprintf("key: %s -> value: %f\n", v.ID, *v.Value)
		}
	}
	return s
}

func NewStorage(vMap map[string]*general.Metrics) *MemStorage {
	if vMap == nil {
		return &MemStorage{make(map[string]*general.Metrics, 0)}
	}
	return &MemStorage{vMap}
}

var MapMetric = NewStorage(nil)
