package general

import (
	"fmt"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

func (m *Metrics) String() string {
	switch m.MType {
	case "gauge":
		return fmt.Sprintf("ID: %s, value: %f\n", m.ID, *m.Value)
	case "counter":
		return fmt.Sprintf("ID: %s, delta: %d\n", m.ID, *m.Delta)
	}
	return ""
}

type Metric interface {
	Update(interface{}) bool
	GetValue() interface{}
}

type Gauge struct {
	Value float64
}

func (r *Gauge) Update(newValue interface{}) bool {
	v, ok := newValue.(float64)
	if ok {
		r.Value = v
	}
	return ok
}

func (r *Gauge) GetValue() interface{} {
	return r.Value
}

func NewGauge(v float64) *Gauge {
	return &Gauge{Value: v}
}

type Counter struct {
	Value int64
}

func (r *Counter) GetValue() interface{} {
	return r.Value
}

func (r *Counter) Update(newValue interface{}) bool {
	v, ok := newValue.(int64)
	if ok {
		r.Value = r.Value + v
	}
	return ok
}

func NewCounter(v int64) *Counter {
	return &Counter{Value: v}
}
