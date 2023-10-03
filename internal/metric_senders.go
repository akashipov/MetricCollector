package internal

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"math/rand"
	"net/http"
	"runtime"
	"time"
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
	URL         string
	ListMetrics *[]string
	client      *resty.Client
}

func (r *MetricSender) PollInterval(isTestMode bool) {
	a := runtime.MemStats{}
	countOfUpdate := 0
	for i := 0; i >= 0; i++ {
		runtime.ReadMemStats(&a)
		countOfUpdate += 1
		time.Sleep(2 * time.Second)
		if i%5 == 0 {
			r.ReportInterval(&a, countOfUpdate)
			countOfUpdate = 0
		}
		if isTestMode {
			break
		}
	}
}

func (r *MetricSender) SendMetric(value interface{}, metricType string, metricName string) error {
	url := fmt.Sprintf("%s/update/%s/%s/%v", r.URL, metricType, metricName, value)
	fmt.Println("Sending post request with url: " + url)
	resp, err := r.client.R().ForceContentType("text/plain").SetBody("").Post(
		url,
	)
	if err != nil {
		fmt.Printf("Request cannot be precossed: %s\n", err.Error())
		return err
	}
	status := resp.StatusCode()
	if status != http.StatusOK && status != http.StatusCreated {
		fmt.Printf("Something wrong with '%v': status code is %v\n", resp, resp.StatusCode())
	}
	return nil
}

func (r *MetricSender) ReportInterval(a *runtime.MemStats, countOfUpdate int) {
	// Cast to map our all got metrics
	b, _ := json.Marshal(a)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	var err error
	for _, v := range *r.ListMetrics {
		err = r.SendMetric(
			m[v],
			GAUGE,
			v,
		)
		if err != nil {
			panic(err)
		}
	}
	err = r.SendMetric(countOfUpdate, COUNTER, "PollCount")
	if err != nil {
		panic(err)
	}
	err = r.SendMetric(rand.Float64(), GAUGE, "RandomValue")
	if err != nil {
		panic(err)
	}
	fmt.Println("All metrics successfully sent")
}
