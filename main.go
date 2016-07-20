package main

import (
	"log"
	"os"
	"time"

	"github.com/prologic/eye/collectors"

	"github.com/shirou/gopsutil/host"

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

	hostinfo, _ := host.Info()

	tags := collectors.Tags{
		"Hostname":             hostinfo.Hostname,
		"OS":                   hostinfo.OS,
		"Platform":             hostinfo.Platform,
		"PlatformFamily":       hostinfo.PlatformFamily,
		"PlatformVersion":      hostinfo.PlatformVersion,
		"VirtualizationSystem": hostinfo.VirtualizationSystem,
		"VirtualizationRole":   hostinfo.VirtualizationRole,
	}

	channels := []<-chan collectors.Data{
		collectors.Collect("cpu", tags, 5, collectors.CPU),
		collectors.Collect("mem", tags, 5, collectors.Mem),
		collectors.Collect("disk", tags, 30, collectors.Disk),
		collectors.Collect("load", tags, 5, collectors.Load),
		collectors.Collect("procs", tags, 10, collectors.Procs),
		collectors.Collect("users", tags, 60, collectors.Users),
		collectors.Collect("uptime", tags, 60, collectors.Uptime),
	}

	for data := range collectors.Merge(channels) {
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
