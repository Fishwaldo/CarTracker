package perf

import (
	"context"
	"time"

	"net/http"

	"github.com/Fishwaldo/CarTracker/internal"
	tm "github.com/Fishwaldo/CarTracker/internal/taskmanager"
	"github.com/Fishwaldo/CarTracker/internal/web"
	"github.com/go-logr/logr"
	"github.com/Fishwaldo/go-taskmanager"
	"github.com/labstack/echo/v4"
	"github.com/sasha-s/go-deadlock"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/spf13/viper"
)

func init() {
		viper.SetDefault("perf.poll", 5)
		viper.SetDefault("perf.timeout", 5)
		internal.RegisterPlugin("perf", &Perf)		
}


type PerfStats struct {
	Cpuinfo        []cpu.InfoStat
	Cputime        []cpu.TimesStat
	Cpuutilization []float64
	Diskpartions   []disk.PartitionStat
	Diskiocounter  map[string]disk.IOCountersStat
	Hostinfo       *host.InfoStat
	Hosttemp       []host.TemperatureStat
	Hostusers      []host.UserStat
	Loadavg        *load.AvgStat
	Loadprocs      *load.MiscStat
	Memstats       *mem.VirtualMemoryStat
	Memswap        *mem.SwapMemoryStat
	Netstats       []net.IOCountersStat
	Netaddr        []net.InterfaceStat
}

type PerfS struct {
	Performance PerfStats
	log         logr.Logger
	mx          deadlock.RWMutex
}

var Perf PerfS

func (p *PerfS) Start(log logr.Logger) {
	p.log = log
	var err error
	ctx, _ := context.WithTimeout(context.Background(), viper.GetDuration("perf.timeout") *time.Second)

	if p.Performance.Cpuinfo, err = cpu.InfoWithContext(ctx); err != nil {
		p.log.Error(err, "Can't Get CPU Info")
	}
	if p.Performance.Diskpartions, err = disk.PartitionsWithContext(ctx, true); err != nil {
		p.log.Error(err, "Can't Get Disk Partitions")
	}
	if p.Performance.Hostinfo, err = host.InfoWithContext(ctx); err != nil {
		p.log.Error(err, "Can't get Host Info")
	}
	p.Poll(ctx)

	web.Web.GetEchoServer().GET("/cpu", p.CPUInfo)
	web.Web.GetEchoServer().GET("/cpu/time", p.CPUTime)
	web.Web.GetEchoServer().GET("/cpu/util", p.CPUUtil)
	web.Web.GetEchoServer().GET("/disk", p.DiskPart)
	web.Web.GetEchoServer().GET("/disk/stats", p.DiskIO)
	web.Web.GetEchoServer().GET("/host", p.HostInfo)
	web.Web.GetEchoServer().GET("/temps", p.Temps)
	web.Web.GetEchoServer().GET("/users", p.Users)
	web.Web.GetEchoServer().GET("/cpu/load", p.Load)
	web.Web.GetEchoServer().GET("/cpu/procs", p.Procs)
	web.Web.GetEchoServer().GET("/memory", p.Memory)
	web.Web.GetEchoServer().GET("/swap", p.Swap)
	web.Web.GetEchoServer().GET("/net", p.NetIF)
	web.Web.GetEchoServer().GET("/net/stats", p.NetStats)

	fixedTimer1second, err := taskmanager.NewFixed(viper.GetDuration("perf.poll") * time.Second)
	if err != nil {
		p.log.Error(err, "invalid interval")
	}
	err = tm.GetScheduler().Add(context.Background(), "Perf", fixedTimer1second, p.Poll)
	if err != nil {
		p.log.Error(err, "Can't Initilize Scheduler for Perf")
	}
	p.log.Info("Added Perf Polling Schedule")
}

func (p *PerfS) Stop() {
	p.mx.Lock()
	defer p.mx.Unlock()
	tm.GetScheduler().Stop("Perf")
}

func (p *PerfS) Poll(ctx context.Context) {
	p.mx.Lock()
	defer p.mx.Unlock()
	var err error
	ctx, cancel := context.WithTimeout(ctx, viper.GetDuration("perf.timeout") * time.Second)
	defer cancel()
	if p.Performance.Cputime, err = cpu.TimesWithContext(ctx, true); err != nil {
		p.log.Error(err, "Can't get CPU Time")
	}
	if p.Performance.Cpuutilization, err = cpu.PercentWithContext(ctx, 1*time.Second, true); err != nil {
		p.log.Error(err, "Can't Get CPU Load")
	}
	if p.Performance.Diskiocounter, err = disk.IOCountersWithContext(ctx); err != nil {
		p.log.Error(err, "Cant get IO Stats")
	}
	if p.Performance.Hosttemp, err = host.SensorsTemperaturesWithContext(ctx); err != nil {
		p.log.Error(err, "Can't Get Temp Stats")
	}
	if p.Performance.Hostusers, err = host.UsersWithContext(ctx); err != nil {
		p.log.Error(err, "Can't get Users")
	}
	if p.Performance.Loadavg, err = load.AvgWithContext(ctx); err != nil {
		p.log.Error(err, "Can't get Load Average")
	}
	if p.Performance.Loadprocs, err = load.MiscWithContext(ctx); err != nil {
		p.log.Error(err, "Can't get Proc Info")
	}
	if p.Performance.Memstats, err = mem.VirtualMemoryWithContext(ctx); err != nil {
		p.log.Error(err, "Can't get Memory Stats")
	}
	if p.Performance.Memswap, err = mem.SwapMemoryWithContext(ctx); err != nil {
		p.log.Error(err, "Can't get Swap Stats")
	}
	if p.Performance.Netstats, err = net.IOCountersWithContext(ctx, true); err != nil {
		p.log.Error(err, "Can't get Network IO Stats")
	}
	if p.Performance.Netaddr, err = net.InterfacesWithContext(ctx); err != nil {
		p.log.Error(err, "Can't get Interface Info")
	}

	internal.ProcessUpdate("Perf", p.Performance)
}

func (p *PerfS) CPUInfo(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Cpuinfo)
}
func (p *PerfS) CPUTime(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
		return c.JSON(http.StatusOK, p.Performance.Cputime)
}
func (p *PerfS) CPUUtil(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Cpuutilization)
}
func (p *PerfS) DiskPart(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Diskpartions)
}
func (p *PerfS) DiskIO(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Diskiocounter)
}
func (p *PerfS) HostInfo(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Hostinfo)
}
func (p *PerfS) Temps(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Hosttemp)
}
func (p *PerfS) Users(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Hostusers)
}
func (p *PerfS) Load(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Loadavg)
}
func (p *PerfS) Procs(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Loadprocs)
}
func (p *PerfS) Memory(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Memstats)
}
func (p *PerfS) Swap(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Memswap)
}
func (p *PerfS) NetStats(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Netstats)
}
func (p *PerfS) NetIF(c echo.Context) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return c.JSON(http.StatusOK, p.Performance.Netaddr)
}
