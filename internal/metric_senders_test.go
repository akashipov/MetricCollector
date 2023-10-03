package internal

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestMetricSender_PollInterval(t *testing.T) {
	type fields struct {
		ListMetrics *[]string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "1",
			fields: fields{
				ListMetrics: &ListMetrics,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := ""
			fmt.Println("mocking server")
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, request *http.Request) {
						s += fmt.Sprintf("%v", request.URL)
						assert.Equal(
							t,
							"text/plain; charset=utf-8",
							request.Header["Content-Type"][0],
						)
					},
				),
			)
			defer server.Close()
			r := MetricSender{
				URL:         server.URL,
				ListMetrics: tt.fields.ListMetrics,
				client:      resty.New(),
			}
			r.PollInterval(true)
			for _, v := range ListMetrics {
				assert.Contains(t, s, "gauge/"+v)
			}
			assert.Contains(t, s, "gauge/RandomValue")
			assert.Contains(t, s, "counter/PollCount")
		})
	}
}

func TestMetricSender_ReportInterval(t *testing.T) {
	type fields struct {
		ListMetrics *[]string
	}
	type args struct {
		a             *runtime.MemStats
		countOfUpdate int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "1",
			fields: fields{
				&[]string{
					"Alloc", "Sys",
				},
			},
			args: args{
				a: &runtime.MemStats{
					Alloc:   1245,
					Sys:     544,
					Lookups: 10,
				},
				countOfUpdate: 5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := ""
			fmt.Println("mocking server")
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, request *http.Request) {
						s += fmt.Sprintf("%v", request.URL)
						assert.Equal(t, "text/plain; charset=utf-8", request.Header["Content-Type"][0])
					},
				),
			)
			r := MetricSender{
				URL:         server.URL,
				ListMetrics: tt.fields.ListMetrics,
				client:      resty.New(),
			}
			defer server.Close()
			r.ReportInterval(tt.args.a, tt.args.countOfUpdate)
			assert.Contains(t, s, "/update/gauge/RandomValue")
			assert.Contains(t, s, "/update/counter/PollCount/5")
			assert.Contains(t, s, "/update/gauge/Alloc/1245")
			assert.Contains(t, s, "/update/gauge/Sys/544")
			assert.NotContains(t, s, "/update/gauge/Lookups/10")
		})
	}
}
