package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/VividCortex/gohistogram"
	tm "github.com/buger/goterm"
)

var firstTime time.Time
var lastTime time.Time

var tf *string

var groupRe *regexp.Regexp

func main() {
	log.SetFlags(log.Lshortfile)
	var (
		optFilterDuration time.Duration
		optHist           bool
		line              string
	)

	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic caught: %s\n%s\n%s", err, line, debug.Stack())
		}
	}()

	flag.DurationVar(&optFilterDuration, "min", -1, "Filter mode: Print log lines taking more than `duration`")
	flag.BoolVar(&optHist, "hist", false, "show histogram instead (for -min)")
	f := flag.String("f", "", "Time format - %T %C %H %M %S %m %d %y %b."+
		"\n\tMore precise format can be given via -re and -tf")
	regex := flag.String("re", "", "Regex to extract date and time")
	tf = flag.String("tf", "", "Time format")
	durationField := flag.Int("field", -1, "Duration field")
	plot := flag.Bool("plot", false, "Plot data in terminal")

	flag.Parse()

	if optFilterDuration == -1 {
		if *f != "" {
			*regex = "(" + *f + ")"
			*tf = *f
		}
		if *regex == "" {
			fmt.Fprintf(os.Stderr, "Time regex must not be empty (-re <regexp)\n")
			os.Exit(1)
		}
		if *tf == "" {
			fmt.Fprintf(os.Stderr, "Time format must not be empty (-tf <format)\n")
			os.Exit(1)
		}
		if *regex != "" {
			*regex = strings.Replace(*regex, "%Z", `[A-Z][A-Z][A-Z][A-Z]?`, -1)
			*regex = strings.Replace(*regex, "%z", `[+-]\d\d\d\d`, -1)
			*regex = strings.Replace(*regex, "%b", `[A-Z][a-z][a-z]`, -1)
			*regex = strings.Replace(*regex, "%T", `\d\d:\d\d:\d\d`, -1)
			*regex = strings.Replace(*regex, "%C", `\d\d`, -1)
			*regex = strings.Replace(*regex, "%F", `\d\d\d\d-\d\d-\d\d`, -1)
			*regex = strings.Replace(*regex, "%H", `\d\d`, -1)
			*regex = strings.Replace(*regex, "%M", `\d\d`, -1)
			*regex = strings.Replace(*regex, "%S", `\d\d(?:\.\d*)?`, -1)
			*regex = strings.Replace(*regex, "%m", `\d\d`, -1)
			*regex = strings.Replace(*regex, "%d", `\d\d`, -1)
			*regex = strings.Replace(*regex, "%Y", `\d\d\d\d`, -1)
		}

		if *tf != "" {
			*tf = strings.Replace(*tf, "%Z", "MST", -1)
			*tf = strings.Replace(*tf, "%z", "-0700", -1)
			*tf = strings.Replace(*tf, "%b", "Jan", -1)
			*tf = strings.Replace(*tf, "%T", "15:04:05", -1)
			*tf = strings.Replace(*tf, "%C", `06`, -1)
			*tf = strings.Replace(*tf, "%F", `2006-01-02`, -1)
			*tf = strings.Replace(*tf, "%H", `15`, -1)
			*tf = strings.Replace(*tf, "%M", `04`, -1)
			// fractional seconds
			//*tf = strings.Replace(*tf, "%S", `05.9`, -1)
			*tf = strings.Replace(*tf, "%S", `05`, -1)
			*tf = strings.Replace(*tf, "%m", `01`, -1)
			*tf = strings.Replace(*tf, "%d", `02`, -1)
			*tf = strings.Replace(*tf, "%Y", `2006`, -1)
		}
	}

	var re *regexp.Regexp
	if optFilterDuration == -1 {
		re = regexp.MustCompile(*regex)
	}

	ret := 0
	files := []string{"-"}
	if len(flag.Args()) > 0 {
		files = flag.Args()
	}

	var src io.Reader

	var tfmt string = "2006/01/02 15:04:05"

	var table *tm.DataTable
	var chart *tm.LineChart
	if *plot {
		table = &tm.DataTable{}
		chart = tm.NewLineChart(100, 20)
		//chart.Flags = tm.DRAW_INDEPENDENT
		table.AddColumn("Time")
		table.AddColumn("Duration ms")
	} else {
		if optFilterDuration == -1 {
			fmt.Println("date time duration(ms)")
		}
	}

	var hist gohistogram.Histogram
	if optHist && optFilterDuration != -1 {
		hist = gohistogram.NewHistogram(20)
	}
	for _, file := range files {
		if file == "-" {
			src = os.Stdin
		} else {
			f, err := os.Open(file)
			if err != nil {
				log.Println(err)
				ret = 1
				continue
			}
			src = f
		}

		lnum := 0
		r := bufio.NewScanner(src)
		for r.Scan() {
			line = r.Text()
			lnum++
			var t time.Time
			var normalizedTstamp string
			var err error

			if optFilterDuration == -1 {
				m := re.FindStringSubmatch(line)
				if m == nil {
					continue
				}
				tstamp := m[1]
				t, err = time.Parse(*tf, tstamp)
				if err != nil {
					log.Fatal(err)
				}
				normalizedTstamp = t.Format(tfmt)
			}
			chunks := strings.Fields(line)
			//log.Printf("len(chunks) %d [%d]", len(chunks), *durationField-1)
			if len(chunks) < *durationField {
				log.Printf("corrupt line? [%s]", line)
				continue
			}
			str := chunks[*durationField-1]
			d, err := time.ParseDuration(strings.Trim(str, ":"))
			if err != nil {
				log.Printf("%s: %s", err, line)
				continue
			}

			if optFilterDuration == -1 {
				ms := d.Nanoseconds() / 1000000
				if *plot {
					table.AddRow(float64(t.Unix()), float64(ms))
				} else {
					fmt.Printf("%s %d\n", normalizedTstamp, ms)
				}
			} else {
				if d > optFilterDuration {
					if optHist {
						hist.Add(d.Seconds())
					} else {
						fmt.Printf("%s\n", line)
						os.Stdout.Sync()
					}
				}
			}
		}
	}
	if optHist && optFilterDuration != -1 {
		fmt.Printf("%s\n", hist)
	}
	if *plot {
		tm.Println(chart.Draw(table))
		tm.Flush()
	}

	os.Exit(ret)
}
