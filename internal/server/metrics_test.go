package server

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestCounter_GetValue(t *testing.T) {
	type fields struct {
		Value int64
	}
	tests := []struct {
		name   string
		fields fields
		want   interface{}
	}{
		{
			name: "1",
			fields: fields{
				Value: 5,
			},
			want: int64(5),
		},
		{
			name: "2",
			fields: fields{
				Value: 5.0,
			},
			want: int64(5),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Counter{
				Value: tt.fields.Value,
			}
			if got := r.GetValue(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCounter_Update(t *testing.T) {
	type fields struct {
		Value int64
	}
	type args struct {
		newValue interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
	}{
		{
			name: "1",
			fields: fields{
				Value: 5,
			},
			args: args{
				newValue: int64(15),
			},
			want: int64(20),
		},
		{
			name: "1",
			fields: fields{
				Value: 5,
			},
			args: args{
				newValue: int64(0),
			},
			want: int64(5),
		},
		{
			name: "1",
			fields: fields{
				Value: 5,
			},
			args: args{
				newValue: int64(-3),
			},
			want: int64(2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Counter{
				Value: tt.fields.Value,
			}
			r.Update(tt.args.newValue)
			assert.Equal(t, tt.want, r.GetValue())
		})
	}
}

func TestGauge_GetValue(t *testing.T) {
	type fields struct {
		Value float64
	}
	tests := []struct {
		name   string
		fields fields
		want   interface{}
	}{
		{
			name: "1",
			fields: fields{
				Value: 5.5,
			},
			want: 5.5,
		},
		{
			name: "2",
			fields: fields{
				Value: 5.0,
			},
			want: float64(5),
		},
		{
			name: "3",
			fields: fields{
				Value: 10,
			},
			want: float64(10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Gauge{
				Value: tt.fields.Value,
			}
			if got := r.GetValue(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGauge_Update(t *testing.T) {
	type fields struct {
		Value float64
	}
	type args struct {
		newValue interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   float64
	}{
		{
			name: "1",
			fields: fields{
				Value: 5,
			},
			args: args{
				newValue: float64(15),
			},
			want: float64(15),
		},
		{
			name: "1",
			fields: fields{
				Value: 5,
			},
			args: args{
				newValue: float64(0),
			},
			want: float64(0),
		},
		{
			name: "1",
			fields: fields{
				Value: 5,
			},
			args: args{
				newValue: float64(-3),
			},
			want: float64(-3),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Gauge{
				Value: tt.fields.Value,
			}
			r.Update(tt.args.newValue)
			assert.Equal(t, tt.want, r.GetValue())
		})
	}
}

func TestNewCounter(t *testing.T) {
	type args struct {
		v int64
	}
	tests := []struct {
		name string
		args args
		want *Counter
	}{
		{
			name: "value_passed",
			args: args{
				int64(5),
			},
			want: &Counter{5},
		},
		{
			name: "nil_passed",
			args: args{
				int64(0),
			},
			want: &Counter{0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCounter(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCounter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewGauge(t *testing.T) {
	type args struct {
		v float64
	}
	tests := []struct {
		name string
		args args
		want *Gauge
	}{
		{
			name: "value_passed",
			args: args{
				float64(5),
			},
			want: &Gauge{5.0},
		},
		{
			name: "nil_passed",
			args: args{
				float64(0),
			},
			want: &Gauge{0.0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewGauge(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGauge() = %v, want %v", got, tt.want)
			}
		})
	}
}
