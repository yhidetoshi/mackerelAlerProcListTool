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
	client = mackerel.NewClient("XXX")
)

//const N = -2

type HostParams struct {
	hostID string
}

type AlertParams struct {
	alert []string
	exist bool
}

type HostMetricsParams struct {
	cpuUserRate  byte
	duration     uint64
	monitorID    string
	toUnixTime   int64
	fromUnixTime int64
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

type MonitorHostMetric struct {
	Duration uint64 `json:"duration,omitempty"`
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
		hmp.FetchMonitorConfigCPUDuration()
		hmp.FetchMetricsValues(hp.hostID)
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
			ap.exist = true
		}

		fmt.Println(resAlert.Type)
		fmt.Println(resAlert.HostID)
	}
}

func (hmp *HostMetricsParams) FetchMonitorConfigCPUDuration() {
	var monitorHostMetric MonitorHostMetric

	monitors, err := client.FindMonitors()
	if err != nil {
		fmt.Println("not get monitor conf")
		os.Exit(1)
	}

	for _, resMonitor := range monitors {
		if resMonitor.MonitorName() == "CPU %" {

			bytesJSON, _ := json.Marshal(resMonitor)
			bytes := []byte(bytesJSON)
			json.Unmarshal(bytes, &monitorHostMetric)

			hmp.duration = monitorHostMetric.Duration
			//fmt.Printf("%v\n", monitorHostMetric.Duration)
		}
	}
}

func jsonMarshal(v []mackerel.MetricValue) []byte {
	bytesJSON, _ := json.Marshal(v)
	bytes := []byte(bytesJSON)

	return bytes
}

func calcTotalCPUPercent(v []CPUValue) float64 {
	var sum float64
	for i := range v {
		sum += v[i].Value.(float64)
	}
	return sum
}

func jsonFormat(m []mackerel.MetricValue, cv *[]CPUValue) {
	bytesJSON, _ := json.Marshal(m)
	bytes := []byte(bytesJSON)

	if err := json.Unmarshal(bytes, &cv); err != nil {
		fmt.Println("JSON Unmarshal error:", err)
	}
}

func (hmp *HostMetricsParams) FetchMetricsValues(strHostID string) {
	var metricsCPUValue []CPUValue
	var beforeTime = (-1 * time.Duration(hmp.duration)) - 1

	// UnixTime
	//hmp.toUnixTime = time.Now().Unix()
	toTime := time.Now().Add(-1 * time.Minute)
	hmp.toUnixTime = toTime.Unix()

	fromTime := time.Now().Add(beforeTime * time.Minute)
	hmp.fromUnixTime = fromTime.Unix()

	fmt.Printf("%v %v \n ", hmp.fromUnixTime, hmp.toUnixTime)

	cpuUser, _ := client.FetchHostMetricValues(strHostID, "cpu.user.percentage", hmp.fromUnixTime, hmp.toUnixTime)
	cpuSystem, _ := client.FetchHostMetricValues(strHostID, "cpu.system.percentage", hmp.fromUnixTime, hmp.toUnixTime)
	cpuIOWait, _ := client.FetchHostMetricValues(strHostID, "cpu.iowait.percentage", hmp.fromUnixTime, hmp.toUnixTime)
	cpuSteal, _ := client.FetchHostMetricValues(strHostID, "cpu.steal.percentage", hmp.fromUnixTime, hmp.toUnixTime)
	cpuIrq, _ := client.FetchHostMetricValues(strHostID, "cpu.irq.percentage", hmp.fromUnixTime, hmp.toUnixTime)
	cpuSoftirq, _ := client.FetchHostMetricValues(strHostID, "cpu.softirq.percentage", hmp.fromUnixTime, hmp.toUnixTime)
	cpuNice, _ := client.FetchHostMetricValues(strHostID, "cpu.nice.percentage", hmp.fromUnixTime, hmp.toUnixTime)
	cpuGuest, _ := client.FetchHostMetricValues(strHostID, "cpu.guest.percentage", hmp.fromUnixTime, hmp.toUnixTime)

	// CPU User
	jsonFormat(cpuUser, &metricsCPUValue)
	sumCPUUser := calcTotalCPUPercent(metricsCPUValue)

	// CPU User
	jsonFormat(cpuSystem, &metricsCPUValue)
	sumCPUSystem := calcTotalCPUPercent(metricsCPUValue)

	// CPU IOWait
	jsonFormat(cpuIOWait, &metricsCPUValue)
	sumCPUIOWait := calcTotalCPUPercent(metricsCPUValue)

	// CPU IOWait
	jsonFormat(cpuSteal, &metricsCPUValue)
	sumCPUSteal := calcTotalCPUPercent(metricsCPUValue)

	// CPU Irq
	jsonFormat(cpuIrq, &metricsCPUValue)
	sumCPUIrq := calcTotalCPUPercent(metricsCPUValue)

	// CPU Softirq
	jsonFormat(cpuSoftirq, &metricsCPUValue)
	sumCPUSoftirq := calcTotalCPUPercent(metricsCPUValue)

	// CPU Nice
	jsonFormat(cpuNice, &metricsCPUValue)
	sumCPUNice := calcTotalCPUPercent(metricsCPUValue)

	// CPU Guest
	jsonFormat(cpuGuest, &metricsCPUValue)
	sumCPUGuest := calcTotalCPUPercent(metricsCPUValue)

	//aveCPUUser = sumCPUUser / float64(len(metricsCPUUserValue))

	fmt.Println("===")
	fmt.Println(sumCPUUser)
	fmt.Println(sumCPUSystem)
	fmt.Println(sumCPUIOWait)
	fmt.Println(sumCPUSteal)
	fmt.Println(sumCPUSteal)
	fmt.Println(sumCPUIrq)
	fmt.Println(sumCPUSoftirq)
	fmt.Println(sumCPUNice)
	fmt.Println(sumCPUGuest)

}

/*
func FindHosts() {
	hosts, _ := client.FindHosts(&mackerel.FindHostsParam{
		Service:  "stg",
		Roles:    []string{"webapp"},
		Statuses: []string{mackerel.HostStatusWorking},
	})
	fmt.Println("Hostname\tHostID\n-------")
	for _, resHost := range hosts {
		fmt.Println(resHost.Name, resHost.ID)
	}
}
*/
