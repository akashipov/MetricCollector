package server

import (
	"fmt"
	"github.com/akashipov/MetricCollector/internal/agent"
	"net/http"
	"strconv"
)

func ValidateMetric(w *http.ResponseWriter, MetricType string, MetricValue string) (Metric, error) {
	badTypeValueMsg := "Bad type of value passed. Please be sure that it can be converted to "
	errFMT := "error: %s, status: %d"
	var err error
	var status int
	if MetricType == agent.GAUGE {
		var n float64
		n, err = strconv.ParseFloat(MetricValue, 64)
		if err != nil {
			(*w).WriteHeader(http.StatusBadRequest)
			status, err = (*w).Write([]byte(badTypeValueMsg + fmt.Sprintf("float64: '%v'", MetricValue)))
		} else {
			return NewGauge(n), nil
		}
	} else if MetricType == agent.COUNTER {
		var n int64
		n, err = strconv.ParseInt(MetricValue, 10, 64)
		if err != nil {
			(*w).WriteHeader(http.StatusBadRequest)
			status, err = (*w).Write([]byte(badTypeValueMsg + fmt.Sprintf("int64: '%v'", MetricValue)))
		} else {
			return NewCounter(n), nil
		}
	} else {
		(*w).WriteHeader(http.StatusBadRequest)
		status, err = (*w).Write(
			[]byte(
				fmt.Sprintf("Wrong type of metric: '%s'", MetricType),
			),
		)
	}
	if err != nil {
		return nil, fmt.Errorf(errFMT, err.Error(), status)
	}
	return nil, nil
}
