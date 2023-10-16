package server

type Metric interface {
	Update(interface{})
	GetValue() interface{}
}

type Gauge struct {
	Value float64
}

func (r *Gauge) Update(newValue interface{}) {
	r.Value = newValue.(float64)
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

func (r *Counter) Update(newValue interface{}) {
	r.Value = r.Value + newValue.(int64)
}

func NewCounter(v int64) *Counter {
	return &Counter{Value: v}
}