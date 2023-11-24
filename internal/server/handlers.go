package server

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
	r.Get("/", MainPage)
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
	return HashHandle(GzipHandle(r))
}

func Update(w http.ResponseWriter, request *http.Request) {
	MetricType := chi.URLParam(request, "MetricType")
	MetricName := chi.URLParam(request, "MetricName")
	MetricValue := chi.URLParam(request, "MetricValue")
	m, err := ValidateMetric(&w, MetricType, MetricValue, MetricName)
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
	w.WriteHeader(http.StatusOK)
	return nil
}

func UpdateShortForm(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	var metric general.Metrics
	_, err := buf.ReadFrom(request.Body)
	defer request.Body.Close()
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	data := buf.Bytes()
	data, err = Decode(&w, request, data)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
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
	m, err := ValidateMetric(&w, MetricType, MetricValue, MetricName)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if m == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = SaveMetric(w, m)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err.Error())
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
	w.WriteHeader(http.StatusOK)
	b, err := json.Marshal(metric)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Println(err.Error())
		return
	}
	w.Write(b)
}

func MainPage(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "text/html")
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
	w.WriteHeader(http.StatusOK)
	status, err := w.Write([]byte(html))
	if err != nil {
		fmt.Printf("%v: %v", status, err.Error())
	}
}

func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		// contentType := w.Header().Get("Content-Type")
		// fmt.Printf("Content-Type of response: '%s'\n", contentType)
		// if !strings.Contains(contentType, "application/json") &&
		// 	!strings.Contains(contentType, "text/html") {
		// 	next.ServeHTTP(w, r)
		// 	return
		// }

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(general.GzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func HashHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *ServerKey != "" {
			v := r.Header.Get("HashSHA256")
			if v != "" {
				var buf bytes.Buffer
				_, err := buf.ReadFrom(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("Cannot read body bytes"))
					return
				}
				decoder := hmac.New(sha256.New, []byte(*ServerKey))
				decoder.Write(buf.Bytes())
				encoded := decoder.Sum(nil)
				if base64.RawURLEncoding.EncodeToString(encoded) == v {
					fmt.Println("Sign checking have been passed")
					r.Body.Close() //  must close
					r.Body = io.NopCloser(bytes.NewBuffer(buf.Bytes()))
					next.ServeHTTP(w, r)
					return
				} else {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("Sign checking haven't been passed"))
					return
				}
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("There is no HashSHA256 to check sign"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func Decode(w *http.ResponseWriter, request *http.Request, data []byte) ([]byte, error) {
	if strings.Contains(request.Header.Get("Content-Encoding"), "gzip") {
		reader := bytes.NewReader(data)
		gzreader, err := gzip.NewReader(reader)
		if err != nil {
			(*w).WriteHeader(http.StatusBadGateway)
			fmt.Println(err.Error())
			return nil, err
		}
		data, err = io.ReadAll(gzreader)
		if err != nil {
			(*w).WriteHeader(http.StatusBadGateway)
			fmt.Println(err.Error())
			return nil, err
		}
	}
	return data, nil
}

func GetMetricShortForm(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var buf bytes.Buffer
	var metric general.Metrics
	_, err := buf.ReadFrom(request.Body)
	defer request.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err.Error())
		return
	}
	data := buf.Bytes()
	data, err = Decode(&w, request, data)
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
			w.WriteHeader(http.StatusOK)
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
	status, err := w.Write(answer)
	if err != nil {
		fmt.Printf("%v: %v", status, err.Error())
	}
}

func GetMetric(w http.ResponseWriter, request *http.Request) {
	MetricName := chi.URLParam(request, "MetricName")
	MetricType := chi.URLParam(request, "MetricType")
	var answer string
	val := MapMetric.Get(MetricName)
	if ok := val != nil; ok {
		if val.MType == MetricType {
			w.WriteHeader(http.StatusOK)
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
