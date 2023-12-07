package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

	"github.com/akashipov/MetricCollector/internal/general"
	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v3/mem"
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
	"TotalMemory",
	"FreeMemory",
	"CPUutilization1",
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
	Done               chan bool
	M                  *sync.Mutex
	GoPull             chan struct{}
	WG                 *sync.WaitGroup
}

func (r *MetricSender) Run() {
	memInfo := make(map[string]interface{})
	var countOfUpdate atomic.Int64
	r.WG.Add(1)
	go r.TickerWithSignal()
	r.WG.Add(1)
	go func() {
		fmt.Println("Has been started PollInterval")
		r.PollInterval(&memInfo, &countOfUpdate)
		r.WG.Done()
	}()
	r.WG.Add(1)
	go func() {
		fmt.Println("Has been started Additional metrics collecting")
		r.AddInterval(&memInfo, &countOfUpdate)
		r.WG.Done()
	}()
	r.WG.Add(1)
	go func() {
		fmt.Println("Has been started ReportInterval")
		r.ReportInterval(&memInfo, &countOfUpdate)
		r.WG.Done()
	}()
	r.WG.Done()
}

func (r *MetricSender) TickerWithSignal() {
	defer r.WG.Done()
loop:
	for {
		tickerPollInterval := time.NewTicker(time.Duration(*r.PollIntervalTime) * time.Second)
		defer tickerPollInterval.Stop()
		select {
		case <-tickerPollInterval.C:
			// Is it good practice or not?
			r.GoPull <- struct{}{}
			r.GoPull <- struct{}{}
		case <-r.Done:
			break loop
		}
	}
}

func (r *MetricSender) AddInterval(
	memInfo *map[string]interface{},
	counter *atomic.Int64,
) {
	for {
		select {
		case <-r.GoPull:
			v, _ := mem.VirtualMemory()
			r.M.Lock()
			(*memInfo)["TotalMemory"] = float64(v.Total)
			(*memInfo)["FreeMemory"] = float64(v.Free)
			(*memInfo)["CPUutilization1"] = v.UsedPercent
			r.M.Unlock()
			counter.Add(1)
			fmt.Println("Done AddInterval!")
		case <-r.Done:
			return
		default:
			continue
		}
	}
}

func (r *MetricSender) PollInterval(
	memInfo *map[string]interface{},
	counter *atomic.Int64,
) {
	for {
		select {
		case <-r.GoPull:
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			b, err := json.Marshal(memStats)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			r.M.Lock()
			err = json.Unmarshal(b, memInfo)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			r.M.Unlock()
			counter.Add(1)
			fmt.Println("Done PollInterval!")
		case <-r.Done:
			return
		default:
			continue
		}
	}
}

func (r *MetricSender) SendMetric(value interface{}, metricType string, metricName string) error {
	url := fmt.Sprintf("%s/update/", r.URL)
	var s string
	switch metricType {
	case COUNTER:
		s = fmt.Sprintf("{\"id\":\"%s\",\"type\":\"%s\",\"delta\":%d}", metricName, metricType, value)
	case GAUGE:
		s = fmt.Sprintf("{\"id\":\"%s\",\"type\":\"%s\",\"value\":%f}", metricName, metricType, value)
	default:
		return fmt.Errorf("wrong type of metric: %v", metricType)
	}
	req := r.Client.R().SetBody(s).SetHeader("Content-Type", "application/json")
	if *AgentKey != "" {
		encoder := hmac.New(sha256.New, []byte(*AgentKey))
		encoder.Write([]byte(s))
		v := encoder.Sum(nil)
		req.SetHeader("HashSHA256", base64.RawURLEncoding.EncodeToString(v[:]))
	}
	var err error
	var resp *resty.Response
	f := func() error {
		resp, err = req.Post(
			url,
		)
		return err
	}
	err = general.RetryCode(f, syscall.ECONNREFUSED)
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

func (r *MetricSender) SendMetrics(metrics []general.Metrics) error {
	url := fmt.Sprintf("%s/updates/", r.URL)
	fmt.Println("Sending post request with url: " + url)
	var err error
	s, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	req := r.Client.R().SetBody(s).SetHeader("Content-Type", "application/json")
	if *AgentKey != "" {
		encoder := hmac.New(sha256.New, []byte(*AgentKey))
		encoder.Write([]byte(s))
		v := encoder.Sum(nil)
		req.SetHeader("HashSHA256", base64.RawURLEncoding.EncodeToString(v[:]))
	}
	var resp *resty.Response
	f := func() error {
		resp, err = req.Post(
			url,
		)
		return err
	}
	err = general.RetryCode(f, syscall.ECONNREFUSED)
	if err != nil {
		err = fmt.Errorf("request cannot be precossed, something is wrong: %w", err)
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("something wrong with resp - '%v', status code - %v", resp, resp.StatusCode())
	}
	fmt.Printf("Success: %v\n", resp.StatusCode())
	return nil
}

func (r *MetricSender) ReportLogic(m *map[string]interface{}, countOfUpdate *atomic.Int64) {
	metrics := make([]general.Metrics, 0)
	var err error
	for _, v := range *r.ListMetrics {
		casted, ok := (*m)[v].(float64)
		if ok {
			metrics = append(
				metrics,
				general.Metrics{ID: v, MType: GAUGE, Value: &casted},
			)
		} else {
			err = errors.Join(err, fmt.Errorf("cannot be cast to float64, wrong type for '%s' metric name with value '%v'", v, (*m)[v]))
		}
	}
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	delta := countOfUpdate.Load()
	metrics = append(
		metrics,
		general.Metrics{ID: "PollCount", MType: COUNTER, Delta: &delta},
	)
	c := rand.Float64()
	metrics = append(
		metrics,
		general.Metrics{ID: "RandomValue", MType: GAUGE, Value: &c},
	)
	err = r.SendMetrics(metrics)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("All metrics successfully sent")
}

func (r *MetricSender) ReportInterval(
	memInfo *map[string]interface{},
	countOfUpdate *atomic.Int64,
) {
	tickerReportInterval := time.NewTicker(time.Duration(*r.ReportIntervalTime) * time.Second)
	defer tickerReportInterval.Stop()
	for {
		select {
		case <-tickerReportInterval.C:
			r.M.Lock()
			r.ReportLogic(memInfo, countOfUpdate)
			r.M.Unlock()
			countOfUpdate.Swap(int64(0))
			fmt.Println("Done ReportInterval!")
		case <-r.Done:
			return
		}
	}
}
