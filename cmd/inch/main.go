package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/bonitoo-io/inch"
	"log"
	"runtime/pprof"
)

// Main represents the main program execution.
type Main struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	inch *inch.Simulator
	cpuProfileFile string
}

func main() {
	m := NewMain()

	// parse command line flags
	if err := m.ParseFlags(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if m.cpuProfileFile != "" {
		f, err := os.Create(m.cpuProfileFile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// run inch
	if err := m.inch.Run(context.Background()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		inch:   inch.NewSimulator(),
	}
}

func (m *Main) ParseFlags(args []string) error {
	// ensure we have an inch
	if m.inch == nil {
		m.inch = inch.NewSimulator()
	}

	// set the output information
	m.inch.Stdout = os.Stdout
	m.inch.Stderr = os.Stderr

	fs := flag.NewFlagSet("inch", flag.ContinueOnError)
	fs.BoolVar(&m.inch.Verbose, "v", false, "Verbose")
	fs.StringVar(&m.inch.ReportHost, "report-host", "", "Host to send metrics")
	fs.StringVar(&m.inch.ReportUser, "report-user", "", "User for Host to send metrics")
	fs.StringVar(&m.inch.ReportPassword, "report-password", "", "Password Host to send metrics")
	reportTags := fs.String("report-tags", "", "Comma separated k:v tags to report alongside metrics")
	fs.BoolVar(&m.inch.DryRun, "dry", false, "Dry run (maximum writer perf of inch on box)")
	fs.IntVar(&m.inch.MaxErrors, "max-errors", 0, "Terminate process if this many errors encountered")
	hosts := fs.String( "hosts", "http://localhost:8086", "Comma-separated list of hosts. Will be used in round-robin fashion")
	fs.StringVar(&m.inch.User, "user", "", "Host User")
	fs.StringVar(&m.inch.Password, "password", "", "Host Password")
	fs.StringVar(&m.inch.Consistency, "consistency", "any", "Write consistency (default any)")
	fs.IntVar(&m.inch.Concurrency, "c", 1, "Concurrency")
	fs.IntVar(&m.inch.Measurements, "m", 1, "Measurements")
	tags := fs.String("t", "10,10,10", "Tag cardinality")
	fs.IntVar(&m.inch.TagSize, "tag-size", -1, "Minimum size of tag value (default same as number of digits of value)")
	fs.IntVar(&m.inch.PointsPerSeries, "p", 100, "Points per series")
	fs.IntVar(&m.inch.FieldsPerPoint, "f", 1, "Fields per point")
	fs.IntVar(&m.inch.FieldSize, "field-size", 1, "Size (in digits) of field value (default 1)")
	fs.IntVar(&m.inch.BatchSize, "b", 5000, "Batch size")
	fs.StringVar(&m.inch.Database, "db", "stress", "Database to write to")
	fs.StringVar(&m.inch.ShardDuration, "shard-duration", "7d", "Set shard duration (default 7d)")
	fs.StringVar(&m.inch.StartTime, "start-time", "", "Set start time (default now)")
	fs.DurationVar(&m.inch.TimeSpan, "time", 0, "Time span to spread writes over")
	fs.DurationVar(&m.inch.Delay, "delay", 0, "Delay between writes")
	fs.IntVar(&m.inch.DatabaseCreationTimeout, "db-creation-timeout", 5, "Max time, in sec, to wait for database is created")
	fs.DurationVar(&m.inch.TargetMaxLatency, "target-latency", 0, "If set inch will attempt to adapt write delay to meet target")
	fs.StringVar(&m.cpuProfileFile, "cpuprofile", "", "write cpu profile to file")

	if err := fs.Parse(args); err != nil {
		return err
	}

	m.inch.Hosts = strings.Split(*hosts,",")
	if len(m.inch.Hosts) == 0 {
		return fmt.Errorf("cannot partse hosts: %s", *hosts)
	}

	// Parse tag cardinalities.
	m.inch.Tags = []int{}
	for _, s := range strings.Split(*tags, ",") {
		v, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("cannot parse tag cardinality: %s", s)
		}
		m.inch.Tags = append(m.inch.Tags, v)
	}

	// Basic report tags.
	m.inch.ReportTags = map[string]string{
		"stress_tool": "inch",
		"t":           *tags,
		"tag_size":	   fmt.Sprint(m.inch.TagSize),
		"batch_size":  fmt.Sprint(m.inch.BatchSize ),
		"p":           fmt.Sprint(m.inch.PointsPerSeries),
		"c":           fmt.Sprint(m.inch.Concurrency),
		"m":           fmt.Sprint(m.inch.Measurements),
		"f":           fmt.Sprint(m.inch.FieldsPerPoint),
		"field_size":  fmt.Sprint(m.inch.FieldSize),
		"sd":          m.inch.ShardDuration,
	}

	// Parse report tags.
	if *reportTags != "" {
		for _, tagPair := range strings.Split(*reportTags, ",") {
			tag := strings.Split(tagPair, ":")
			if len(tag) != 2 {
				return fmt.Errorf("invalid tag pair %q", tagPair)
			}
			m.inch.ReportTags[tag[0]] = tag[1]
		}
	}

	return nil
}
