package internal

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

var StatusOK string = fmt.Sprintf("%v", http.StatusOK)
var StatusBadRequest string = fmt.Sprintf("%v", http.StatusBadRequest)
var StatusNotFound string = fmt.Sprintf("%v", http.StatusNotFound)

type CustomResponseWriter struct {
	header http.Header
}

func (r *CustomResponseWriter) Write(bytes []byte) (int, error) {
	if r.header == nil {
		r.header = make(map[string][]string)
	}
	record := string(bytes)
	r.header["Record"] = []string{record}
	return len(record), nil
}

func (r *CustomResponseWriter) WriteHeader(statusCode int) {
	if r.header == nil {
		r.header = make(map[string][]string)
	}
	statusCodeStr := fmt.Sprintf("%v", statusCode)
	r.header["Status-Code"] = []string{statusCodeStr}
}

func (r *CustomResponseWriter) Header() http.Header {
	return r.header
}

func TestSaveMetric(t *testing.T) {
	type args struct {
		w          http.ResponseWriter
		metric     Metric
		metricName string
	}
	commonValue1 := int64(10)
	commonValue2 := int64(7)
	commonValue3 := int64(13)
	commonValue4 := float64(13)
	var customWriter http.ResponseWriter = &CustomResponseWriter{}
	tests := []struct {
		name            string
		args            args
		triggerCount    int
		BaseDirEnvValue string
		wantStatusCode  []string
		wantAnswer      string
		wantFileRecord  string
	}{
		{
			name: "common1",
			args: args{
				customWriter,
				NewCounter(&commonValue1),
				"Blabla1",
			},
			triggerCount:    1,
			BaseDirEnvValue: t.TempDir(),
			wantStatusCode:  []string{StatusOK},
			wantAnswer:      "updated mapMetric",
			wantFileRecord:  "key: Blabla1 -> value: 10\n",
		},
		{
			name: "common2",
			args: args{
				customWriter,
				NewCounter(&commonValue2),
				"Blabla2",
			},
			triggerCount:    1,
			BaseDirEnvValue: filepath.Join(t.TempDir(), "test_folder"),
			wantStatusCode:  []string{StatusOK},
			wantAnswer:      "updated mapMetric",
			wantFileRecord:  "key: Blabla2 -> value: 7\n",
		},
		{
			name: "common_counter_repeated",
			args: args{
				customWriter,
				NewCounter(&commonValue3),
				"Blabla3",
			},
			triggerCount:    2,
			BaseDirEnvValue: t.TempDir(),
			wantStatusCode:  []string{StatusOK},
			wantAnswer:      "updated mapMetric",
			wantFileRecord:  "key: Blabla3 -> value: 26\n",
		},
		{
			name: "common_gauge_repeated",
			args: args{
				customWriter,
				NewGauge(&commonValue4),
				"Blabla4",
			},
			triggerCount:    2,
			BaseDirEnvValue: t.TempDir(),
			wantStatusCode:  []string{StatusOK},
			wantAnswer:      "updated mapMetric",
			wantFileRecord:  "key: Blabla4 -> value: 13\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.BaseDirEnvValue, t.TempDir())
			for i := 0; i < tt.triggerCount; i++ {
				SaveMetric(tt.args.w, tt.args.metric, tt.args.metricName)
			}
			header := customWriter.Header()
			assert.EqualValues(t, header["Status-Code"], tt.wantStatusCode)
			assert.Contains(
				t,
				header["Record"][0],
				tt.wantAnswer,
			)
			f, err := os.ReadFile(filepath.Join(os.Getenv(BaseDirEnv), "map.txt"))
			if err != nil {
				panic(err)
			}
			assert.EqualValues(t, string(f), tt.wantFileRecord)
			MapMetric.m = make(map[string]*Metric)
		})
	}
}

func TestUpdate(t *testing.T) {
	type args struct {
		w       http.ResponseWriter
		request *http.Request
	}
	var customWriter http.ResponseWriter = &CustomResponseWriter{}
	tests := []struct {
		name           string
		args           args
		wantStatusCode string
		wantAnswer     string
	}{
		{
			name: "common1",
			args: args{
				w: customWriter,
				request: &http.Request{
					Method: http.MethodPost,
					URL: &url.URL{
						Host: "not important",
						Path: "/update/counter/A/10",
					},
					Header: http.Header{
						"Content-Type": []string{"text/plain"},
					},
				},
			},
			wantStatusCode: StatusOK,
			wantAnswer:     "updated mapMetric",
		},
		{
			name: "common_bad_metric_type",
			args: args{
				w: customWriter,
				request: &http.Request{
					Method: http.MethodPost,
					URL: &url.URL{
						Host: "not important",
						Path: "/update/counter1/A/10",
					},
					Header: http.Header{
						"Content-Type": []string{"text/plain"},
					},
				},
			},
			wantStatusCode: StatusBadRequest,
			wantAnswer:     "Wrong type of metric: counter1",
		},
		{
			name: "common_inconvertible_type",
			args: args{
				w: customWriter,
				request: &http.Request{
					Method: http.MethodPost,
					URL: &url.URL{
						Host: "not important",
						Path: "/update/counter/A/none",
					},
					Header: http.Header{
						"Content-Type": []string{"text/plain"},
					},
				},
			},
			wantStatusCode: StatusBadRequest,
			wantAnswer:     "Bad type of value passed. Please be sure that it can be converted to int64",
		},
		{
			name: "common_not_found",
			args: args{
				w: customWriter,
				request: &http.Request{
					Method: http.MethodPost,
					URL: &url.URL{
						Host: "not important",
						Path: "/update/A/10",
					},
					Header: http.Header{
						"Content-Type": []string{"text/plain"},
					},
				},
			},
			wantStatusCode: StatusNotFound,
			wantAnswer:     "Wrong path length. Please use the format: /update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>",
		},
		{
			name: "common_not_allowed_method",
			args: args{
				w: customWriter,
				request: &http.Request{
					Method: http.MethodGet,
					URL: &url.URL{
						Host: "not important",
						Path: "/update/counter/A/10",
					},
					Header: http.Header{
						"Content-Type": []string{"text/plain"},
					},
				},
			},
			wantStatusCode: StatusBadRequest,
			wantAnswer:     "Others methods are not allowed. Have got: GET",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Update(tt.args.w, tt.args.request)
			header := customWriter.Header()
			assert.EqualValues(t, tt.wantStatusCode, header["Status-Code"][0])
			assert.Contains(
				t,
				header["Record"][0],
				tt.wantAnswer,
			)
		})
	}
}
