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
	//CMDCPU ps command for cpu
	CMDCPU         = "ps aux --sort -%cpu | head -n 6"
	//CMDMEM ps command for mem
	CMDMEM         = "ps aux --sort -%mem | head -n 6"
	// IDFILE id file path
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
	mkrKey   = os.Getenv("MACKEREL_APIKEY")
	mkr      = mackerel.NewClient(mkrKey)
)

const (
	alertTime = -3
)

// HostParams host value
type HostParams struct {
	hostID   string
	hostName string
}

// AlertParams alert value
type AlertParams struct {
	hostName      string
	monitorIDList []string
}

// HostMetricsParams host metric value
type HostMetricsParams struct {
	cpuUserRate         byte
	duration            uint64
	cpuMonitorID        string
	memMonitorID        string
	toUnixTime          int64
	fromUnixTime        int64
	cpuWarning          *float64
	memWarning          *float64
	cpuSumValuePerItems []float64
}

// MonitorHostMetricDuration get monitor host value
type MonitorHostMetricDuration struct {
	Duration uint64 `json:"duration,omitempty"`
}

// MonitorHostCPUMetricWarning get monitor cpu value
type MonitorHostCPUMetricWarning struct {
	Warning *float64 `json:"warning"`
}

// MonitorHostMemMetricWarning get monitor host mem value
type MonitorHostMemMetricWarning struct {
	Warning *float64 `json:"warning"`
}

// CPUValue get host cpu value
type CPUValue struct {
	Time  int64       `json:"time"`
	Value interface{} `json:"value"`
}

func main() {
	flag.Parse()

	hp := &HostParams{}
	hp.GetHostID()
	hp.FetchHostname(mkr)

	// 監視ルールのIDを取得
	hmp := &HostMetricsParams{}
	hmp.FetchMonitorID(mkr)

	// 発生している監視ルールIDを取得
	ap := &AlertParams{}
	ap.FetchOpenAlerts(mkr, hp.hostID)

	// CPU MEMのアラートが発生しているか確認( 発生している監視ルールIDにCPUとMemが含まれているか判定 )
	cpuAlertFlag := CheckMonitorIDContains(ap.monitorIDList, hmp.cpuMonitorID)
	memAlertFlag := CheckMonitorIDContains(ap.monitorIDList, hmp.memMonitorID)

	// CPUアラートが発生した場合の処理
	if cpuAlertFlag == true {
		hmp.FetchMonitorConfigDurationWarning(mkr)
		hmp.FetchMetricsValues(mkr, hp.hostID)
		psList, err := exec.Command("sh", "-c", CMDCPU).Output()
		if err != nil {
			fmt.Println("CPU List Error")
			os.Exit(1)
		}

		// CPUのプロセスリストをSlackへPost
		PostSlackCPU(hp.hostName, hmp.cpuSumValuePerItems[0], hmp.cpuSumValuePerItems[1],
			hmp.cpuSumValuePerItems[2], hmp.cpuSumValuePerItems[3], string(psList))

		// CPUアラートが発生していない場合の処理
	} else {
		fmt.Println("no match CPU alert")
	}

	// Memアラート発生時の処理
	if memAlertFlag == true {
		psList, err := exec.Command("sh", "-c", CMDMEM).Output()
		if err != nil {
			fmt.Println("Mem List Error")
			os.Exit(1)
		}

		// MemのプロセスリストをSlackへPost
		PostSlackMem(hp.hostName, string(psList))

		// Memアラートが発生していない場合の処理
	} else {
		fmt.Println("no match Mem alert")
	}
}

// CheckMonitorIDContains Listの中に要素が存在するかの判定
func CheckMonitorIDContains(ids []string, id string) bool {
	for _, v := range ids {
		if id == v {
			return true
		}
	}
	return false
}

// GetHostID hostidを取得
func (hp *HostParams) GetHostID() {
	content, err := ioutil.ReadFile(IDFILE)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	lines := strings.Split(string(content), "\n")
	hp.hostID = lines[0]
}

// FetchHostname host名を取得
func (hp *HostParams) FetchHostname(mkr *mackerel.Client) {
	host, err := mkr.FindHost(hp.hostID)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	hp.hostName = host.Name
}

// FetchOpenAlerts アラート発生中の情報を取得
func (ap *AlertParams) FetchOpenAlerts(mkr *mackerel.Client, strHostID string) {
	alerts, err := mkr.FindAlerts()
	baseTime := time.Now().Add(alertTime * time.Minute)

	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	for _, resAlert := range alerts.Alerts {
		if (resAlert.HostID == strHostID) && (resAlert.Type == "host") && (resAlert.OpenedAt > baseTime.Unix()) {
			ap.monitorIDList = append(ap.monitorIDList, resAlert.MonitorID)
		}
	}
}

// FetchMonitorID 監視設定idを取得
func (hmp *HostMetricsParams) FetchMonitorID(mkr *mackerel.Client) {
	monitors, err := mkr.FindMonitors()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, resMonitor := range monitors {
		if resMonitor.MonitorName() == "CPU %" {
			hmp.cpuMonitorID = resMonitor.MonitorID()
		}
		if resMonitor.MonitorName() == "Memory %" {
			hmp.memMonitorID = resMonitor.MonitorID()
		}
	}

}

// FetchMonitorConfigDurationWarning 監視設定のwarnningに設定している値を取得
func (hmp *HostMetricsParams) FetchMonitorConfigDurationWarning(mkr *mackerel.Client) {
	var monitorHostMetricDuration MonitorHostMetricDuration
	var monitorHostCPUMetricWarning MonitorHostCPUMetricWarning
	var monitorHostMemMetricWarning MonitorHostMemMetricWarning

	monitors, err := mkr.FindMonitors()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, resMonitor := range monitors {

		// CPUのMonitorIDとDuration/Warmomg値を取得
		if resMonitor.MonitorName() == "CPU %" {

			// Get Duration value
			durationBytesJSON, _ := json.Marshal(resMonitor)
			bytesDuration := []byte(durationBytesJSON)

			if err := json.Unmarshal(bytesDuration, &monitorHostMetricDuration); err != nil {
				fmt.Println("JSON Unmarshal error:", err)
			}
			hmp.duration = monitorHostMetricDuration.Duration
			//hmp.cpuMonitorID = resMonitor.MonitorID()

			// Get CPU Warning value
			warningBytesJSON, _ := json.Marshal(resMonitor)
			bytesWarning := []byte(warningBytesJSON)

			if err := json.Unmarshal(bytesWarning, &monitorHostCPUMetricWarning); err != nil {
				fmt.Println("JSON Unmarshal error:", err)
			}
			hmp.cpuWarning = monitorHostCPUMetricWarning.Warning

		}

		// MemoryのMonitorIDとWarning値を取得
		if resMonitor.MonitorName() == "Memory %" {
			hmp.memMonitorID = resMonitor.MonitorID()

			// Get Mem Warning value
			warningBytesJSON, _ := json.Marshal(resMonitor)
			bytesWarning := []byte(warningBytesJSON)

			if err := json.Unmarshal(bytesWarning, &monitorHostMemMetricWarning); err != nil {
				fmt.Println("JSON Unmarshal error:", err)
			}
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

// FetchMetricsValues ホストメトリクスを取得
func (hmp *HostMetricsParams) FetchMetricsValues(mkr *mackerel.Client, strHostID string) {
	var metricsCPUValue []CPUValue
	var beforeTime = (-1 * time.Duration(hmp.duration)) - 1
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

// PostSlackCPU CPUのプロセスリストをSlackに投稿
func PostSlackCPU(hostName string, cpuUser float64, cpuSystem float64, cpuIOWait float64, cpuSteal float64, psList string) {
	field0 := slack.Field{Title: "HOSTNAME", Value: hostName}
	field1 := slack.Field{Title: "cpu.user", Value: strconv.FormatFloat(cpuUser, 'f', 4, 64)}
	field2 := slack.Field{Title: "cpu.system", Value: strconv.FormatFloat(cpuSystem, 'f', 4, 64)}
	field3 := slack.Field{Title: "cpu.iowait", Value: strconv.FormatFloat(cpuIOWait, 'f', 4, 64)}
	field4 := slack.Field{Title: "cpu.steal", Value: strconv.FormatFloat(cpuSteal, 'f', 4, 64)}
	field5 := slack.Field{Title: "CPU ps list top 5", Value: "```" + psList + "```"}

	attachment := slack.Attachment{}
	attachment.AddField(field0).AddField(field1).AddField(field2).AddField(field3).AddField(field4).AddField(field5)
	color := "warning"
	attachment.Color = &color
	payload := slack.Payload{
		Username: username,
		//Channel:     channel,
		Attachments: []slack.Attachment{attachment},
	}
	err := slack.Send(*argSlackURL, "", payload)
	if len(err) > 0 {
		fmt.Println(err)
		os.Exit(1)
	}
}

// PostSlackMem MemoryのプロセスリストをSlackに投稿
func PostSlackMem(hostName string, psList string) {
	field0 := slack.Field{Title: "HOSTNAME", Value: hostName}
	field1 := slack.Field{Title: "Mem ps list top 5", Value: "```" + psList + "```"}

	attachment := slack.Attachment{}
	attachment.AddField(field0).AddField(field1)
	color := "warning"
	attachment.Color = &color
	payload := slack.Payload{
		Username:    username,
		Attachments: []slack.Attachment{attachment},
	}
	err := slack.Send(*argSlackURL, "", payload)
	if len(err) > 0 {
		fmt.Println(err)
		os.Exit(1)
	}
}
