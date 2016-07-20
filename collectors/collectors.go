package collectors

import (
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

// Tags ...
type Tags map[string]string

// Fields ...
type Fields map[string]interface{}

// Data ...
type Data struct {
	Name   string
	Tags   Tags
	Fields Fields
}

// Collector ...
type Collector func() (field Fields, err error)

// CPU ...
func CPU() (fields Fields, err error) {
	cpu, err := cpu.Percent(0, true)
	if err != nil {
		return nil, err
	}
	fields = make(Fields)
	for i := 0; i < len(cpu); i++ {
		fields[fmt.Sprintf("cpu%d.util", i)] = cpu[i]
	}
	return
}

// Mem ...
func Mem() (fields Fields, err error) {
	mem, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	return Fields{
		"total": mem.Total,
		"avail": mem.Available,
		"free":  mem.Free,
	}, nil
}

// Uptime ...
func Uptime() (fields Fields, err error) {
	hostinfo, err := host.Info()
	if err != nil {
		return nil, err
	}
	return Fields{"value": hostinfo.Uptime}, nil
}

// Procs ...
func Procs() (fields Fields, err error) {
	hostinfo, err := host.Info()
	if err != nil {
		return nil, err
	}
	return Fields{"value": hostinfo.Procs}, nil
}

// Users ...
func Users() (fields Fields, err error) {
	users, err := host.Users()
	if err != nil {
		return nil, err
	}
	return Fields{"value": len(users)}, nil
}

// Load ...
func Load() (fields Fields, err error) {
	load, err := load.Avg()
	if err != nil {
		return nil, err
	}
	return Fields{
		"1m":  load.Load1,
		"5m":  load.Load5,
		"15m": load.Load15,
	}, nil
}

// Disk ...
func Disk() (fields Fields, err error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}
	fields = make(Fields)
	for i := range partitions {
		partition := partitions[i]
		parts := strings.Split(partition.Device, "/")
		diskName := parts[len(parts)-1]
		usage, _ := disk.Usage(partition.Mountpoint)
		fields[fmt.Sprintf("%s.total", diskName)] = usage.Total
		fields[fmt.Sprintf("%s.used", diskName)] = usage.Used
		fields[fmt.Sprintf("%s.free", diskName)] = usage.Free
	}
	return
}
