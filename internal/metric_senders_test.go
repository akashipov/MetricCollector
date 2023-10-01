package internal

import (
	"fmt"
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
	fmt.Println("mocking server")
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
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, request *http.Request) {
						s += fmt.Sprintf("%v", request.URL)
						assert.Equal(t, []string{"text/plain"}, request.Header["Content-Type"])
					},
				),
			)
			defer server.Close()
			r := MetricSender{
				URL:         server.URL,
				ListMetrics: tt.fields.ListMetrics,
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := MetricSender{

				ListMetrics: tt.fields.ListMetrics,
			}
			r.ReportInterval(tt.args.a, tt.args.countOfUpdate)
		})
	}
}
