package server

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/akashipov/MetricCollector/internal/agent"
	"github.com/akashipov/MetricCollector/internal/general"
)

func ValidateMetric(
	w http.ResponseWriter, MetricType string,
	MetricValue interface{},
	MetricName string,
) (*general.Metrics, error) {
	badTypeValueMsg := "Bad type of value passed. Please be sure that it can be converted to "
	errFMT := "error: %s, status: %d"
	var err error
	err = nil
	var status int
	if MetricType == agent.GAUGE {
		var n float64
		if reflect.TypeOf(MetricValue).String() == "string" {
			n, err = strconv.ParseFloat(MetricValue.(string), 64)
			if err == nil {
				return &general.Metrics{ID: MetricName, MType: MetricType, Value: &n}, nil
			}
		}
		n, ok := MetricValue.(float64)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			status, err = w.Write([]byte(badTypeValueMsg + fmt.Sprintf("float64: '%v'", MetricValue)))
		} else {
			return &general.Metrics{ID: MetricName, MType: MetricType, Value: &n}, nil
		}
	} else if MetricType == agent.COUNTER {
		var n int64
		if reflect.TypeOf(MetricValue).String() == "string" {
			n, err = strconv.ParseInt(MetricValue.(string), 10, 64)
			if err == nil {
				return &general.Metrics{ID: MetricName, MType: MetricType, Delta: &n}, nil
			}
		}
		n, ok := MetricValue.(int64)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			status, err = w.Write([]byte(badTypeValueMsg + fmt.Sprintf("int64: '%v'", MetricValue)))
		} else {
			return &general.Metrics{ID: MetricName, MType: MetricType, Delta: &n}, nil
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		status, err = w.Write(
			[]byte(
				fmt.Sprintf("wrong type of metric: '%s'", MetricType),
			),
		)
	}
	if err != nil {
		return nil, fmt.Errorf(errFMT, err.Error(), status)
	}
	return nil, nil
}
