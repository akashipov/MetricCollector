package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"syscall"

	"github.com/akashipov/MetricCollector/internal/general"
)

type Storage interface {
	Get(metricName string, request *http.Request) *general.Metrics
	GetAll() map[string]*general.Metrics
	Record(
		value *general.Metrics, request *http.Request,
		tx *sql.Tx,
	)
	Clean() error
}

type PsqlStorage struct {
	PsqlInfo *string
}

func (r *PsqlStorage) Get(metricName string, request *http.Request) *general.Metrics {
	if (r.PsqlInfo == nil) || (*r.PsqlInfo == "") {
		fmt.Printf("Wrong settings for class PsqlInfo: '%s'\n", *r.PsqlInfo)
		return nil
	}
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

func (r *PsqlStorage) GetAll() map[string]*general.Metrics {
	if (r.PsqlInfo == nil) || (*r.PsqlInfo == "") {
		fmt.Printf("Wrong settings for class PsqlInfo: '%s'\n", *r.PsqlInfo)
		return nil
	}
	rows, err := DB.QueryContext(context.Background(), "SELECT * FROM metrics")
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	defer rows.Close()
	metrics := make(map[string]*general.Metrics)
	var rErr error
	for rows.Next() {
		var metric general.Metrics
		var v sql.NullFloat64
		var delta sql.NullInt64
		err := rows.Scan(&metric.ID, &metric.MType, &v, &delta)
		if err != nil {
			rErr = errors.Join(rErr, err)
		}
		if v.Valid {
			metric.Value = &v.Float64
			metric.Delta = nil
		}
		if delta.Valid {
			metric.Delta = &delta.Int64
			metric.Value = nil
		}
		metrics[metric.ID] = &metric
	}
	if rErr != nil {
		fmt.Println(rErr.Error())
		return nil
	}
	return metrics
}

func (r *PsqlStorage) Clean() error {
	if (r.PsqlInfo == nil) || (*r.PsqlInfo == "") {
		return fmt.Errorf("wrong settings for class PsqlInfo: '%s'", *r.PsqlInfo)
	}
	_, err := DB.ExecContext(context.Background(), "TRUNCATE TABLE metrics")
	if err != nil {
		return err
	}
	return nil
}

func (r *PsqlStorage) Record(
	value *general.Metrics, request *http.Request,
	tx *sql.Tx,
) {
	if (r.PsqlInfo == nil) || (*r.PsqlInfo == "") {
		fmt.Printf("Wrong settings for class PsqlInfo: '%s'\n", *r.PsqlInfo)
		return
	}
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
	query := "INSERT INTO metrics VALUES($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET mtype = $2, value = $3, delta = $4;"
	if tx != nil {
		f := func() error {
			_, err := tx.ExecContext(
				request.Context(),
				query, value.ID, value.MType, v, delta,
			)
			return err
		}
		err := general.RetryCode(f, syscall.ECONNREFUSED)
		if err != nil {
			tx.Rollback()
		}
	} else {
		f := func() error {
			_, err := DB.ExecContext(
				request.Context(),
				query, value.ID, value.MType, v, delta,
			)
			return err
		}
		err := general.RetryCode(f, syscall.ECONNREFUSED)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

type MemStorage struct {
	MetricList map[string]*general.Metrics `json:"metrics"`
}

func (r *MemStorage) Get(metricName string, request *http.Request) *general.Metrics {
	val, ok := r.MetricList[metricName]
	if ok {
		return val
	}
	return nil
}

func (r *MemStorage) GetAll() map[string]*general.Metrics {
	return r.MetricList
}

func (r *MemStorage) Clean() error {
	r.MetricList = make(map[string]*general.Metrics)
	return nil
}

func (r *MemStorage) Record(
	value *general.Metrics, request *http.Request,
	tx *sql.Tx,
) {
	r.MetricList[value.ID] = value
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

func NewStorage(vMap map[string]*general.Metrics) Storage {
	if (PsqlInfo == nil) || (*PsqlInfo == "") {
		fmt.Println("Is used memory local storage")
		if vMap == nil {
			return &MemStorage{make(map[string]*general.Metrics, 0)}
		}
		return &MemStorage{vMap}
	} else {
		fmt.Println("Is used psql storage")
		return &PsqlStorage{PsqlInfo: PsqlInfo}
	}
}

var MapMetric = NewStorage(nil)
