package internal

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var BaseDirEnv = "BASE_DIR"

func SaveMetric(w http.ResponseWriter, metric Metric, metricName string) {
	val, ok := MapMetric.m[metricName]
	if ok {
		(*val).Update(metric.GetValue())
	} else {
		if MapMetric.m == nil {
			m := make(map[string]*Metric)
			m[metricName] = &metric
			MapMetric.m = m
		} else {
			MapMetric.m[metricName] = &metric
		}
	}
	fPath := filepath.Join(os.Getenv(BaseDirEnv), "map.txt")
	_, err := os.Create(fPath)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(fPath, []byte(MapMetric.String()), 0644)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusOK)
	status, err := w.Write([]byte(fmt.Sprintf("updated mapMetric: %v", MapMetric)))
	if err != nil {
		panic(fmt.Sprintf("%s: %v", err.Error(), status))
	}
}

func Update(w http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodPost {
		//contentType := request.Header.Get("Content-Type")
		elements := strings.Split(request.URL.Path, "/")
		//if contentType != "text/plain" {
		//	w.WriteHeader(http.StatusNotFound)
		//	status, err := w.Write(
		//		[]byte("Request doesn't contain correct Content-Type"),
		//	)
		//	if err != nil {
		//		panic(fmt.Sprintf("%s: %v", err.Error(), status))
		//	}
		//	return
		//}
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
		badTypeValueMsg := "Bad type of value passed. Please be sure that it can be converted to "
		if metricType == GAUGE {
			n, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				status, err := w.Write([]byte(badTypeValueMsg + "float64"))
				if err != nil {
					panic(fmt.Sprintf("%s: %v", err.Error(), status))
				}
				return
			}
			SaveMetric(w, NewGauge(&n), metricName)
			return
		} else if metricType == COUNTER {
			n, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				status, err := w.Write([]byte(badTypeValueMsg + "int64"))
				if err != nil {
					panic(fmt.Sprintf("%s: %v", err.Error(), status))
				}
				return
			}
			SaveMetric(w, NewCounter(&n), metricName)
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
