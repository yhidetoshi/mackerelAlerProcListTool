# MackerelCPUAlertTool

- 目的
  - MackerelでCPU使用率のアラートが発生した場合にSlackにCPU使用率の高いプロセスリストをPostさせる
- 実装
  - `mackerel-client-go` を利用してツールを作成
    - https://github.com/mackerelio/mackerel-client-go
  - systemdのtimerで1分に1回実行させる
  - Mackerelのアラート一覧に自インスタンスのmackerel-idがあるかチェック
  - 自インスタンスのアラートが存在する場合、アラートの種類が `CPU %` で 管理画面の閾値設定値を超えている場合はSlackに以下のコマンド実行結果をPostする
    - `ps aux --sort -%cpu | head -n 5`

## Usage
``` 
> $ main -slackurl=<SLACKURL>
```

```
HOSTNAME: 		test-server
Threshold: 		80.0000
UnixTime: 		1553408909 to 1553409089
cpu.user.percentage:	1.3416875034728009
cpu.system.percentage:	0.5833472245374229
cpu.iowait.percentage:	0.04166805578707562
cpu.steal.percentage:	0.15000555648163583
cpu.irq.percentage:	0
cpu.softirq.percentage:	0
cpu.nice.percentage:	0
cpu.guest.percentage:	0
TotalCPUUsage: 		2.1
USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND

(ps aux --sort -%cpu | head -n 5) の結果が出力される
```

## Slack通知の結果
![Alt Text](https://github.com/yhidetoshi/Pictures/raw/master/Go_study/mackerel-alert-slack.png)
