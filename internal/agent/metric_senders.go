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

func (r *MetricSender) PollIntervalCatch(
	donePollInterval chan bool,
) {
	time.Sleep(time.Duration(*r.PollIntervalTime) * time.Second)
	donePollInterval <- true
}

func (r *MetricSender) ReportIntervalCatch(
	doneReportInterval chan bool,
) {
	time.Sleep(time.Duration(*r.ReportIntervalTime) * time.Second)
	doneReportInterval <- true
}

func (r *MetricSender) PollInterval(isTestMode bool) {
	memInfo := runtime.MemStats{}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	donePollInterval := make(chan bool)
	doneReportInterval := make(chan bool)
	countOfUpdate := 0
	go r.PollIntervalCatch(donePollInterval)
	go r.ReportIntervalCatch(doneReportInterval)
	for {
		select {
		case <-donePollInterval:
			runtime.ReadMemStats(&memInfo)
			countOfUpdate += 1
			go r.PollIntervalCatch(donePollInterval)
			fmt.Println("Done PollInterval!")
		case <-doneReportInterval:
			r.ReportInterval(&memInfo, countOfUpdate)
			countOfUpdate = 0
			go r.ReportIntervalCatch(doneReportInterval)
			fmt.Println("Done ReportInterval!")
			if isTestMode {
				return
			}
		case t := <-ticker.C:
			fmt.Println("Current time: ", t)
		}
	}
}

func (r *MetricSender) SendMetric(value interface{}, metricType string, metricName string) int {
	url := fmt.Sprintf("%s/update/%s/%s/%v", r.URL, metricType, metricName, value)
	fmt.Println("Sending post request with url: " + url)
	resp, err := r.Client.R().ForceContentType("text/plain").SetBody("").Post(
		url,
	)
	if err != nil {
		fmt.Printf("Request cannot be precossed, something is wrong: %s\n", err.Error())
		return -2
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		fmt.Printf("Something wrong with resp - '%v', status code - %v\n", resp, resp.StatusCode())
		return -1
	}
	return 0
}

func (r *MetricSender) ReportInterval(a *runtime.MemStats, countOfUpdate int) {
	// Cast to map our all got metrics
	b, _ := json.Marshal(a)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	var code int
	for _, v := range *r.ListMetrics {
		code = r.SendMetric(
			m[v],
			GAUGE,
			v,
		)
		if code != 0 {
			return
		}
	}
	code = r.SendMetric(countOfUpdate, COUNTER, "PollCount")
	if code != 0 {
		return
	}
	code = r.SendMetric(rand.Float64(), GAUGE, "RandomValue")
	if code != 0 {
		return
	}
	fmt.Println("All metrics successfully sent")
}
