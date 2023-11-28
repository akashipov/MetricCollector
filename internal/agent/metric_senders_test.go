package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/akashipov/MetricCollector/internal/general"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

func GetHandler(s *string, t *testing.T) func(w http.ResponseWriter, request *http.Request) {
	return func(w http.ResponseWriter, request *http.Request) {
		var buf bytes.Buffer
		buf.ReadFrom(request.Body)
		var m []general.Metrics
		err := json.Unmarshal(buf.Bytes(), &m)
		if err != nil {
			panic(err)
		}
		var v interface{}
		for _, value := range m {
			if value.Delta != nil {
				v = *value.Delta
			} else {
				v = *value.Value
			}
			(*s) += fmt.Sprintf("id: '%v', type: '%v', value: '%v'||", value.ID, value.MType, v)
		}
		assert.Equal(t, "application/json", request.Header["Content-Type"][0])
		hash := request.Header.Get("HashSHA256")
		if hash != "" {
			(*s) += fmt.Sprintf("Hashed")
		}
	}
}

func TestMetricSender_PollInterval(t *testing.T) {
	type fields struct {
		ListMetrics *[]string
	}
	if AgentKey == nil {
		ParseArgsClient()
	}
	tests := []struct {
		name          string
		fields        fields
		keyForHashing string
		isHashed      bool
	}{
		{
			name: "1",
			fields: fields{
				ListMetrics: &ListMetrics,
			},
			keyForHashing: "",
			isHashed:      false,
		},
		{
			name: "1",
			fields: fields{
				ListMetrics: &ListMetrics,
			},
			keyForHashing: "blabla",
			isHashed:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AgentKey = &tt.keyForHashing
			s := ""
			fmt.Println("Mocking server...")
			server := httptest.NewServer(
				http.HandlerFunc(
					GetHandler(&s, t),
				),
			)
			defer server.Close()
			PullIntervalTime := 1
			ReportIntervalTime := 2
			var m sync.Mutex
			done := make(chan bool)
			goPull := make(chan struct{})
			var wg sync.WaitGroup
			r := MetricSender{
				URL:                server.URL,
				ListMetrics:        tt.fields.ListMetrics,
				Client:             resty.New(),
				ReportIntervalTime: &PullIntervalTime,
				PollIntervalTime:   &ReportIntervalTime,
				Done:               done,
				M:                  &m,
				GoPull:             goPull,
				WG:                 &wg,
			}
			wg.Add(1)
			go r.Run()
			time.Sleep(time.Duration(*ReportInterval)*time.Second + time.Millisecond*500)
			close(done)
			wg.Wait()
			for _, v := range ListMetrics {
				assert.Contains(t, s, fmt.Sprintf("id: '%s', type: 'gauge', value:", v))
			}
			assert.Contains(t, s, "id: 'RandomValue', type: 'gauge', value:")
			assert.Contains(t, s, "id: 'PollCount', type: 'counter', value: '2'")
			if tt.isHashed {
				assert.Equal(t, s[len(s)-len("||Hashed"):], "||Hashed")
			}
		})
	}
}

func TestMetricSender_ReportInterval(t *testing.T) {
	type fields struct {
		ListMetrics *[]string
	}
	if AgentKey == nil {
		ParseArgsClient()
	}
	type args struct {
		a             *map[string]interface{}
		countOfUpdate int64
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
				a: &map[string]interface{}{
					"Alloc":   1245.0,
					"Sys":     544.0,
					"Lookups": 10.0,
				},
				countOfUpdate: int64(5),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := ""
			fmt.Println("Mocking server...")
			server := httptest.NewServer(
				http.HandlerFunc(
					GetHandler(&s, t),
				),
			)
			defer server.Close()
			ReportIntervalTime := 1
			PollIntervalTime := 1
			done := make(chan bool)
			goPull := make(chan struct{})
			var m sync.Mutex
			r := MetricSender{
				URL:                server.URL,
				ListMetrics:        tt.fields.ListMetrics,
				Client:             resty.New(),
				ReportIntervalTime: &ReportIntervalTime,
				PollIntervalTime:   &PollIntervalTime,
				Done:               done,
				M:                  &m,
				GoPull:             goPull,
			}
			var count atomic.Int64
			count.Swap(tt.args.countOfUpdate)
			go r.ReportInterval(tt.args.a, &count)
			time.Sleep(time.Duration(*r.ReportIntervalTime)*time.Second + time.Millisecond*50)
			close(done)
			assert.Contains(t, s, "id: 'Alloc', type: 'gauge', value: '1245'")
			assert.Contains(t, s, "id: 'Sys', type: 'gauge', value: '544'")
			assert.Contains(t, s, "id: 'PollCount', type: 'counter', value: '5'")
			assert.Contains(t, s, "id: 'RandomValue', type: 'gauge', value:")
		})
	}
}
