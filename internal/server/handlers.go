package server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/akashipov/MetricCollector/internal/agent"
	"github.com/akashipov/MetricCollector/internal/general"
	"github.com/akashipov/MetricCollector/internal/server/logger"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

func ServerRouter(s *zap.SugaredLogger) chi.Router {
	r := chi.NewRouter()
	r.Get("/", MainPage)
	r.Route(
		"/update",
		func(r chi.Router) {
			r.Route(
				"/{MetricType}/{MetricName}/{MetricValue}",
				func(r chi.Router) {
					r.Post("/", logger.WithLogging(http.HandlerFunc(Update), s))
				},
			)
			r.Post("/", logger.WithLogging(http.HandlerFunc(UpdateShortForm), s))
		},
	)
	r.Route(
		"/value",
		func(r chi.Router) {
			r.Route(
				"/{MetricType}/{MetricName}",
				func(r chi.Router) {
					r.Get("/", logger.WithLogging(http.HandlerFunc(GetMetric), s))
				},
			)
			r.Post("/", logger.WithLogging(http.HandlerFunc(GetMetricShortForm), s))
		},
	)
	return r
}

func Update(w http.ResponseWriter, request *http.Request) {
	MetricType := chi.URLParam(request, "MetricType")
	MetricName := chi.URLParam(request, "MetricName")
	MetricValue := chi.URLParam(request, "MetricValue")
	m, err := ValidateMetric(&w, MetricType, MetricValue)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if m == nil {
		return
	}
	SaveMetric(w, m, MetricName)
}

func SaveMetric(w http.ResponseWriter, metric general.Metric, metricName string) error {
	val, ok := MapMetric.m[metricName]
	if ok {
		ok = val.Update(metric.GetValue())
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return errors.New("bad type of metric value is passed for already existed")
		}
	} else {
		MapMetric.m[metricName] = metric
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func UpdateShortForm(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	var metric general.Metrics
	_, err := buf.ReadFrom(request.Body)
	defer request.Body.Close()
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	data := buf.Bytes()
	data, err = Decode(&w, request, data)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(data, &metric)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Unmarshal problem: %s", err.Error())))
		return
	}
	MetricType := metric.MType
	MetricName := metric.ID
	var MetricValue interface{}
	switch metric.MType {
	case agent.GAUGE:
		MetricValue = *metric.Value
	case agent.COUNTER:
		MetricValue = *metric.Delta
	default:
		print(errors.New("wrong type of metric type"))
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("Wrong type of metric: '%s'", MetricType)))
		return
	}
	m, err := ValidateMetric(&w, MetricType, MetricValue)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if m == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = SaveMetric(w, m, MetricName)
	switch MetricType {
	case agent.COUNTER:
		v, ok := MapMetric.m[MetricName].GetValue().(int64)
		if ok {
			metric.Delta = &v
		}
	case agent.GAUGE:
		v, ok := MapMetric.m[MetricName].GetValue().(float64)
		if ok {
			metric.Value = &v
		}
	}

	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Printf("error - %s", err.Error())
	}
	w.WriteHeader(http.StatusOK)
	b, err := json.Marshal(metric)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Println(err.Error())
		return
	}
	w.Write(b)
}

func MainPage(w http.ResponseWriter, request *http.Request) {
	ul := "<ul>"
	var keys []string
	for k := range MapMetric.m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ul += fmt.Sprintf("<li>%v: %v</li>", k, MapMetric.m[k].GetValue())
	}
	ul += "</ul>"
	html := fmt.Sprintf("<html>%s</html>", ul)
	w.WriteHeader(http.StatusOK)
	status, err := w.Write([]byte(html))
	if err != nil {
		fmt.Printf("%v: %v", status, err.Error())
	}
}

func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") &&
			!strings.Contains(r.Header.Get("Content-Type"), "text/html") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(general.GzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func Decode(w *http.ResponseWriter, request *http.Request, data []byte) ([]byte, error) {
	if strings.Contains(request.Header.Get("Content-Encoding"), "gzip") {
		reader := bytes.NewReader(data)
		gzreader, err := gzip.NewReader(reader)
		if err != nil {
			(*w).WriteHeader(http.StatusBadGateway)
			fmt.Println(err.Error())
			return nil, err
		}
		data, err = io.ReadAll(gzreader)
		if err != nil {
			(*w).WriteHeader(http.StatusBadGateway)
			fmt.Println(err.Error())
			return nil, err
		}
	}
	return data, nil
}

func GetMetricShortForm(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var buf bytes.Buffer
	var metric general.Metrics
	_, err := buf.ReadFrom(request.Body)
	defer request.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err.Error())
		return
	}
	data := buf.Bytes()
	data, err = Decode(&w, request, data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err.Error())
		return
	}
	json.Unmarshal(data, &metric)
	MetricName := metric.ID
	MetricType := metric.MType
	var answer []byte
	if MetricValue, ok := MapMetric.m[MetricName]; ok {
		typeMetricValue := strings.ToLower(reflect.TypeOf(MetricValue).Elem().Name())
		if typeMetricValue == MetricType {
			w.WriteHeader(http.StatusOK)
			switch MetricType {
			case agent.COUNTER:
				v, ok := MetricValue.GetValue().(int64)
				if ok {
					metric.Delta = &v
					answer, err = json.Marshal(metric)
					if err != nil {
						answer = []byte(err.Error())
					}
				} else {
					answer = []byte("Something wrong with type of metric")
				}
			case agent.GAUGE:
				v, ok := MetricValue.GetValue().(float64)
				if ok {
					metric.Value = &v
					answer, err = json.Marshal(metric)
					if err != nil {
						answer = []byte(err.Error())
					}
				} else {
					answer = []byte("Something wrong with type of metric")
				}
			default:
				answer = []byte("Wrong type of metric")
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			answer = []byte(fmt.Sprintf("It has other metric type: '%s'", typeMetricValue))
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		answer = []byte(fmt.Sprintf("There is no metric like this: '%v'", MetricName))
	}
	status, err := w.Write(answer)
	if err != nil {
		fmt.Printf("%v: %v", status, err.Error())
	}
}

func GetMetric(w http.ResponseWriter, request *http.Request) {
	MetricName := chi.URLParam(request, "MetricName")
	MetricType := chi.URLParam(request, "MetricType")
	var answer string
	if MetricValue, ok := MapMetric.m[MetricName]; ok {
		typeMetricValue := strings.ToLower(reflect.TypeOf(MetricValue).Elem().Name())
		if typeMetricValue == MetricType {
			w.WriteHeader(http.StatusOK)
			answer = fmt.Sprintf("%v", MetricValue.GetValue())
		} else {
			w.WriteHeader(http.StatusNotFound)
			answer = fmt.Sprintf("It has other metric type: '%s'", typeMetricValue)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		answer = fmt.Sprintf("There is no metric like this: %v", MetricName)
	}
	status, err := w.Write([]byte(answer))
	if err != nil {
		fmt.Printf("%v: %v", status, err.Error())
	}
}
