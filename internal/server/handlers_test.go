package server

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/akashipov/MetricCollector/internal/agent"
	"github.com/akashipov/MetricCollector/internal/general"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
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
	return 0, nil
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
		w      http.ResponseWriter
		metric *general.Metrics
	}
	i := int64(10)
	commonMetric1 := &general.Metrics{
		ID:    "Blabla1",
		MType: agent.COUNTER,
		Delta: &i,
	}
	i = int64(7)
	commonMetric2 := &general.Metrics{
		ID:    "Blabla2",
		MType: agent.COUNTER,
		Delta: &i,
	}
	i = int64(13)
	commonMetric3Before := &general.Metrics{
		ID:    "Blabla3",
		MType: agent.COUNTER,
		Delta: &i,
	}
	i = int64(26)
	commonMetric3 := &general.Metrics{
		ID:    "Blabla3",
		MType: agent.COUNTER,
		Delta: &i,
	}
	f := float64(13)
	commonMetric4 := &general.Metrics{
		ID:    "Blabla4",
		MType: agent.GAUGE,
		Value: &f,
	}
	var customWriter http.ResponseWriter = &CustomResponseWriter{}
	tests := []struct {
		name            string
		args            args
		triggerCount    int
		BaseDirEnvValue string
		wantStatusCode  []string
		wantMap         []general.Metrics
	}{
		{
			name: "common1",
			args: args{
				customWriter,
				commonMetric1,
			},
			triggerCount:    1,
			BaseDirEnvValue: t.TempDir(),
			wantStatusCode:  nil,
			wantMap:         []general.Metrics{*commonMetric1},
		},
		{
			name: "common2",
			args: args{
				customWriter,
				commonMetric2,
			},
			triggerCount:    1,
			BaseDirEnvValue: filepath.Join(t.TempDir(), "test_folder"),
			wantStatusCode:  nil,
			wantMap:         []general.Metrics{*commonMetric2},
		},
		{
			name: "common_counter_repeated",
			args: args{
				customWriter,
				commonMetric3Before,
			},
			triggerCount:    2,
			BaseDirEnvValue: t.TempDir(),
			wantStatusCode:  nil,
			wantMap:         []general.Metrics{*commonMetric3},
		},
		{
			name: "common_gauge_repeated",
			args: args{
				customWriter,
				commonMetric4,
			},
			triggerCount:    2,
			BaseDirEnvValue: t.TempDir(),
			wantStatusCode:  nil,
			wantMap:         []general.Metrics{*commonMetric4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.BaseDirEnvValue, t.TempDir())
			for i := 0; i < tt.triggerCount; i++ {
				SaveMetric(tt.args.w, tt.args.metric)
			}
			header := customWriter.Header()
			assert.EqualValues(t, tt.wantStatusCode, header["Status-Code"])
			assert.Equal(t, len(MapMetric.MetricList), len(tt.wantMap))
			for _, v := range tt.wantMap {
				actualValue := MapMetric.MetricList[v.ID]
				assert.Equal(t, v.Delta, actualValue.Delta)
				assert.Equal(t, v.Value, actualValue.Value)
			}
			MapMetric.MetricList = make(map[string]*general.Metrics, 0)
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
			assert.NotContains(t, resp.Header(), "Content-Encoding")
			assert.EqualValues(t, tt.wantStatusCode, resp.StatusCode())
			assert.Contains(
				t,
				resp.String(),
				tt.wantAnswer,
			)
			MapMetric.MetricList = make(map[string]*general.Metrics, 0)
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

func DecodeBytes(data []byte) ([]byte, error) {
	buf := bytes.NewReader(data)
	var err error
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return []byte(""), errors.New("Create reader block: " + err.Error())
	}
	b, err := io.ReadAll(gz)
	if err := gz.Close(); err != nil {
		return []byte(""), errors.New("Close reader block: " + err.Error())
	}
	return b, nil
}

func TestUpdateShortForm(t *testing.T) {
	type args struct {
		Method         string
		URL            string
		contentType    string
		IsEncodedReq   bool
		Body           []byte
		acceptEncoding string
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
			name: "common_ok_simple",
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
				Method:       http.MethodPost,
				URL:          server.URL + "/update",
				contentType:  "application/json",
				IsEncodedReq: true,
				Body:         Encode([]byte("{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}")),
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
		{
			name: "common_ok_ae_true",
			args: args{
				Method:         http.MethodPost,
				URL:            server.URL + "/update",
				contentType:    "application/json",
				Body:           []byte("{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}"),
				acceptEncoding: "gzip",
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}",
		},
		{
			name: "common_ok_ae_false",
			args: args{
				Method:         http.MethodPost,
				URL:            server.URL + "/update",
				contentType:    "application/json",
				Body:           []byte("{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}"),
				acceptEncoding: "",
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     "{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := resty.New()
			r := c.R().SetHeader("Content-Type", tt.args.contentType).SetBody(tt.args.Body)
			if tt.args.IsEncodedReq {
				r.SetHeader("Content-Encoding", "gzip")
			}
			r.SetHeader("Accept-Encoding", fmt.Sprintf("%v", tt.args.acceptEncoding))
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
			if resp.StatusCode() == http.StatusOK {
				fmt.Println("All list of content-type:", resp.Header().Values("Content-Type"))
				assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
			} else {
				assert.Contains(t, resp.Header().Get("Content-Type"), "") // it can be text/plain or empty
			}
			if tt.args.acceptEncoding == "gzip" {
				assert.Equal(t, "gzip", resp.Header().Get("Content-Encoding"))
			} else {
				fmt.Println(resp.Header().Get("Content-Encoding"))
			}
			assert.EqualValues(t, tt.wantStatusCode, resp.StatusCode())
			assert.Contains(
				t,
				resp.String(),
				tt.wantAnswer,
			)
			MapMetric.MetricList = make(map[string]*general.Metrics, 0)
		})
	}
}

func TestGetMetricShortForm(t *testing.T) {
	type args struct {
		Method        string
		URL           string
		contentType   string
		IsEncodedResp bool
		IsEncodedReq  bool
		Body          []byte
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	s := *logger.Sugar()
	server := httptest.NewServer(ServerRouter(&s))

	MapMetric.MetricList = make(map[string]*general.Metrics, 0)
	a := int64(10)
	MapMetric.MetricList["A"] = &general.Metrics{ID: "A", MType: agent.COUNTER, Delta: &a}
	b := float64(17)
	MapMetric.MetricList["B"] = &general.Metrics{ID: "B", MType: agent.GAUGE, Value: &b}
	defer server.Close()
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
		wantAnswer     []string
	}{
		{
			name: "common_counter_ok_sf",
			args: args{
				Method:        http.MethodPost,
				URL:           server.URL + "/value",
				contentType:   "application/json",
				Body:          []byte("{\"type\":\"counter\",\"id\":\"A\"}"),
				IsEncodedResp: true,
				IsEncodedReq:  false,
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     []string{"{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}"},
		},
		{
			name: "common_counter_ok_encoding",
			args: args{
				Method:        http.MethodPost,
				URL:           server.URL + "/value",
				contentType:   "application/json",
				IsEncodedReq:  true,
				IsEncodedResp: true,
				Body:          Encode([]byte("{\"type\":\"counter\",\"id\":\"A\"}")),
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     []string{"{\"id\":\"A\",\"type\":\"counter\",\"delta\":10}"},
		},
		{
			name: "common_gauge_ok",
			args: args{
				Method:        http.MethodPost,
				URL:           server.URL + "/value",
				contentType:   "application/json",
				IsEncodedResp: true,
				Body:          []byte("{\"type\":\"gauge\",\"id\":\"B\"}"),
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     []string{"{\"id\":\"B\",\"type\":\"gauge\",\"value\":17}"},
		},
		{
			name: "common_gauge_wrong_type",
			args: args{
				Method:        http.MethodPost,
				URL:           server.URL + "/value",
				contentType:   "application/json",
				IsEncodedResp: false,
				Body:          []byte("{\"id\":\"C\",\"type\":\"counter\"}"),
			},
			wantStatusCode: http.StatusNotFound,
			wantAnswer:     []string{"There is no metric like this: 'C'"},
		},
		{
			name: "common_gauge_wrong_type",
			args: args{
				Method:        http.MethodPost,
				URL:           server.URL + "/value",
				contentType:   "application/json",
				IsEncodedResp: false,
				Body:          []byte("{\"type\":\"counter\",\"id\":\"B\"}"),
			},
			wantStatusCode: http.StatusNotFound,
			wantAnswer:     []string{"It has other metric type: 'gauge'"},
		},
		{
			name: "root_url_list_encoded_req",
			args: args{
				Method:        http.MethodGet,
				URL:           server.URL,
				contentType:   "text/plain",
				IsEncodedResp: true,
				IsEncodedReq:  true,
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     []string{"<li>A: 10</li>", "<li>B: 17</li>"},
		},
		{
			name: "common_not_allowed_get_base",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL + "/value",
				contentType: "application/json",
			},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantAnswer:     []string{""},
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
			wantAnswer:     []string{""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := resty.New()
			r := c.R().SetHeader("Content-Type", tt.args.contentType).SetBody(tt.args.Body)
			r.SetHeader("Accept-Encoding", "gzip")
			if tt.args.IsEncodedReq {
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
			if tt.args.IsEncodedResp {
				assert.Equal(t, "gzip", resp.Header().Get("Content-Encoding"))
			} else {
				assert.Equal(t, "", resp.Header().Get("Content-Encoding"))
			}
			if resp.StatusCode() == http.StatusOK {
				fmt.Println("All list of content-type:", resp.Header().Values("Content-Type"))
				assert.Contains(t, []string{"application/json", "text/html"}, resp.Header().Get("Content-Type"))
			} else {
				assert.Contains(t, resp.Header().Get("Content-Type"), "") // it can be text/plain or empty
			}
			fmt.Println("Have got:", resp.Body())
			decoded := string(resp.Body())
			for _, v := range tt.wantAnswer {
				assert.Contains(
					t,
					string(decoded),
					v,
				)
			}
		})
	}
}

func TestGetMetricFull(t *testing.T) {
	type args struct {
		Method        string
		URL           string
		contentType   string
		IsEncodedResp bool
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
	MapMetric.MetricList = make(map[string]*general.Metrics, 0)
	MapMetric.MetricList["A"] = &general.Metrics{ID: "A", MType: agent.COUNTER, Delta: &a}
	b := float64(17)
	MapMetric.MetricList["B"] = &general.Metrics{ID: "B", MType: agent.GAUGE, Value: &b}
	defer server.Close()
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
		wantAnswer     []string
	}{
		{
			name: "common_counter_ok_ff",
			args: args{
				Method:        http.MethodGet,
				URL:           server.URL + "/value/counter/A",
				contentType:   "text/plain",
				IsEncodedResp: true,
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     []string{"10"},
		},
		{
			name: "common_gauge_ok_ff",
			args: args{
				Method:        http.MethodGet,
				URL:           server.URL + "/value/gauge/B",
				contentType:   "text/plain",
				IsEncodedResp: true,
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     []string{"17"},
		},
		{
			name: "common_gauge_wrong_type",
			args: args{
				Method:        http.MethodGet,
				URL:           server.URL + "/value/counter/C",
				contentType:   "text/plain",
				IsEncodedResp: false,
			},
			wantStatusCode: http.StatusNotFound,
			wantAnswer:     []string{"There is no metric like this: C"},
		},
		{
			name: "common_gauge_wrong_type",
			args: args{
				Method:      http.MethodGet,
				URL:         server.URL + "/value/counter/B",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusNotFound,
			wantAnswer:     []string{"It has other metric type: 'gauge'"},
		},
		{
			name: "gauge_base_dir_full_form",
			args: args{
				Method:        http.MethodGet,
				URL:           server.URL,
				contentType:   "text/plain",
				IsEncodedResp: true,
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     []string{"<li>A: 10</li>", "<li>B: 17</li>"},
		},
		{
			name: "root_url_list_metrics_not_encoded_req",
			args: args{
				Method:        http.MethodGet,
				URL:           server.URL,
				contentType:   "text/plain",
				IsEncodedResp: true,
			},
			wantStatusCode: http.StatusOK,
			wantAnswer:     []string{"<li>A: 10</li>", "<li>B: 17</li>"},
		},
		{
			name: "common_bad_type_content_type",
			args: args{
				Method:        http.MethodGet,
				URL:           server.URL,
				contentType:   "text/html",
				IsEncodedResp: false,
			},
			wantStatusCode: http.StatusBadRequest,
			wantAnswer:     []string{"Bad type of content-type, please change it"},
		},
		{
			name: "common_not_allowed_post_base_root",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL,
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantAnswer:     []string{""},
		},
		{
			name: "common_not_allowed_post_base_get_value",
			args: args{
				Method:      http.MethodPost,
				URL:         server.URL + "/value/gauge/B",
				contentType: "text/plain",
			},
			wantStatusCode: http.StatusMethodNotAllowed,
			wantAnswer:     []string{""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := resty.New()
			r := c.R().SetHeader("Content-Type", tt.args.contentType)
			r.SetHeader("Accept-Encoding", "gzip")
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
			if tt.args.IsEncodedResp {
				assert.Equal(t, "gzip", resp.Header().Get("Content-Encoding"))
			} else {
				assert.Equal(t, "", resp.Header().Get("Content-Encoding"))
			}
			assert.EqualValues(t, tt.wantStatusCode, resp.StatusCode())
			decoded := resp.Body()
			fmt.Println("Have got:", string(decoded))
			for _, wanted := range tt.wantAnswer {
				assert.Contains(
					t,
					string(decoded),
					wanted,
				)
			}
		})
	}
}
