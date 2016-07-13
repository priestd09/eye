package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	_ "github.com/shirou/gopsutil/net"
	_ "github.com/shirou/gopsutil/process"

	"github.com/influxdata/influxdb/client/v2"
)

var buildCommit string

func main() {
	url := os.Getenv("INFLUX_URL")

	database := os.Getenv("INFLUX_DATABASE")
	if database == "" {
		database = "eye"
	}

	username := os.Getenv("INFLUX_USERNAME")
	if username == "" {
		username = "root"
	}

	password := os.Getenv("INFLUX_PASSWORD")
	if password == "" {
		password = "root"
	}

	clientConfig := client.HTTPConfig{
		Addr:     url,
		Username: username,
		Password: password,
	}

	batchConfig := client.BatchPointsConfig{
		Database:  database,
		Precision: "s",
	}

	for {
		c, err := client.NewHTTPClient(clientConfig)
		if err != nil {
			log.Fatalln("Error: ", err)
		}

		bp, err := client.NewBatchPoints(batchConfig)
		if err != nil {
			log.Fatalln("Error: ", err)
		}

		hostinfo, _ := host.Info()

		tags := map[string]string{
			"Hostname":             hostinfo.Hostname,
			"OS":                   hostinfo.OS,
			"Platform":             hostinfo.Platform,
			"PlatformFamily":       hostinfo.PlatformFamily,
			"PlatformVersion":      hostinfo.PlatformVersion,
			"VirtualizationSystem": hostinfo.VirtualizationSystem,
			"VirtualizationRole":   hostinfo.VirtualizationRole,
		}

		fields := make(map[string]interface{})

		// Uptime
		fields["uptime"] = hostinfo.Uptime

		// Processes
		fields["procs"] = hostinfo.Procs

		// Users
		users, _ := host.Users()
		fields["users"] = len(users)

		// CPU
		cpu, _ := cpu.Percent(0, true)
		for i := 0; i < len(cpu); i++ {
			fields[fmt.Sprintf("cpu%d.util", i)] = cpu[i]
		}

		// Memory
		mem, _ := mem.VirtualMemory()
		fields["mem.total"] = mem.Total
		fields["mem.avail"] = mem.Available
		fields["mem.free"] = mem.Free

		// Load
		load, _ := load.Avg()
		fields["load.1m"] = load.Load1
		fields["load.5m"] = load.Load5
		fields["load.15m"] = load.Load15

		// Disk
		partitions, _ := disk.Partitions(false)
		for i := range partitions {
			partition := partitions[i]
			parts := strings.Split(partition.Device, "/")
			diskName := parts[len(parts)-1]
			usage, _ := disk.Usage(partition.Mountpoint)
			fields[fmt.Sprintf("disk.%s.total", diskName)] = usage.Total
			fields[fmt.Sprintf("disk.%s.used", diskName)] = usage.Used
			fields[fmt.Sprintf("disk.%s.free", diskName)] = usage.Free
		}

		pt, err := client.NewPoint("system", tags, fields, time.Now())
		if err != nil {
			log.Fatalln("Error: ", err)
		}

		bp.AddPoint(pt)

		c.Write(bp)

		c.Close()

		time.Sleep(5 * time.Second)
	}
}
