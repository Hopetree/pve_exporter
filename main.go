package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

// 将要注册的指标定义为全局变量
var (
	cpuTemperature = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_temperature_celsius",
		Help: "Current temperature of the CPU in degrees Celsius",
	})
	powerUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "power_usage_watts",
		Help: "Current power usage in watts",
	})
)

var addr = flag.String("listen-address", ":9010", "The address to listen on for HTTP requests.")

func initCustomRegistry() *prometheus.Registry {
	// 创建一个新的注册表
	reg := prometheus.NewRegistry()
	// 注册自定义指标
	reg.MustRegister(cpuTemperature)
	reg.MustRegister(powerUsage)
	return reg
}

// executeCommand 执行给定的 shell 命令并返回执行结果
func executeCommand(cmd string) (string, error) {
	// 创建一个新的命令
	command := exec.Command("bash", "-c", cmd)

	// 捕获命令的标准输出和错误输出
	var out bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &out
	command.Stderr = &stderr

	// 运行命令
	err := command.Run()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

// getInfoByRegexp 使用正则提取信息
func getInfoByRegexp(input, pattern string) (string, error) {
	// 定义匹配
	re := regexp.MustCompile(pattern)

	// 查找匹配项
	match := re.FindStringSubmatch(input)

	// 如果找到匹配项，返回第一个捕获组的内容
	if len(match) > 1 {
		str := match[1]
		return str, nil
	}

	// 如果没有找到匹配项，返回错误
	return "", fmt.Errorf("no info found")
}

// recordMetrics 可以每个指标一个单独的 Goroutine 来采集
func recordMetrics() {
	go func() {
		for {
			var temperature, power float64

			output, err := executeCommand("sensors")
			if err != nil {
				fmt.Println(err)
			}

			temperatureStr, err := getInfoByRegexp(output, `Tctl:\s*\+([0-9.]+)°C`)
			powerStr, err := getInfoByRegexp(output, `PPT:\s*([0-9.]+)\s*W`)

			temperature, err = strconv.ParseFloat(temperatureStr, 64)
			power, err = strconv.ParseFloat(powerStr, 64)

			cpuTemperature.Set(temperature)
			powerUsage.Set(power)

			// 休眠 10 秒
			time.Sleep(10 * time.Second)
		}
	}()
}

func main() {
	flag.Parse()
	// 创建自定义注册表并注册指标
	reg := initCustomRegistry()

	// 开始记录指标
	recordMetrics()

	// 使用自定义注册表创建 HTTP 处理器
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	_ = http.ListenAndServe(*addr, nil)

}
