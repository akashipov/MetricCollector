package agent

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/akashipov/MetricCollector/internal/general"
	"github.com/go-resty/resty/v2"
)

var ListMetrics = []string{
	"Alloc",
	"BuckHashSys",
	"Frees",
	"GCCPUFraction",
	"GCSys",
	"HeapAlloc",
	"HeapIdle",
	"HeapInuse",
	"HeapObjects",
	"HeapReleased",
	"HeapSys",
	"LastGC",
	"Lookups",
	"MCacheInuse",
	"MCacheSys",
	"MSpanInuse",
	"MSpanSys",
	"Mallocs",
	"NextGC",
	"NumForcedGC",
	"NumGC",
	"OtherSys",
	"PauseTotalNs",
	"StackInuse",
	"StackSys",
	"Sys",
	"TotalAlloc",
}

type MetricSenderInterface interface {
	pollInterval(bool)
	reportInterval()
}

const COUNTER string = "counter"
const GAUGE string = "gauge"

type MetricSender struct {
	URL                string
	ListMetrics        *[]string
	Client             *resty.Client
	ReportIntervalTime *int
	PollIntervalTime   *int
}

func (r *MetricSender) PollInterval(isTestMode bool) {
	memInfo := runtime.MemStats{}
	countOfUpdate := int64(0)
	tickerPollInterval := time.NewTicker(time.Duration(*r.PollIntervalTime) * time.Second)
	defer tickerPollInterval.Stop()
	tickerReportInterval := time.NewTicker(time.Duration(*r.ReportIntervalTime) * time.Second)
	defer tickerReportInterval.Stop()
	for {
		select {
		case <-tickerPollInterval.C:
			runtime.ReadMemStats(&memInfo)
			countOfUpdate += 1
			fmt.Println("Done PollInterval!")
		case <-tickerReportInterval.C:
			r.ReportInterval(&memInfo, countOfUpdate)
			countOfUpdate = 0
			fmt.Println("Done ReportInterval!")
			if isTestMode {
				return
			}
		}
	}
}

func (r *MetricSender) SendMetric(value interface{}, metricType string, metricName string) error {
	url := fmt.Sprintf("%s/update/", r.URL)
	fmt.Println("Sending post request with url: " + url)
	var s string
	switch metricType {
	case COUNTER:
		s = fmt.Sprintf("{\"id\":\"%s\",\"type\":\"%s\",\"delta\":%d}", metricName, metricType, value)
	case GAUGE:
		s = fmt.Sprintf("{\"id\":\"%s\",\"type\":\"%s\",\"value\":%f}", metricName, metricType, value)
	default:
		return fmt.Errorf("wrong type of metric: %v\n", metricType)
	}
	req := r.Client.R().SetBody(s).SetHeader("Content-Type", "application/json")
	resp, err := req.Post(
		url,
	)
	if err != nil {
		fmt.Printf("Request cannot be precossed, something is wrong: %s\n", err.Error())
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		fmt.Printf("Something wrong with resp - '%v', status code - %v\n", resp, resp.StatusCode())
		return err
	}
	fmt.Printf("Success: %v\n", resp.StatusCode())
	return nil
}

func (r *MetricSender) SendMetrics(metrics general.SeveralMetrics) error {
	url := fmt.Sprintf("%s/updates/", r.URL)
	fmt.Println("Sending post request with url: " + url)
	s, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	req := r.Client.R().SetBody(s).SetHeader("Content-Type", "application/json")
	resp, err := req.Post(
		url,
	)
	if err != nil {
		fmt.Printf("Request cannot be precossed, something is wrong: %s\n", err.Error())
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("something wrong with resp - '%v', status code - %v", resp, resp.StatusCode())
	}
	fmt.Printf("Success: %v\n", resp.StatusCode())
	return nil
}

func (r *MetricSender) ReportInterval(a *runtime.MemStats, countOfUpdate int64) {
	// Cast to map our all got metrics
	b, _ := json.Marshal(a)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	var metrics general.SeveralMetrics
	metrics.Mtrcs = make([]general.Metrics, 0)
	for _, v := range *r.ListMetrics {
		casted, ok := m[v].(float64)
		if ok {
			metrics.Mtrcs = append(
				metrics.Mtrcs,
				general.Metrics{ID: v, MType: GAUGE, Value: &casted},
			)
		} else {
			fmt.Println("Cannot be cast to float64, some wrong type")
		}
	}
	metrics.Mtrcs = append(
		metrics.Mtrcs,
		general.Metrics{ID: "PollCount", MType: COUNTER, Delta: &countOfUpdate},
	)
	c := rand.Float64()
	metrics.Mtrcs = append(
		metrics.Mtrcs,
		general.Metrics{ID: "RandomValue", MType: GAUGE, Value: &c},
	)
	err := r.SendMetrics(metrics)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("All metrics successfully sent")
}
