package controller

import (
	"runtime"
	"time"

	"ginproject/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

var processStartTime = time.Now()

type DashboardController struct{}

func NewDashboardController() *DashboardController {
	return &DashboardController{}
}

type serverInfoResponse struct {
	CPU struct {
		UsagePercent float64 `json:"usage_percent"`
		Cores        int     `json:"cores"`
	} `json:"cpu"`
	Memory struct {
		Total        uint64  `json:"total"`
		Used         uint64  `json:"used"`
		UsagePercent float64 `json:"usage_percent"`
	} `json:"memory"`
	Disk struct {
		Total        uint64  `json:"total"`
		Used         uint64  `json:"used"`
		UsagePercent float64 `json:"usage_percent"`
	} `json:"disk"`
	Runtime struct {
		GoVersion string `json:"go_version"`
		OS        string `json:"os"`
		Arch      string `json:"arch"`
		Goroutines int   `json:"goroutines"`
		MemAlloc  uint64 `json:"mem_alloc"`
		Uptime    int64  `json:"uptime"`
	} `json:"runtime"`
}

// GetServerInfo 获取服务器资源使用情况
// @Summary      获取服务器信息
// @Description  返回 CPU、内存、磁盘使用率及 Go 运行时信息
// @Tags         dashboard
// @Security     BearerAuth
// @Success      200  {object}  utils.Response{data=serverInfoResponse}
// @Router       /api/dashboard/server-info [get]
func (ctl *DashboardController) GetServerInfo(c *gin.Context) {
	var resp serverInfoResponse

	// CPU
	if percents, err := cpu.Percent(time.Second, false); err == nil && len(percents) > 0 {
		resp.CPU.UsagePercent = percents[0]
	}
	if counts, err := cpu.Counts(true); err == nil {
		resp.CPU.Cores = counts
	}

	// 内存
	if v, err := mem.VirtualMemory(); err == nil {
		resp.Memory.Total = v.Total
		resp.Memory.Used = v.Used
		resp.Memory.UsagePercent = v.UsedPercent
	}

	// 磁盘（根分区）
	if usage, err := disk.Usage("/"); err == nil {
		resp.Disk.Total = usage.Total
		resp.Disk.Used = usage.Used
		resp.Disk.UsagePercent = usage.UsedPercent
	}

	// Go 运行时
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	resp.Runtime.GoVersion = runtime.Version()
	resp.Runtime.OS = runtime.GOOS
	resp.Runtime.Arch = runtime.GOARCH
	resp.Runtime.Goroutines = runtime.NumGoroutine()
	resp.Runtime.MemAlloc = m.Alloc
	resp.Runtime.Uptime = int64(time.Since(processStartTime).Seconds())

	utils.Success(c, resp)
}
