package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/ashwanthkumar/slack-go-webhook"
	"github.com/mackerelio/mackerel-client-go"
)

var (
	CMD    = "ps aux --sort -%cpu | head -n 5"
	IDFILE = "/var/lib/mackerel-agent/id" // for Ubuntu
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
	exist    bool
	cpuUsage float64
	hostName string
}

type HostMetricsParams struct {
	cpuUserRate byte
	duration    uint64
	warning     *float64
}

type Host interface {
	GetHostID() string
	FetchHostname(*mackerel.Client, string) string
}

type Alert interface {
	CheckOpenAlerts(*mackerel.Client, string) (exist bool, usage float64)
}

type HostMetrics interface {
	FetchMonitorConfigCPUDurationWarning(*mackerel.Client) *float64
}

type MonitorHostMetricDuration struct {
	Duration uint64 `json:"duration,omitempty"`
}

type MonitorHostMetricWarning struct {
	Warning *float64 `json:"warning"`
}

func main() {
	flag.Parse()
	mkr := mackerel.NewClient(*argMackerelKey)

	var host Host = &HostParams{}
	var alert Alert = &AlertParams{}
	var metrics HostMetrics = &HostMetricsParams{}

	hostID := host.GetHostID()
	alertExist, cpuUsage := alert.CheckOpenAlerts(mkr, hostID)

	// Alert一覧にhost自身のIDが存在する場合に処理実行
	if alertExist == true {
		hostname := host.FetchHostname(mkr, hostID)
		warning := metrics.FetchMonitorConfigCPUDurationWarning(mkr)

		if cpuUsage >= *warning {
			psList, err := exec.Command("sh", "-c", CMD).Output()

			if err != nil {
				fmt.Println("Error")
				os.Exit(1)
			}
			//fmt.Printf("%s", psList)
			PostSlack(hostname, string(psList))
		}
	} else {
		fmt.Println("no match alert")
		os.Exit(0)
	}
}

func (hp *HostParams) GetHostID() string {
	content, err := ioutil.ReadFile(IDFILE)
	if err != nil {
		fmt.Println("Error")
		os.Exit(1)
	}
	lines := strings.Split(string(content), "\n")
	hp.hostID = lines[0]

	return hp.hostID

}

func (hp *HostParams) FetchHostname(mkr *mackerel.Client, hostID string) string {
	host, err := mkr.FindHost(hostID)
	if err != nil {
		fmt.Println("no hosts")
		os.Exit(0)
	}
	hp.hostName = host.Name
	fmt.Printf("HOSTNAME: \t\t%s\n", hp.hostName)

	return hp.hostName
}

func (ap *AlertParams) CheckOpenAlerts(mkr *mackerel.Client, strHostID string) (bool, float64) {

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

	return ap.exist, ap.cpuUsage
}

func (hmp *HostMetricsParams) FetchMonitorConfigCPUDurationWarning(mkr *mackerel.Client) *float64 {
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
		}
	}
	return hmp.warning
}

func PostSlack(hostName string, psList string) {
	field0 := slack.Field{Title: "HOSTNAME", Value: hostName}
	field1 := slack.Field{Title: "ps list top 5", Value: "```" + psList + "```"}

	attachment := slack.Attachment{}
	attachment.AddField(field0).AddField(field1)
	color := "warning"
	attachment.Color = &color
	payload := slack.Payload{
		Username: username,
		//Channel:     channel,
		Attachments: []slack.Attachment{attachment},
	}
	err := slack.Send(*argSlackURL, "", payload)
	if len(err) > 0 {
		os.Exit(1)
	}
}
