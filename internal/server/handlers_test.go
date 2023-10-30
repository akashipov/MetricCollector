package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/akashipov/MetricCollector/internal/general"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var StatusOK string = fmt.Sprintf("%v", http.StatusOK)

type CustomResponseWriter struct {
	header http.Header
}

func (r *CustomResponseWriter) Write(bytes []byte) (int, error) {
	if r.header == nil {
		r.header = make(map[string][]string)
	}
	return 6, nil
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
		metric     general.Metric
		metricName string
	}
	var commonMetric1 general.Metric = general.NewCounter(int64(10))
	var commonMetric2 general.Metric = general.NewCounter(int64(7))
	var commonMetric3 general.Metric = general.NewCounter(int64(26))
	var commonMetric4 general.Metric = general.NewGauge(float64(13))
	var customWriter http.ResponseWriter = &CustomResponseWriter{}
	tests := []struct {
		name            string
		args            args
		triggerCount    int
		BaseDirEnvValue string
		wantStatusCode  []string
		wantMap         map[string]general.Metric
	}{
		{
			name: "common1",
			args: args{
				customWriter,
				commonMetric1,
				"Blabla1",
			},
			triggerCount:    1,
			BaseDirEnvValue: t.TempDir(),
			wantStatusCode:  []string{StatusOK},
			wantMap:         map[string]general.Metric{"Blabla1": commonMetric1},
		},
		{
			name: "common2",
			args: args{
				customWriter,
				commonMetric2,
				"Blabla2",
			},
			triggerCount:    1,
			BaseDirEnvValue: filepath.Join(t.TempDir(), "test_folder"),
			wantStatusCode:  []string{StatusOK},
			wantMap:         map[string]general.Metric{"Blabla2": commonMetric2},
		},
		{
			name: "common_counter_repeated",
			args: args{
				customWriter,
				general.NewCounter(int64(13)),
				"Blabla3",
			},
			triggerCount:    2,
			BaseDirEnvValue: t.TempDir(),
			wantStatusCode:  []string{StatusOK},
			wantMap:         map[string]general.Metric{"Blabla3": commonMetric3},
		},
		{
			name: "common_gauge_repeated",
			args: args{
				customWriter,
				commonMetric4,
				"Blabla4",
			},
			triggerCount:    2,
			BaseDirEnvValue: t.TempDir(),
			wantStatusCode:  []string{StatusOK},
			wantMap:         map[string]general.Metric{"Blabla4": commonMetric4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.BaseDirEnvValue, t.TempDir())
			for i := 0; i < tt.triggerCount; i++ {
				SaveMetric(tt.args.w, tt.args.metric, tt.args.metricName)
			}
			header := customWriter.Header()
			assert.EqualValues(t, tt.wantStatusCode, header["Status-Code"])
			assert.Equal(t, len(MapMetric.m), len(tt.wantMap))
			for k, v := range tt.wantMap {
				actualValue, ok := MapMetric.m[k]
				require.True(t, ok)
				assert.Equal(t, v.GetValue(), actualValue.GetValue())
			}
			MapMetric.m = make(map[string]general.Metric)
		})
	}
}

func TestUpdate(t *testing.T) {
	type args struct {
		Method      string
		URL         string
		contentType string
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	s := *logger.Sugar()
	server := httptest.NewServer(ServerRouter(&s))
	defer server.Close()
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
		wantAnswer     string
	}{
		{
			name: "common_ok",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/update/counter/A/10",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "",
		},
		{
			name: "common_bad_metric_type",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/update/counter1/A/10",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusBadRequest,
			wantAnswer:     "Wrong type of metric: 'counter1'",
		},
		{
			name: "common_inconvertible_type",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/update/counter/A/none",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusBadRequest,
			wantAnswer:     "Bad type of value passed. Please be sure that it can be converted to int64",
		},
		{
			name: "common_not_found",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/update/A/10",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusNotFound,
			wantAnswer:     "",
		},
		{
			name: "common_not_allowed_method",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL + "/update/counter/A/10",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantAnswer:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := resty.New()
			r := c.R().ForceContentType(tt.args.contentType)
			var resp *resty.Response
			var err error
			switch tt.args.Method {
			case http.MethodPost:
				resp, err = r.Post(tt.args.URL)
			case http.MethodGet:
				resp, err = r.Get(tt.args.URL)
			}

			if err != nil {
				panic(err)
			}
			assert.EqualValues(t, tt.wantStatusCode, resp.StatusCode())
			assert.Contains(
				t,
				resp.String(),
				tt.wantAnswer,
			)
			MapMetric.m = make(map[string]general.Metric)
		})
	}
}

func Encode(data []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(data); err != nil {
		fmt.Println(err.Error())
	}
	if err := gz.Close(); err != nil {
		fmt.Println(err.Error())
	}
	return b.Bytes()
}

func TestUpdateShortForm(t *testing.T) {
	type args struct {
		Method          string
		URL             string
		contentType     string
		contentEncoding bool
		Body            []byte
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	s := *logger.Sugar()
	server := httptest.NewServer(ServerRouter(&s))
	defer server.Close()
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
		wantAnswer     string
	}{
		{
			name: "common_ok",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/update",
				contentType: "application/json",
				Body:        []byte("{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}"),
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}",
		},
		{
			name: "common_ok_encoding",
			args: args{
				Method:          http.MethodPost,
				URL:             server.URL + "/update",
				contentType:     "application/json",
				contentEncoding: true,
				Body:            Encode([]byte("{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}")),
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}",
		},
		{
			name: "common_bad_metric_type",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/update",
				contentType: "application/json",
				Body:        []byte("{\"id\":\"A\",\"type\":\"counter1\",\"delta\":10}"),
			},
			wantStatusCode: http.StatusNotFound,
			wantAnswer:     "Wrong type of metric: 'counter1'",
		},
		{
			name: "common_inconvertible_type",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/update",
				contentType: "application/json",
				Body:        []byte("{\"id\":\"A\",\"type\":\"counter\",\"delta\":\"none\"}"),
			},
			wantStatusCode: http.StatusBadRequest,
			wantAnswer:     "field Metrics.delta of type int64",
		},
		{
			name: "common_not_found",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/update",
				contentType: "application/json",
				Body:        []byte("{\"id\":\"A\",\"delta\":10}"),
			},
			wantStatusCode: http.StatusNotFound,
			wantAnswer:     "",
		},
		{
			name: "common_not_allowed_method",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL + "/update",
				contentType: "application/json",
				Body:        []byte("{\"id\":\"A\",\"type\":\"counter\",\"delta\":\"10\"}"),
			},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantAnswer:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := resty.New()
			r := c.R().ForceContentType(tt.args.contentType).SetBody(tt.args.Body)
			if tt.args.contentEncoding {
				r.SetHeader("Content-Encoding", "gzip")
			}
			var resp *resty.Response
			var err error
			switch tt.args.Method {
			case http.MethodPost:
				resp, err = r.Post(tt.args.URL)
			case http.MethodGet:
				resp, err = r.Get(tt.args.URL)
			}

			if err != nil {
				panic(err)
			}
			assert.EqualValues(t, tt.wantStatusCode, resp.StatusCode())
			assert.Contains(
				t,
				resp.String(),
				tt.wantAnswer,
			)
			MapMetric.m = make(map[string]general.Metric)
		})
	}
}

func TestGetMetricShortForm(t *testing.T) {
	type args struct {
		Method          string
		URL             string
		contentType     string
		contentEncoding bool
		Body            []byte
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	s := *logger.Sugar()
	server := httptest.NewServer(ServerRouter(&s))
	a := int64(10)
	MapMetric.m = make(map[string]general.Metric)
	MapMetric.m["A"] = general.NewCounter(a)
	b := float64(17)
	MapMetric.m["B"] = general.NewGauge(b)
	defer server.Close()
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
		wantAnswer     string
	}{
		{
			name: "common_counter_ok",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/value",
				contentType: "application/json",
				Body:        []byte("{\"type\":\"counter\",\"id\":\"A\"}"),
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}",
		},
		{
			name: "common_counter_ok_encoding",
			args: args{
				Method:          http.MethodPost,
				URL:             server.URL + "/value",
				contentType:     "application/json",
				contentEncoding: true,
				Body:            Encode([]byte("{\"type\":\"counter\",\"id\":\"A\"}")),
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}",
		},
		{
			name: "common_gauge_ok",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/value",
				contentType: "application/json",
				Body:        []byte("{\"type\":\"gauge\",\"id\":\"B\"}"),
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "{\"id\":\"B\",\"type\":\"gauge\",\"value\":17}",
		},
		{
			name: "common_gauge_wrong_type",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/value",
				contentType: "application/json",
				Body:        []byte("{\"id\":\"C\",\"type\":\"counter\"}"),
			},
			wantStatusCode: http.StatusNotFound,
			wantAnswer:     "There is no metric like this: 'C'",
		},
		{
			name: "common_gauge_wrong_type",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/value",
				contentType: "application/json",
				Body:        []byte("{\"type\":\"counter\",\"id\":\"B\"}"),
			},
			wantStatusCode: http.StatusNotFound,
			wantAnswer:     "It has other metric type: 'gauge'",
		},
		{
			name: "common_gauge_base_dir",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL,
				contentType: "application/json",
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "<html><ul><li>A: 10</li><li>B: 17</li></ul></html>",
		},
		{
			name: "common_not_allowed_get_base",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL + "/value",
				contentType: "application/json",
			},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantAnswer:     "",
		},
		{
			name: "common_not_allowed_get_base",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL + "/value",
				contentType: "application/json",
				Body:        []byte("{\"type\":\"gauge\",\"id\":\"B\"}"),
			},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantAnswer:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := resty.New()
			r := c.R().ForceContentType(tt.args.contentType).SetBody(tt.args.Body)
			if tt.args.contentEncoding {
				r.SetHeader("Content-Encoding", "gzip")
			}
			var resp *resty.Response
			var err error
			switch tt.args.Method {
			case http.MethodPost:
				resp, err = r.Post(tt.args.URL)
			case http.MethodGet:
				resp, err = r.Get(tt.args.URL)
			}

			if err != nil {
				panic(err)
			}
			assert.EqualValues(t, tt.wantStatusCode, resp.StatusCode())
			assert.Contains(
				t,
				resp.String(),
				tt.wantAnswer,
			)
		})
	}
}

func TestGetMetric(t *testing.T) {
	type args struct {
		Method         string
		URL            string
		contentType    string
		acceptEncoding bool
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		panic(err)
	}
	defer logger.Sync()
	s := *logger.Sugar()
	server := httptest.NewServer(ServerRouter(&s))
	a := int64(10)
	MapMetric.m = make(map[string]general.Metric)
	MapMetric.m["A"] = general.NewCounter(a)
	b := float64(17)
	MapMetric.m["B"] = general.NewGauge(b)
	defer server.Close()
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
		wantAnswer     string
	}{
		{
			name: "common_counter_ok",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL + "/value/counter/A",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "10",
		},
		{
			name: "common_gauge_ok",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL + "/value/gauge/B",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "17",
		},
		{
			name: "common_gauge_wrong_type",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL + "/value/counter/C",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusNotFound,
			wantAnswer:     "There is no metric like this: C",
		},
		{
			name: "common_gauge_wrong_type",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL + "/value/counter/B",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusNotFound,
			wantAnswer:     "It has other metric type: 'gauge'",
		},
		{
			name: "common_gauge_base_dir",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL,
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "<html><ul><li>A: 10</li><li>B: 17</li></ul></html>",
		},
		{
			name: "common_gauge_base_dir_encoding",
			args: args{
				Method:         http.MethodGet,
				URL:            server.URL,
				contentType:    "text/html",
				acceptEncoding: true,
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "<html><ul><li>A: 10</li><li>B: 17</li></ul></html>",
		},
		{
			name: "common_not_allowed_post_base",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL,
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantAnswer:     "",
		},
		{
			name: "common_not_allowed_post_base",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/value/gauge/B",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantAnswer:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := resty.New()
			r := c.R().SetHeader("Content-Type", tt.args.contentType)
			if tt.args.acceptEncoding {
				r.SetHeader("Accept-Encoding", "gzip")
			}
			var resp *resty.Response
			var err error
			switch tt.args.Method {
			case http.MethodPost:
				resp, err = r.Post(tt.args.URL)
			case http.MethodGet:
				resp, err = r.Get(tt.args.URL)
			}

			if err != nil {
				panic(err)
			}
			if tt.args.acceptEncoding {
				assert.Equal(t, resp.Header().Get("Content-Encoding"), "")
			}
			assert.EqualValues(t, tt.wantStatusCode, resp.StatusCode())
			assert.Contains(
				t,
				resp.String(),
				tt.wantAnswer,
			)
		})
	}
}
