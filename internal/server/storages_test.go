package server

import (
	"testing"

	"github.com/akashipov/MetricCollector/internal/general"
)

func TestMemStorage_String(t *testing.T) {
	type fields struct {
		m map[string]general.Metric
	}
	var m int64 = 1
	var a general.Metric = general.NewCounter(m)
	m = 2
	var b general.Metric = general.NewCounter(m)
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "one",
			fields: fields{
				m: map[string]general.Metric{"a": a},
			},
			want: []string{"key: a -> value: 1\n"},
		},
		{
			name: "several",
			fields: fields{
				m: map[string]general.Metric{
					"a": a,
					"b": b,
				},
			},
			want: []string{"key: a -> value: 1\nkey: b -> value: 2\n", "key: b -> value: 2\nkey: a -> value: 1\n"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MemStorage{
				m: tt.fields.m,
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
