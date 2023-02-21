package monitor

import (
	"encoding/json"
	"net/http"
	"os"
	"backnet/controllers"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type stats struct {
	PID statsPID `json:"pid"`
	OS  statsOS  `json:"os"`
}

type statsPID struct {
	CPU   float64 `json:"cpu"`
	RAM   uint64  `json:"ram"`
	Conns int     `json:"conns"`
}

type statsOS struct {
	CPU      float64 `json:"cpu"`
	RAM      uint64  `json:"ram"`
	TotalRAM uint64  `json:"total_ram"`
	LoadAvg  float64 `json:"load_avg"`
	Conns    int     `json:"conns"`
}

var (
	monitPidCpu   atomic.Value
	monitPidRam   atomic.Value
	monitPidConns atomic.Value

	monitOsCpu      atomic.Value
	monitOsRam      atomic.Value
	monitOsTotalRam atomic.Value
	monitOsLoadAvg  atomic.Value
	monitOsConns    atomic.Value
)

var (
	mutex sync.RWMutex
	once  sync.Once
	data  = &stats{}
)

// New creates a new middleware handler
func New() func(request *controllers.Request) {
	// Start routine to update statistics
	go func() {
		p, err := process.NewProcess(int32(os.Getpid()))

		if err == nil {
			updateStatistics(p)

			go func() {
				for {
					time.Sleep(time.Second * 3)

					updateStatistics(p)
				}
			}()
		}
	}()

	// Return new handler
	return func(request *controllers.Request) {
		if request.Request.Method == "POST" {
			mutex.Lock()
			data.PID.CPU = PidCpu()
			data.PID.RAM = PidRam()
			data.PID.Conns = PidConns()

			data.OS.CPU = OsCpu()
			data.OS.RAM = OsRam()
			data.OS.TotalRAM = OsTotalRam()
			data.OS.LoadAvg = OsLoadAvg()
			data.OS.Conns = OsConns()
			mutex.Unlock()

			request.Writer.Header().Set("Content-Type", "application/json")
			request.Writer.WriteHeader(http.StatusOK)

			jsonResponse, jsonError := json.Marshal(data)

			if jsonError == nil {
				request.Writer.Write(jsonResponse)
			} else {
				request.Writer.Write([]byte("{}"))
			}

			return
		}

		request.View([]string{
			"views/admin/layouts/main.html",
			"views/monitor/index.html",
		}, 200, map[string]any{
			"Title": "Admin Monitor",
		})
	}
}

func updateStatistics(p *process.Process) {
	pidCpu, _ := p.CPUPercent()
	monitPidCpu.Store(pidCpu / 10)

	if osCpu, _ := cpu.Percent(0, false); len(osCpu) > 0 {
		monitOsCpu.Store(osCpu[0])
	}

	if pidMem, _ := p.MemoryInfo(); pidMem != nil {
		monitPidRam.Store(pidMem.RSS)
	}

	if osMem, _ := mem.VirtualMemory(); osMem != nil {
		monitOsRam.Store(osMem.Used)
		monitOsTotalRam.Store(osMem.Total)
	}

	if loadAvg, _ := load.Avg(); loadAvg != nil {
		monitOsLoadAvg.Store(loadAvg.Load1)
	}

	pidConns, _ := net.ConnectionsPid("tcp", p.Pid)
	monitPidConns.Store(len(pidConns))

	osConns, _ := net.Connections("tcp")
	monitOsConns.Store(len(osConns))
}

func PidCpu() float64 {
	return monitPidCpu.Load().(float64)
}

func PidRam() uint64 {
	return monitPidRam.Load().(uint64)
}

func PidConns() int {
	return monitPidConns.Load().(int)
}

func OsCpu() float64 {
	return monitOsCpu.Load().(float64)
}

func OsRam() uint64 {
	return monitOsRam.Load().(uint64)
}

func OsTotalRam() uint64 {
	return monitOsTotalRam.Load().(uint64)
}

func OsLoadAvg() float64 {
	return monitOsLoadAvg.Load().(float64)
}

func OsConns() int {
	return monitOsConns.Load().(int)
}
