package agent

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

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
	url := fmt.Sprintf("%s/update/", r.URL)
	fmt.Println("Sending post request with url: " + url)
	var s string
	switch metricType {
	case COUNTER:
		s = fmt.Sprintf("{\"id\":\"%s\",\"type\":\"%s\",\"delta\":%d}", metricName, metricType, value)
	case GAUGE:
		s = fmt.Sprintf("{\"id\":\"%s\",\"type\":\"%s\",\"value\":%f}", metricName, metricType, value)
	default:
		fmt.Printf("Wrong type of metric: %v\n", metricType)
		return nil
	}
	req := r.Client.R().SetBody(s).SetHeader("Content-Type", "application/json")
	if *AgentKey != "" {
		encoder := hmac.New(sha256.New, []byte(*AgentKey))
		encoder.Write([]byte(s))
		v := encoder.Sum(nil)
		req.SetHeader("HashSHA256", base64.RawURLEncoding.EncodeToString(v[:]))
	}
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
