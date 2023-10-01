package internal

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
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
}

func (r MetricSender) PollInterval(isTestMode bool) {
	fmt.Printf(BaseDirEnv + " dirname -> '" + os.Getenv(BaseDirEnv) + "'\n")
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

func (r MetricSender) SendMetric(value interface{}, metricType string, metricName string) error {
	url := fmt.Sprintf("%s/update/%s/%s/%v", r.URL, metricType, metricName, value)
	fmt.Println("Sending post request with url: " + url)
	resp, err := http.Post(
		url,
		"text/plain",
		strings.NewReader(""),
	)
	if err != nil {
		fmt.Println(fmt.Sprintf("Request cannot be precossed: %s", err.Error()))
		panic(err.Error())
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		fmt.Printf("Something wrong with '%v': status code is %v\n", resp.Body, resp.StatusCode)
	}
	return nil
}

func (r MetricSender) ReportInterval(a *runtime.MemStats, countOfUpdate int) {
	// Cast to map our all got metrics
	b, _ := json.Marshal(a)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)

	for _, v := range *r.ListMetrics {
		r.SendMetric(
			m[v],
			GAUGE,
			v,
		)
	}
	r.SendMetric(countOfUpdate, COUNTER, "PollCount")
	r.SendMetric(rand.Float64(), GAUGE, "RandomValue")
	fmt.Println(fmt.Sprintf("All metrics successfully sent"))
}
