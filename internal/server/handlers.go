package server

import (
	"bytes"
	"compress/gzip"
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
	r.Get("/", logger.WithLogging(http.HandlerFunc(MainPage), s))
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
	SaveMetric(w, m)
}

func SaveMetric(w http.ResponseWriter, metric *general.Metrics) error {
	val := MapMetric.Get(metric.ID)
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
		MapMetric.MetricList[metric.ID] = metric
	}
	return nil
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
		w.Write([]byte(fmt.Sprintf("Wrong type of metric: '%s'", MetricType)))
		return
	}
	m, err := ValidateMetric(w, MetricType, MetricValue, MetricName)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	if m == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = SaveMetric(w, m)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	val := MapMetric.Get(MetricName)
	switch MetricType {
	case agent.COUNTER:
		if val != nil {
			metric.Delta = val.Delta
		}
	case agent.GAUGE:
		if val != nil {
			metric.Value = val.Value
		}
	}
	b, err := json.Marshal(metric)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func СheckContentType(w http.ResponseWriter, request *http.Request, pattern string) error {
	contentType := request.Header.Get("Content-Type")
	fmt.Printf("Content-type has been got: '%s'\n", contentType)
	if !strings.Contains(contentType, pattern) {
		w.WriteHeader(http.StatusBadRequest)
		msg := "Bad type of content-type, please change it"
		w.Write([]byte(msg))
		return errors.New(msg)
	}
	return nil
}

func MainPage(w http.ResponseWriter, request *http.Request) {
	err := СheckContentType(w, request, "text/plain")
	if err != nil {
		return
	}
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

func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fmt.Println("Skip gzip... content-type =", r.Header.Get("Content-Type"))
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
	val := MapMetric.Get(MetricName)
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
				answer = []byte("Wrong type of metric")
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
	err := СheckContentType(w, request, "text/plain")
	if err != nil {
		return
	}
	MetricName := chi.URLParam(request, "MetricName")
	MetricType := chi.URLParam(request, "MetricType")
	var answer string
	val := MapMetric.Get(MetricName)
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
