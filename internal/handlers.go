package internal

import (
	"fmt"
	"github.com/go-chi/chi"
	"net/http"
	"strconv"
)

func SaveMetric(w http.ResponseWriter, metric Metric, metricName string) {
	val, ok := MapMetric.m[metricName]
	if ok {
		val.Update(metric.GetValue())
	} else {
		if MapMetric.m == nil {
			m := make(map[string]Metric)
			m[metricName] = metric
			MapMetric.m = m
		} else {
			MapMetric.m[metricName] = metric
		}
	}
	w.WriteHeader(http.StatusOK)
	status, err := w.Write([]byte(fmt.Sprintf("updated mapMetric: %v", MapMetric)))
	if err != nil {
		panic(fmt.Sprintf("%s: %v", err.Error(), status))
	}
}

func Update(w http.ResponseWriter, request *http.Request) {
	MetricType := chi.URLParam(request, "MetricType")
	MetricName := chi.URLParam(request, "MetricName")
	MetricValue := chi.URLParam(request, "MetricValue")

	// I don't know how to escape repetition of code here. Because I have 2 different type of map value
	badTypeValueMsg := "Bad type of value passed. Please be sure that it can be converted to "
	if MetricType == GAUGE {
		n, err := strconv.ParseFloat(MetricValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			status, err := w.Write([]byte(badTypeValueMsg + fmt.Sprintf("float64: '%v'", MetricValue)))
			if err != nil {
				panic(fmt.Sprintf("%s: %v", err.Error(), status))
			}
			return
		}
		SaveMetric(w, NewGauge(&n), MetricName)
		return
	} else if MetricType == COUNTER {
		n, err := strconv.ParseInt(MetricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			status, err := w.Write([]byte(badTypeValueMsg + fmt.Sprintf("int64: '%v'", MetricValue)))
			if err != nil {
				panic(fmt.Sprintf("%s: %v", err.Error(), status))
			}
			return
		}
		SaveMetric(w, NewCounter(&n), MetricName)
		return
	} else {
		w.WriteHeader(http.StatusBadRequest)
		status, err := w.Write(
			[]byte(
				fmt.Sprintf("Wrong type of metric: '%s'", MetricType),
			),
		)
		if err != nil {
			panic(fmt.Sprintf("%s: %v", err.Error(), status))
		}
	}
}

func ServerRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/", MainPage)
	r.Route(
		"/update/{MetricType}/{MetricName}/{MetricValue}",
		func(r chi.Router) {
			r.Post("/", Update)
		},
	)
	return r
}

func MainPage(w http.ResponseWriter, request *http.Request) {
	ul := "<ul>"
	for k, v := range MapMetric.m {
		ul += fmt.Sprintf("<li>%v: %v</li>", k, v.GetValue())
	}
	ul += "</ul>"
	html := fmt.Sprintf("<html>%s</html>", ul)
	status, err := w.Write([]byte(html))
	if err != nil {
		fmt.Printf("%v: %v", status, err.Error())
	}
}
