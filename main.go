package main

import (
	"encoding/json"
	"fmt"
	"github.com/mackerelio/mackerel-client-go"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var (
	client   = mackerel.NewClient("XXX")
	cpuItems = []string{
		"cpu.user.percentage",
		"cpu.system.percentage",
		"cpu.iowait.percentage",
		"cpu.steal.percentage",
		"cpu.irq.percentage",
		"cpu.softirq.percentage",
		"cpu.nice.percentage",
		"cpu.guest.percentage",
	}
)

type HostParams struct {
	hostID string
}

type AlertParams struct {
	alert    []string
	exist    bool
	cpuUsage float64
}

type HostMetricsParams struct {
	cpuUserRate  byte
	duration     uint64
	monitorID    string
	toUnixTime   int64
	fromUnixTime int64
	warning      *float64
}

type Host interface {
	GetHostID()
}

type Alert interface {
	CheckOpenAlerts()
}

type HostMetrics interface {
	FetchMetricsValues()
}

type MonitorHostMetricDuration struct {
	Duration uint64 `json:"duration,omitempty"`
}
type MonitorHostMetricWarning struct {
	Warning *float64 `json:"warning"`
}

type CPUValue struct {
	Time  int64       `json:"time"`
	Value interface{} `json:"value"`
}

func main() {

	hp := &HostParams{}
	hp.GetHostID()

	ap := &AlertParams{}
	ap.CheckOpenAlerts(hp.hostID)

	// Alert一覧にhost自身のIDが存在する場合に処理実行
	if ap.exist == true {
		hmp := &HostMetricsParams{}
		hmp.FetchMonitorConfigCPUDurationWarning()
		hmp.FetchMetricsValues(hp.hostID)
		if ap.cpuUsage >= *hmp.warning {
			fmt.Println("CPU使用率の高いコマンド発行する")
			fmt.Println(ap.cpuUsage)
		}
	} else {
		fmt.Println("no match alert")
		os.Exit(0)
	}

	//FetchMetricsValues(hostID)
}

func (hp *HostParams) GetHostID() {
	content, err := ioutil.ReadFile("/Users/hidetoshi/mkr-id")
	if err != nil {
		fmt.Println("Error")
		os.Exit(1)
	}
	lines := strings.Split(string(content), "\n")
	hp.hostID = lines[0]
}

func (ap *AlertParams) CheckOpenAlerts(strHostID string) {

	ap.exist = true
	alerts, err := client.FindAlerts()
	if err != nil {
		fmt.Println("no alerts")
		os.Exit(0)
	}

	for _, resAlert := range alerts.Alerts {
		if resAlert.HostID == strHostID {
			ap.cpuUsage = resAlert.Value
			ap.exist = true
		} else {
			ap.exist = false
		}
		fmt.Printf("AlertType: %v\t AlertHostID: %v\n", resAlert.Type, resAlert.HostID)
	}
}

func (hmp *HostMetricsParams) FetchMonitorConfigCPUDurationWarning() {
	var monitorHostMetricDuration MonitorHostMetricDuration
	var monitorHostMetricWarning MonitorHostMetricWarning

	monitors, err := client.FindMonitors()
	if err != nil {
		fmt.Println("not get monitor conf")
		os.Exit(1)
	}

	for _, resMonitor := range monitors {
		if resMonitor.MonitorName() == "CPU %" {

			// Get Duration value
			durationBytesJSON, _ := json.Marshal(resMonitor)
			bytesDuration := []byte(durationBytesJSON)

			if err := json.Unmarshal(bytesDuration, &monitorHostMetricDuration); err != nil {
				fmt.Println("JSON Unmarshal error:", err)
			}
			hmp.duration = monitorHostMetricDuration.Duration

			// Get Warning value
			warningBytesJSON, _ := json.Marshal(resMonitor)
			bytesWarning := []byte(warningBytesJSON)

			if err := json.Unmarshal(bytesWarning, &monitorHostMetricWarning); err != nil {
				fmt.Println("JSON Unmarshal error:", err)
			}
			hmp.warning = monitorHostMetricWarning.Warning
			fmt.Println(*hmp.warning)
			//fmt.Printf("%v\n", monitorHostMetric.Duration)
		}
	}
}

func calcTotalCPUPercentPerItem(cv []CPUValue) float64 {
	var sum float64
	for i := range cv {
		sum += cv[i].Value.(float64)
	}
	return sum / float64(len(cv))
}

func jsonFormat(m []mackerel.MetricValue, cv *[]CPUValue) {
	// APIの戻り値をJSONで受ける
	bytesJSON, _ := json.Marshal(m)
	bytes := []byte(bytesJSON)

	// JSONから必要なデータだけ構造体定義してパースする
	if err := json.Unmarshal(bytes, &cv); err != nil {
		fmt.Println("JSON Unmarshal error:", err)
	}
}

func (hmp *HostMetricsParams) FetchMetricsValues(strHostID string) {
	var metricsCPUValue []CPUValue
	var beforeTime = (-1 * time.Duration(hmp.duration)) - 1
	var totalCPUUsage float64
	cpuItemsValue := [][]mackerel.MetricValue{}
	cpuSumValuePerItems := []float64{}

	// UnixTime
	//hmp.toUnixTime = time.Now().Unix()
	toTime := time.Now().Add(-1 * time.Minute)
	hmp.toUnixTime = toTime.Unix()

	fromTime := time.Now().Add(beforeTime * time.Minute)
	hmp.fromUnixTime = fromTime.Unix()

	// Print UnixTime
	fmt.Printf("%v %v\n", hmp.fromUnixTime, hmp.toUnixTime)

	// EXAMPLE(user): cpuUser, _ := client.FetchHostMetricValues(strHostID, "cpu.user.percentage", hmp.fromUnixTime, hmp.toUnixTime)
	for i := range cpuItems {
		tmp, _ := client.FetchHostMetricValues(strHostID, cpuItems[i], hmp.fromUnixTime, hmp.toUnixTime)
		cpuItemsValue = append(cpuItemsValue, tmp)
	}

	// Get CPU Usage per Item
	for i := range cpuItemsValue {
		jsonFormat(cpuItemsValue[i], &metricsCPUValue)
		tmp := calcTotalCPUPercentPerItem(metricsCPUValue)
		cpuSumValuePerItems = append(cpuSumValuePerItems, tmp)
	}
	// Calc total cpu utilization
	for i := range cpuSumValuePerItems {
		fmt.Printf("%s\t%v\n", cpuItems[i], cpuSumValuePerItems[i])
		totalCPUUsage += cpuSumValuePerItems[i]
	}

	result := fmt.Sprintf("%.1f", totalCPUUsage)
	fmt.Printf("%v\n", result)
}
