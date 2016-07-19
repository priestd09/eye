package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
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

type Tags map[string]string
type Fields map[string]interface{}

type Data struct {
	Name   string
	Tags   Tags
	Fields Fields
}

type Collector func() (field Fields, err error)

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

func Uptime() (fields Fields, err error) {
	hostinfo, err := host.Info()
	if err != nil {
		return nil, err
	}
	return Fields{"value": hostinfo.Uptime}, nil
}

func Procs() (fields Fields, err error) {
	hostinfo, err := host.Info()
	if err != nil {
		return nil, err
	}
	return Fields{"value": hostinfo.Procs}, nil
}

func Users() (fields Fields, err error) {
	users, err := host.Users()
	if err != nil {
		return nil, err
	}
	return Fields{"value": len(users)}, nil
}

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

func collect(name string, tags Tags, interval int, fn Collector) chan Data {
	ch := make(chan Data)

	go func(ch chan<- Data) {
		for {
			fields, err := fn()
			if err == nil {
				ch <- Data{
					Name:   name,
					Tags:   tags,
					Fields: fields,
				}
			}
			time.Sleep(time.Second * time.Duration(interval))
		}
	}(ch)

	return ch
}

func merge(cs []<-chan Data) <-chan Data {
	var wg sync.WaitGroup

	out := make(chan Data)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan Data) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}

	wg.Add(len(cs))

	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

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

	collectors := []<-chan Data{
		collect("cpu", tags, 5, CPU),
		collect("mem", tags, 5, Mem),
		collect("disk", tags, 30, Disk),
		collect("load", tags, 5, Load),
		collect("procs", tags, 10, Procs),
		collect("users", tags, 60, Users),
		collect("uptime", tags, 60, Uptime),
	}

	for data := range merge(collectors) {
		c, err := client.NewHTTPClient(clientConfig)
		if err != nil {
			log.Fatalln("Error: ", err)
		}

		bp, err := client.NewBatchPoints(batchConfig)
		if err != nil {
			log.Fatalln("Error: ", err)
		}

		pt, err := client.NewPoint(
			data.Name, data.Tags, data.Fields, time.Now(),
		)
		if err != nil {
			log.Fatalln("Error: ", err)
		}

		bp.AddPoint(pt)

		c.Write(bp)

		c.Close()
	}
}
