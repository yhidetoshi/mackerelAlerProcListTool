# mackerelAlerProcListTool

Blog: [MackerelのCPU・Memoryアラート発生時にそれぞれの高負荷なプロセスをSlackに通知するツールを作った](https://yhidetoshi.github.io/2019/06/mackerel%E3%81%AEcpumemory%E3%82%A2%E3%83%A9%E3%83%BC%E3%83%88%E7%99%BA%E7%94%9F%E6%99%82%E3%81%AB%E3%81%9D%E3%82%8C%E3%81%9E%E3%82%8C%E3%81%AE%E9%AB%98%E8%B2%A0%E8%8D%B7%E3%81%AA%E3%83%97%E3%83%AD%E3%82%BB%E3%82%B9%E3%82%92slack%E3%81%AB%E9%80%9A%E7%9F%A5%E3%81%99%E3%82%8B%E3%83%84%E3%83%BC%E3%83%AB%E3%82%92%E4%BD%9C%E3%81%A3%E3%81%9F/)

- 目的
  - MackerelでCPU/MEM使用率のアラートが発生した場合にSlackにCPU/MEM使用率の高いプロセスリストをPostさせる
- 実装
  - `mackerel-client-go` を利用してツールを作成
    - https://github.com/mackerelio/mackerel-client-go
  - mackerel-agentのidをインスタンス内部で取得します。（今回はUbuntuのパスを指定）他のディストリビューションの場合は修正が必要です。  
  - systemdのtimerで1分に1回実行させる
  - Mackerelのアラート一覧に自インスタンスのmackerel-idがあるかチェック
  - 自インスタンスのアラートが存在する場合、アラートの種類が `CPU % / Memory &` で 管理画面の閾値設定値を超えている場合はSlackに以下のコマンド実行結果とホスト名、cpu使用率の内訳をPostする
    - `ps aux --sort -%cpu | head -n 6`
    - `ps aux --sort -%mem | head -n 6`

## Usage

``` 
> $ main -slackurl=<SLACKURL> -mkrkey=<MACKEREL_API_KEY>
```

- `MACKEREL_API_KEY` は readonly


■ 実行結果（ターミナル） 例なので、閾値以下の結果を出力されています
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


## Ansibleでデプロイする方法
- [Go製のツール ( MackerelCPUAlertTool ) をAnsibleでEC2インスタンスにデプロイする](https://yhidetoshi.github.io/2019/04/go%E8%A3%BD%E3%81%AE%E3%83%84%E3%83%BC%E3%83%AB-mackerelcpualerttool-%E3%82%92ansible%E3%81%A7ec2%E3%82%A4%E3%83%B3%E3%82%B9%E3%82%BF%E3%83%B3%E3%82%B9%E3%81%AB%E3%83%87%E3%83%97%E3%83%AD%E3%82%A4%E3%81%99%E3%82%8B/)
