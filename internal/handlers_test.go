package internal

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

var StatusOK string = fmt.Sprintf("%v", http.StatusOK)

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
	var commonMetric1 Metric = NewCounter(&commonValue1)
	commonValue2 := int64(7)
	var commonMetric2 Metric = NewCounter(&commonValue2)
	commonValue3 := int64(13)
	commonValue3R := int64(26)
	var commonMetric3 Metric = NewCounter(&commonValue3R)
	commonValue4 := float64(13)
	var commonMetric4 Metric = NewGauge(&commonValue4)
	var customWriter http.ResponseWriter = &CustomResponseWriter{}
	tests := []struct {
		name            string
		args            args
		triggerCount    int
		BaseDirEnvValue string
		wantStatusCode  []string
		wantAnswer      string
		wantMap         map[string]Metric
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
			wantAnswer:      "updated mapMetric",
			wantMap:         map[string]Metric{"Blabla1": commonMetric1},
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
			wantAnswer:      "updated mapMetric",
			wantMap:         map[string]Metric{"Blabla2": commonMetric2},
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
			wantMap:         map[string]Metric{"Blabla3": commonMetric3},
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
			wantAnswer:      "updated mapMetric",
			wantMap:         map[string]Metric{"Blabla4": commonMetric4},
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
			assert.Equal(t, len(MapMetric.m), len(tt.wantMap))
			for k, v := range tt.wantMap {
				actualValue, ok := MapMetric.m[k]
				require.True(t, ok)
				assert.Equal(t, v.GetValue(), actualValue.GetValue())
			}
			MapMetric.m = make(map[string]Metric)
		})
	}
}

func TestUpdate(t *testing.T) {
	type args struct {
		Method      string
		URL         string
		contentType string
	}
	server := httptest.NewServer(ServerRouter())
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
			wantAnswer:     "updated mapMetric",
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
		})
	}
}
