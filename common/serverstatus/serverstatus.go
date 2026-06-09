// Package serverstatus 采集服务器系统状态
package serverstatus

import (
	"errors"
	"fmt"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// GetSystemInfo 获取 CPU/内存/磁盘/运行时长
func GetSystemInfo() (Cpu float64, Mem float64, Disk float64, Uptime uint64, err error) {

	errorString := ""

	cpuPercent, err := cpu.Percent(0, false)
	// cpu.Percent 返回多个 CPU 的使用率，这里取总平均
	// 返回值为空或采集失败时，将 CPU 使用率记为 0 并汇总错误。
	if len(cpuPercent) > 0 && err == nil {
		Cpu = cpuPercent[0]
	} else {
		Cpu = 0
		errorString += fmt.Sprintf("get cpu usage failed: %s ", err)
	}

	memUsage, err := mem.VirtualMemory()
	if err != nil {
		errorString += fmt.Sprintf("get mem usage failed: %s ", err)
	} else {
		Mem = memUsage.UsedPercent
	}

	diskUsage, err := disk.Usage("/")
	if err != nil {
		errorString += fmt.Sprintf("get disk usage failed: %s ", err)
	} else {
		Disk = diskUsage.UsedPercent
	}

	uptime, err := host.Uptime()
	if err != nil {
		errorString += fmt.Sprintf("get uptime failed: %s ", err)
	} else {
		Uptime = uptime
	}

	if errorString != "" {
		err = errors.New(errorString)
	}

	return Cpu, Mem, Disk, Uptime, err
}
