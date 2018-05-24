package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

var firstTime time.Time
var lastTime time.Time

var tf *string

var groupRe *regexp.Regexp

func main() {
	log.SetFlags(log.Lshortfile)
	f := flag.String("f", "", "Time format - %T %C %H %M %S %m %d %y %b."+
		"\n\tMore precise format can be given via -re and -tf")
	regex := flag.String("re", "", "Regex to extract date and time")
	tf = flag.String("tf", "", "Time format")
	durationField := flag.Int("field", -1, "Duration field")

	flag.Parse()

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

	re := regexp.MustCompile(*regex)

	ret := 0
	files := []string{"-"}
	if len(flag.Args()) > 0 {
		files = flag.Args()
	}

	var src io.Reader

	var tfmt string = "2006/01/02 15:04:05"

	fmt.Println("date time duration(ms)")
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
			line := r.Text()
			lnum++
			m := re.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			tstamp := m[1]
			t, err := time.Parse(*tf, tstamp)
			if err != nil {
				log.Fatal(err)
			}
			normalizedTstamp := t.Format(tfmt)
			chunks := strings.Split(line, " ")
			//log.Printf("len(chunks) %d [%d]", len(chunks), *durationField-1)
			if len(chunks) < *durationField {
				log.Printf("corrupt line? [%s]", line)
				continue
			}
			str := chunks[*durationField-1]
			d, err := time.ParseDuration(str)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s %d\n", normalizedTstamp, d.Nanoseconds()/10000000)
		}
	}

	os.Exit(ret)
}
