package server

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/akashipov/MetricCollector/internal/general"
)

type MemStorage struct {
	MetricList map[string]*general.Metrics `json:"metrics"`
}

func (r *MemStorage) Get(metricName string, request *http.Request) *general.Metrics {
	if (PsqlInfo == nil) || (*PsqlInfo == "") {
		val, ok := r.MetricList[metricName]
		if ok {
			return val
		}
		return nil
	} else {
		fmt.Println("Get db... ", DB)
		row := DB.QueryRowContext(request.Context(), "SELECT * FROM metrics WHERE id = $1", metricName)
		var metric general.Metrics
		var v sql.NullFloat64
		var delta sql.NullInt64
		err := row.Scan(&metric.ID, &metric.MType, &v, &delta)
		if v.Valid {
			metric.Value = &v.Float64
			metric.Delta = nil
		}
		if delta.Valid {
			metric.Delta = &delta.Int64
			metric.Value = nil
		}
		if err != nil {
			return nil
		}
		return &metric
	}
}

func (r *MemStorage) Record(name string, value *general.Metrics, request *http.Request) {
	if (PsqlInfo == nil) || (*PsqlInfo == "") {
		MapMetric.MetricList[name] = value
	} else {
		var v sql.NullFloat64
		var delta sql.NullInt64
		if value.Delta != nil {
			delta.Int64 = *value.Delta
			delta.Valid = true
		} else {
			delta.Int64 = 0
			delta.Valid = false
		}
		if value.Value != nil {
			v.Float64 = *value.Value
			v.Valid = true
		} else {
			v.Float64 = 0.0
			v.Valid = false
		}
		fmt.Println(value.ID, value.MType, v, delta, DB, request.Context())
		_, err := DB.ExecContext(
			request.Context(),
			"INSERT INTO metrics VALUES($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET mtype = $2, value = $3, delta = $4;", value.ID, value.MType, v, delta,
		)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func (r *MemStorage) String() string {
	s := ""
	for _, v := range r.MetricList {
		if v.Delta != nil {
			s += fmt.Sprintf("key: %s -> value: %d\n", v.ID, *v.Delta)
		} else {
			s += fmt.Sprintf("key: %s -> value: %f\n", v.ID, *v.Value)
		}
	}
	return s
}

func NewStorage(vMap map[string]*general.Metrics) *MemStorage {
	if vMap == nil {
		return &MemStorage{make(map[string]*general.Metrics, 0)}
	}
	return &MemStorage{vMap}
}

var MapMetric = NewStorage(nil)
