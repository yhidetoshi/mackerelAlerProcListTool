package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ashwanthkumar/slack-go-webhook"
	"github.com/mackerelio/mackerel-client-go"
)

var (
	CMD            = "ps aux --sort -%cpu | head -n 5"
	IDFILE         = "/var/lib/mackerel-agent/id" // for Ubuntu
	argSlackURL    = flag.String("slackurl", "", "set slack url")
	argMackerelKey = flag.String("mkrkey", "", "set mkr key")
	cpuItems       = []string{
		"cpu.user.percentage",
		"cpu.system.percentage",
		"cpu.iowait.percentage",
		"cpu.steal.percentage",
		"cpu.irq.percentage",
		"cpu.softirq.percentage",
		"cpu.nice.percentage",
		"cpu.guest.percentage",
	}
	username = "MackerelClientTool"
	//channel  = "alert"
)

type HostParams struct {
	hostID   string
	hostName string
}

type AlertParams struct {
	alert    []string
	exist    bool
	cpuUsage float64
	hostName string
}

type HostMetricsParams struct {
	cpuUserRate         byte
	duration            uint64
	monitorID           string
	toUnixTime          int64
	fromUnixTime        int64
	warning             *float64
	cpuSumValuePerItems []float64
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
	flag.Parse()
	mkr := mackerel.NewClient(*argMackerelKey)

	hp := &HostParams{}
	hp.GetHostID()

	ap := &AlertParams{}
	ap.CheckOpenAlerts(mkr, hp.hostID)

	hp.FetchHostname(mkr)

	if ap.exist == true {
		hmp := &HostMetricsParams{}
		hmp.FetchMonitorConfigCPUDurationWarning(mkr)
		hmp.FetchMetricsValues(mkr, hp.hostID)
		hmp.FetchMetricsValues(mkr, hp.hostID)
		if ap.cpuUsage >= *hmp.warning {
			psList, err := exec.Command("sh", "-c", CMD).Output()

			if err != nil {
				fmt.Println("Error")
				os.Exit(1)
			}
			//fmt.Printf("%s", psList)
			//fmt.Println(ap.cpuUsage)

			PostSlack(hp.hostName, hmp.cpuSumValuePerItems[0], hmp.cpuSumValuePerItems[1],
				hmp.cpuSumValuePerItems[2], hmp.cpuSumValuePerItems[3], string(psList))

		}
	} else {
		fmt.Println("no match alert")
		os.Exit(0)
	}
}

func (hp *HostParams) GetHostID() {
	content, err := ioutil.ReadFile(IDFILE)
	if err != nil {
		fmt.Println("Error")
		os.Exit(1)
	}
	lines := strings.Split(string(content), "\n")
	hp.hostID = lines[0]
}

func (hp *HostParams) FetchHostname(mkr *mackerel.Client) {
	host, err := mkr.FindHost(hp.hostID)
	if err != nil {
		fmt.Println("no hosts")
		os.Exit(0)
	}
	hp.hostName = host.Name
	//fmt.Printf("HOSTNAME: \t\t%s\n", hp.hostName)
}

func (ap *AlertParams) CheckOpenAlerts(mkr *mackerel.Client, strHostID string) {

	ap.exist = false
	alerts, err := mkr.FindAlerts()
	if err != nil {
		fmt.Println("err")
		os.Exit(0)
	}

	for _, resAlert := range alerts.Alerts {
                if (resAlert.HostID == strHostID) && (resAlert.Type == "host") {
			ap.cpuUsage = resAlert.Value
			ap.hostName = resAlert.ID
			ap.exist = true
		}
		//fmt.Printf("AlertType: %v\t AlertHostID: %v\n", resAlert.Type, resAlert.HostID)
	}
}

func (hmp *HostMetricsParams) FetchMonitorConfigCPUDurationWarning(mkr *mackerel.Client) {
	var monitorHostMetricDuration MonitorHostMetricDuration
	var monitorHostMetricWarning MonitorHostMetricWarning

	monitors, err := mkr.FindMonitors()
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
			//fmt.Printf("Threshold: \t\t%s\n", strconv.FormatFloat(*hmp.warning, 'f', 4, 64))
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

func (hmp *HostMetricsParams) FetchMetricsValues(mkr *mackerel.Client, strHostID string) {
	var metricsCPUValue []CPUValue
	var beforeTime = (-1 * time.Duration(hmp.duration)) - 1
	//var totalCPUUsage float64
	cpuItemsValue := [][]mackerel.MetricValue{}

	// UnixTime
	//hmp.toUnixTime = time.Now().Unix()
	toTime := time.Now().Add(-1 * time.Minute)
	hmp.toUnixTime = toTime.Unix()

	fromTime := time.Now().Add(beforeTime * time.Minute)
	hmp.fromUnixTime = fromTime.Unix()

	// Print UnixTime
	//fmt.Printf("UnixTime: \t\t%v to %v\n", hmp.fromUnixTime, hmp.toUnixTime)

	// EXAMPLE(user): cpuUser, _ := mkr.FetchHostMetricValues(strHostID, "cpu.user.percentage", hmp.fromUnixTime, hmp.toUnixTime)
	for i := range cpuItems {
		tmp, _ := mkr.FetchHostMetricValues(strHostID, cpuItems[i], hmp.fromUnixTime, hmp.toUnixTime)
		cpuItemsValue = append(cpuItemsValue, tmp)
	}

	// Get CPU Usage per Item
	for i := range cpuItemsValue {
		jsonFormat(cpuItemsValue[i], &metricsCPUValue)
		tmp := calcTotalCPUPercentPerItem(metricsCPUValue)
		hmp.cpuSumValuePerItems = append(hmp.cpuSumValuePerItems, tmp)
	}
}

func PostSlack(hostName string, cpuUser float64, cpuSystem float64, cpuIOWait float64, cpuSteal float64, psList string) {
	field0 := slack.Field{Title: "HOSTNAME", Value: hostName}
	field1 := slack.Field{Title: "cpu.user", Value: strconv.FormatFloat(cpuUser, 'f', 4, 64)}
	field2 := slack.Field{Title: "cpu.system", Value: strconv.FormatFloat(cpuSystem, 'f', 4, 64)}
	field3 := slack.Field{Title: "cpu.iowait", Value: strconv.FormatFloat(cpuIOWait, 'f', 4, 64)}
	field4 := slack.Field{Title: "cpu.steal", Value: strconv.FormatFloat(cpuSteal, 'f', 4, 64)}
	field5 := slack.Field{Title: "ps list top 5", Value: "```" + psList + "```"}

	attachment := slack.Attachment{}
	attachment.AddField(field0).AddField(field1).AddField(field2).AddField(field3).AddField(field4).AddField(field5)
	color := "warning"
	attachment.Color = &color
	payload := slack.Payload{
		Username:    username,
		//Channel:     channel,
		Attachments: []slack.Attachment{attachment},
	}
	err := slack.Send(*argSlackURL, "", payload)
	if len(err) > 0 {
		os.Exit(1)
	}
}
