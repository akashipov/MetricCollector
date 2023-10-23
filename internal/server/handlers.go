package server

import (
	"fmt"
	"github.com/akashipov/MetricCollector/internal/server/logger"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

func ServerRouter(s *zap.SugaredLogger) chi.Router {
	r := chi.NewRouter()
	r.Get("/", MainPage)
	r.Route(
		"/update/{MetricType}/{MetricName}/{MetricValue}",
		func(r chi.Router) {
			r.Post("/", logger.WithLogging(http.HandlerFunc(Update), s))
		},
	)
	r.Route(
		"/value/{MetricType}/{MetricName}",
		func(r chi.Router) {
			r.Get("/", logger.WithLogging(http.HandlerFunc(GetMetric), s))
		},
	)
	return r
}

func SaveMetric(w http.ResponseWriter, metric Metric, metricName string) {
	val, ok := MapMetric.m[metricName]
	errFMT := "error - %s: status - %v\n"
	if ok {
		ok = val.Update(metric.GetValue())
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Println("Bad type of metric value is passed for already existed")
		}
	} else {
		MapMetric.m[metricName] = metric
	}
	w.WriteHeader(http.StatusOK)
	status, err := w.Write([]byte(fmt.Sprintf("updated mapMetric: %v", MapMetric)))
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Printf(errFMT, err.Error(), status)
	}
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
