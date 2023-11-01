package server

import (
	"testing"

	"github.com/akashipov/MetricCollector/internal/agent"
	"github.com/akashipov/MetricCollector/internal/general"
)

func TestMemStorage_String(t *testing.T) {
	type fields struct {
		m []*general.Metrics
	}
	var m int64 = 1
	var a general.Metrics = general.Metrics{ID: "a", MType: agent.COUNTER, Delta: &m}
	var newM int64 = 2
	var b general.Metrics = general.Metrics{ID: "b", MType: agent.COUNTER, Delta: &newM}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "one",
			fields: fields{
				m: []*general.Metrics{&a},
			},
			want: []string{"key: a -> value: 1\n"},
		},
		{
			name: "several",
			fields: fields{
				m: []*general.Metrics{
					&a, &b,
				},
			},
			want: []string{"key: a -> value: 1\nkey: b -> value: 2\n", "key: b -> value: 2\nkey: a -> value: 1\n"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MemStorage{
				MetricList: tt.fields.m,
			}
			flag := false
			got := r.String()
			for _, v := range tt.want {
				if got == v {
					flag = true
					break
				}
			}
			if !flag {
				t.Errorf("String() = %v, want any of %v", got, tt.want)
			}
		})
	}
}
