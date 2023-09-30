package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Gauge struct {
	Value float64
}

func (r *Gauge) Update(newValue float64) {
	r.Value = newValue
}

type Counter struct {
	Value int64
}

func (r *Counter) Update(newValue int64) {
	r.Value = r.Value + newValue
}

type MemStorageCounter struct {
	m map[string]*Counter
	//fmt.Stringer
}

type MemStorageGauge struct {
	m map[string]*Gauge
	//fmt.Stringer
}

var mapMetricGauge = MemStorageGauge{}
var mapMetricCounter = MemStorageCounter{}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func Update(w http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodPost {
		contentType := request.Header.Get("Content-Type")
		elements := strings.Split(request.URL.Path, "/")
		if contentType != "text/plain" {
			status, err := w.Write(
				[]byte("Request doesn't contain correct Content-Type"),
			)
			if err != nil {
				panic(fmt.Sprintf("%s: %v", err.Error(), status))
			}
			return
		}
		if len(elements) != 5 {
			w.WriteHeader(http.StatusNotFound)
			status, err := w.Write(
				[]byte("Wrong path length. Please use the format: /update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>"),
			)
			if err != nil {
				panic(fmt.Sprintf("%s: %v", err.Error(), status))
			}
			return
		}
		metricType := elements[2]
		metricName := elements[3]
		metricValue := elements[4]
		// I don't know how to escape repetition of code here. Because I have 2 different type of map value
		if metricType == "gauge" {
			n, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				status, err := w.Write([]byte("Bad type of value passed. Please be sure that it can be converted to float64"))
				if err != nil {
					panic(fmt.Sprintf("%s: %v", err.Error(), status))
				}
				return
			}
			val, ok := mapMetricGauge.m[metricName]
			if ok {
				val.Update(n)
			} else {
				if mapMetricGauge.m == nil {
					m := make(map[string]*Gauge)
					m[metricName] = &Gauge{n}
					mapMetricGauge.m = m
				} else {
					mapMetricGauge.m[metricName] = &Gauge{n}
				}
			}
			w.WriteHeader(http.StatusOK)
			status, err := w.Write([]byte(fmt.Sprintf("updated mapMetricGauge: %s", mapMetricGauge)))
			if err != nil {
				panic(fmt.Sprintf("%s: %v", err.Error(), status))
			}
			return
		} else if metricType == "counter" {
			n, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				status, err := w.Write([]byte("Bad type of value passed. Please be sure that it can be converted to int64"))
				if err != nil {
					panic(fmt.Sprintf("%s: %v", err.Error(), status))
				}
				return
			}
			val, ok := mapMetricCounter.m[metricName]
			if ok {
				val.Update(n)
			} else {
				if mapMetricCounter.m == nil {
					m := make(map[string]*Counter)
					m[metricName] = &Counter{n}
					mapMetricCounter.m = m
				} else {
					mapMetricCounter.m[metricName] = &Counter{n}
				}
			}
			w.WriteHeader(http.StatusOK)
			status, err := w.Write([]byte(fmt.Sprintf("updated mapMetricCounter: %s", mapMetricCounter)))
			if err != nil {
				panic(fmt.Sprintf("%s: %v", err.Error(), status))
			}
			return
		} else {
			w.WriteHeader(http.StatusBadRequest)
			status, err := w.Write(
				[]byte(
					fmt.Sprintf("Wrong type of metric: %s", metricType),
				),
			)
			if err != nil {
				panic(fmt.Sprintf("%s: %v", err.Error(), status))
			}
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		status, err := w.Write(
			[]byte(
				fmt.Sprintf("Others methods are not allowed. Have got: %s", request.Method),
			),
		)
		if err != nil {
			panic(fmt.Sprintf("%s: %v", err.Error(), status))
		}
	}
}

func run() error {
	h := http.NewServeMux()
	h.Handle("/update/", http.HandlerFunc(Update))
	return http.ListenAndServe(`:8080`, h)
}
