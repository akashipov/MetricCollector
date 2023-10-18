package agent

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
	URL                string
	ListMetrics        *[]string
	Client             *resty.Client
	ReportIntervalTime *int
	PollIntervalTime   *int
}

func (r *MetricSender) PollInterval(isTestMode bool) {
	memInfo := runtime.MemStats{}
	countOfUpdate := 0
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
	url := fmt.Sprintf("%s/update/%s/%s/%v", r.URL, metricType, metricName, value)
	fmt.Println("Sending post request with url: " + url)
	resp, err := r.Client.R().ForceContentType("text/plain").SetBody("").Post(
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
			return
		}
	}
	err = r.SendMetric(countOfUpdate, COUNTER, "PollCount")
	if err != nil {
		return
	}
	err = r.SendMetric(rand.Float64(), GAUGE, "RandomValue")
	if err != nil {
		return
	}
	fmt.Println("All metrics successfully sent")
}
