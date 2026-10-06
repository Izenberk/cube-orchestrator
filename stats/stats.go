package stats

import (
	"log"
	"github.com/c9s/goprocinfo/linux"
)

type Stats struct {
	MemStats		*linux.MemInfo
	DiskStats		*linux.Disk
	CpuStats		*linux.CPUStat
	LoadStats		*linux.LoadAvg
	TaskCount 	int
}

type CPUStat struct {
	Id 					string	`json:"id"`
	User 				uint64	`json:"user"`
	Nice				uint64	`json:"system"`
	Idle				uint64	`json:"idle"`
	IOWait			uint64	`json:"iowait"`
	IRQ					uint64	`json:"irq"`
	SoftIRQ			uint64	`json:"softirq"`
	Steal				uint64	`json:"steal"`
	Guest				uint64	`json:"guest"`
	GuestNice		uint64	`json:"guest_nice"`
}

// Memory metrics
func (s *Stats) MemTotalKb() uint64 {
	return s.MemStats.MemTotal
}

func (s *Stats) MemAvailableKb() uint64 {
	return s.MemStats.MemAvailable
}

func (s *Stats) MemUsedKb() uint64 {
	return s.MemStats.MemTotal - s.MemStats.MemAvailable
}

func (s *Stats) MemUsedPercent() uint64 {
	return s.MemStats.MemAvailable / s.MemStats.MemTotal
}

// Disk metrics
func (s *Stats) DiskTotal() uint64 {
	return s.DiskStats.All
}

func (s *Stats) DiskFree() uint64 {
	return s.DiskStats.Free
}

func (s *Stats) DiskUsed() uint64 {
	return s.DiskStats.Used
}

// CPU metrics
func (s *Stats) CpuUsage() float64 {
	idle := s.CpuStats.Idle + s.CpuStats.IOWait
	nonIdle := s.CpuStats.User + s.CpuStats.Nice + s.CpuStats.System +
		s.CpuStats.IRQ + s.CpuStats.SoftIRQ + s.CpuStats.Steal
	total := idle + nonIdle

	if total == 0 {
		return 0.00
	}

	return (float64(total) - float64(idle)) / float64(total)
}

func GetStats() *Stats {
	return &Stats{
		MemStats:		GetmemoryInfo(),
		DiskStats: 	GetDiskInfo(),
		CpuStats: 	GetCpuStats(),
		LoadStats: 	GetLoadAvg(),
	}
}

func GetmemoryInfo() *linux.MemInfo {
	memstats, err := linux.ReadMemInfo("/proc/meminfo")
	if err != nil {
		log.Printf("Error reading from /proc/meminfo")
		return &linux.MemInfo{}
	}

	return memstats
}

func GetDiskInfo() *linux.Disk {
	diskstats, err := linux.ReadDisk("/")
	if err != nil {
		log.Printf("Error reading from /")
		return &linux.Disk{}
	}

	return diskstats
}

func GetCpuStats() *linux.CPUStat {
	stats, err := linux.ReadStat("/proc/stat")
	if err != nil {
		log.Printf("Error reading from /proc/stat")
		return &linux.CPUStat{}
	}

	return &stats.CPUStatAll
}

func GetLoadAvg() *linux.LoadAvg{
	loadavg, err := linux.ReadLoadAvg("/proc/loadavg")
	if err != nil {
		log.Printf("Error reading from /proc/loadavg")
		return &linux.LoadAvg{}
	}

	return loadavg
}