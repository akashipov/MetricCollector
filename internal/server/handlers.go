package server

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/akashipov/MetricCollector/internal/agent"
	"github.com/akashipov/MetricCollector/internal/general"
	"github.com/akashipov/MetricCollector/internal/server/logger"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

func ServerRouter(s *zap.SugaredLogger) http.Handler {
	r := chi.NewRouter()
	r.Get("/ping", logger.WithLogging(http.HandlerFunc(TestConnection), s))
	r.Get("/", logger.WithLogging(http.HandlerFunc(MainPage), s))
	r.Post("/updates/", logger.WithLogging(http.HandlerFunc(Updates), s))
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
	return GzipHandle(r)
}

func Update(w http.ResponseWriter, request *http.Request) {
	MetricType := chi.URLParam(request, "MetricType")
	MetricName := chi.URLParam(request, "MetricName")
	MetricValue := chi.URLParam(request, "MetricValue")
	m, err := ValidateMetric(w, MetricType, MetricValue, MetricName)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if m == nil {
		return
	}
	SaveMetric(w, m, request, nil)
}

func SaveMetric(
	w http.ResponseWriter, metric *general.Metrics, request *http.Request,
	tx *sql.Tx,
) error {
	val := MapMetric.Get(metric.ID, request)
	if val != nil {
		switch metric.MType {
		case agent.COUNTER:
			if val.Delta == nil {
				val.Delta = metric.Delta
			} else {
				*val.Delta += *metric.Delta
			}
			val.MType = agent.COUNTER
			val.Value = nil
		case agent.GAUGE:
			val.Value = metric.Value
			val.MType = agent.GAUGE
			val.Delta = nil
		}
	} else {
		val = metric
	}
	fmt.Println("Have got")
	fmt.Println(val.ID, val.MType)
	MapMetric.Record(val, request, tx)
	metric.Delta = val.Delta
	metric.Value = val.Value
	return nil
}

func ProcessMetric(
	w http.ResponseWriter, request *http.Request, metric *general.Metrics,
	tx *sql.Tx,
) error {
	if metric == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Was passed wrong nil value like metric"))
		return nil
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
		w.WriteHeader(http.StatusNotFound)
		err := fmt.Errorf("wrong type of metric: '%s'", MetricType)
		w.Write([]byte(err.Error()))
		return err
	}
	m, err := ValidateMetric(w, MetricType, MetricValue, MetricName)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return err
	}
	if m == nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	err = SaveMetric(w, m, request, tx)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return err
	}
	return nil
}

func SaveMetrics(w http.ResponseWriter, request *http.Request, metrics []general.Metrics) {
	results := make([]general.Metrics, 0)
	var tx *sql.Tx
	var err error
	if !((PsqlInfo == nil) || (*PsqlInfo == "")) {
		tx, err = DB.Begin()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	}
	for _, metric := range metrics {
		err := ProcessMetric(w, request, &metric, tx)
		if err != nil {
			if !((PsqlInfo == nil) || (*PsqlInfo == "")) {
				err = tx.Rollback()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}
			}
			return
		}
	}
	if !((PsqlInfo == nil) || (*PsqlInfo == "")) {
		err := tx.Commit()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}
	for _, metric := range metrics {
		val := MapMetric.Get(metric.ID, request)
		if val != nil {
			results = append(results, *val)
		}
	}
	var jsonEncoded []byte
	if len(results) == 1 {
		jsonEncoded, err = json.Marshal(results[0])
	} else {
		jsonEncoded, err = json.Marshal(results)
	}
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonEncoded)
}

func Updates(w http.ResponseWriter, request *http.Request) {
	fmt.Println("Updates block is ran")
	var buf bytes.Buffer
	var metrics []general.Metrics
	_, err := buf.ReadFrom(request.Body)
	defer request.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	data := buf.Bytes()
	data, err = Decode(w, request, data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Unmarshal problem: %s", err.Error())))
		return
	}
	SaveMetrics(w, request, metrics)
}

func UpdateShortForm(w http.ResponseWriter, request *http.Request) {
	var buf bytes.Buffer
	var metric general.Metrics
	_, err := buf.ReadFrom(request.Body)
	defer request.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	data := buf.Bytes()
	data, err = Decode(w, request, data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	err = json.Unmarshal(data, &metric)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Unmarshal problem: %s", err.Error())))
		return
	}
	SaveMetrics(w, request, []general.Metrics{metric})
}

func СheckContentType(w http.ResponseWriter, request *http.Request, pattern string) error {
	contentType := request.Header.Get("Content-Type")
	fmt.Printf("Content-type has been got: '%s'\n", contentType)
	if !strings.Contains(contentType, pattern) {
		w.WriteHeader(http.StatusBadRequest)
		msg := "bad type of content-type, please change it"
		w.Write([]byte(msg))
		return errors.New(msg)
	}
	return nil
}

func MainPage(w http.ResponseWriter, request *http.Request) {
	ul := "<ul>"
	for _, k := range MapMetric.MetricList {
		if k.MType == agent.GAUGE {
			ul += fmt.Sprintf("<li>%v: %v</li>", k.ID, *k.Value)
		}
		if k.MType == agent.COUNTER {
			ul += fmt.Sprintf("<li>%v: %d</li>", k.ID, *k.Delta)
		}
	}
	ul += "</ul>"
	html := fmt.Sprintf("<html>%s</html>", ul)
	w.Header().Set("Content-Type", "text/html")
	status, err := w.Write([]byte(html))
	if err != nil {
		fmt.Printf("%v: %v", status, err.Error())
	}
}

func TestConnection(w http.ResponseWriter, request *http.Request) {
	TestConnectionPostgres(w, request)
}

func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fmt.Printf("Skip gzip... content-type = '%s'\n", r.Header.Get("Content-Type"))
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		next.ServeHTTP(general.GzipWriter{OldW: w, Writer: gz}, r)
	})
}

func Decode(w http.ResponseWriter, request *http.Request, data []byte) ([]byte, error) {
	if strings.Contains(request.Header.Get("Content-Encoding"), "gzip") {
		reader := bytes.NewReader(data)
		gzreader, err := gzip.NewReader(reader)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Println(err.Error())
			return nil, err
		}
		data, err = io.ReadAll(gzreader)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Println(err.Error())
			return nil, err
		}
	}
	return data, nil
}

func GetMetricShortForm(w http.ResponseWriter, request *http.Request) {
	err := СheckContentType(w, request, "application/json")
	if err != nil {
		return
	}

	var buf bytes.Buffer
	var metric general.Metrics
	_, err = buf.ReadFrom(request.Body)
	defer request.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err.Error())
		return
	}
	data := buf.Bytes()
	data, err = Decode(w, request, data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err.Error())
		return
	}
	json.Unmarshal(data, &metric)
	MetricName := metric.ID
	MetricType := metric.MType
	var answer []byte
	val := MapMetric.Get(MetricName, request)
	if ok := val != nil; ok {
		if val.MType == MetricType {
			switch MetricType {
			case agent.COUNTER:
				metric.Delta = val.Delta
				answer, err = json.Marshal(metric)
				if err != nil {
					answer = []byte(err.Error())
				}
			case agent.GAUGE:
				metric.Value = val.Value
				answer, err = json.Marshal(metric)
				if err != nil {
					answer = []byte(err.Error())
				}
			default:
				answer = []byte("wrong type of metric")
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			answer = []byte(fmt.Sprintf("It has other metric type: '%s'", val.MType))
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		answer = []byte(fmt.Sprintf("There is no metric like this: '%v'", MetricName))
	}
	w.Header().Set("Content-Type", "application/json")
	status, err := w.Write(answer)
	if err != nil {
		fmt.Printf("%v: %v", status, err.Error())
	}
}

func GetMetric(w http.ResponseWriter, request *http.Request) {
	MetricName := chi.URLParam(request, "MetricName")
	MetricType := chi.URLParam(request, "MetricType")
	var answer string
	val := MapMetric.Get(MetricName, request)
	if ok := val != nil; ok {
		if val.MType == MetricType {
			w.Header().Set("Content-Type", "text/html")
			if val.MType == agent.GAUGE {
				answer = fmt.Sprintf("%v", *val.Value)
			} else {
				answer = fmt.Sprintf("%d", *val.Delta)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			answer = fmt.Sprintf("It has other metric type: '%s'", val.MType)
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
