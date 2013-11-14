package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/holygeek/piper"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
	"strconv"
)

var fan = []byte{'|', '/', '-', '\\', '|', '/', '-', '\\'}
var idx = 0
var startTime int64

var usage = `NAME
  linefan - show spinning fan

SYNOPSIS
   linefan -h
   linefan [-e N] [-c] [-d N] [-t N] [-R <file>] [-r] [-T Title] [-P] [<args...>]

DESCRIPTION
  If <args> is given, linefan executes <args...> using /bin/sh and display a
  fan leaf each time a stdout line is read and when the shell exits linefan
  stores the run metadata - duration, line count and output in .linefan
  directory.  Next invocation with the same argument in the same directory will
  use the same metadata for estimating the completion time.

  Without <args> linefan read lines from stdin and no metadata is saved or
  used unless explicitly requested using the options.

OPTIONS`

func main() {
	chdir := flag.String(
		"C", "", docStr(
		"Change to given directory before doing anything else"))
	clean := flag.Bool(
		"c", false, docStr(
		"Clear output when done. Leave no fan trace."))
	duration := flag.Int64(
		"d", 0, docStr(
		"Show estimated time remaining based on N seconds total",
		"runtime. Setting N to 0 turns off the remaining time",
		"estimation. If runtime exceeds N, ?s is shown."))
	freq := flag.Int(
		"e", 1, docStr("Fan speed. Lower is faster."))
	echo := flag.Bool(
		"P", false, docStr(
		"Echo whatever was read from stdin to stdout."))
	quiet := flag.Bool(
		"q", false, docStr(
		"Do not show fan."))
	record := flag.String(
		"R", "", docStr(
		"If file does not exist, record the duration and number of",
		"lines read from stdin into <file>. If file exist, use the",
		"values in <file> for the -t and -d arguments."))
	saveNewRecord := flag.Bool(
		"r", false, docStr(
		"When done, record the duration and number of",
		"lines read from stdin into <file> given by -R."))
	tLines := flag.Int(
		"t", 0, docStr(
		"Show estimated completion percentage based on N lines of max",
		"input. Setting N to 0 turns off the percentage estimation. If",
		"input lines is more than N, ?% will be shown instead."))
	title := flag.String(
		"T", "", docStr(
		"Print given title before the fan. If title is '-', and <args>",
	        "is supplied, then use <args> as the title"))

	flag.Usage = func() {
		fmt.Println(usage)
		flag.PrintDefaults()
	}

	flag.Parse()
	if (*chdir != "") {
		err := os.Chdir(*chdir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}

	lastLen := 0
	nLines := 0

	cmd := ""
	var proc *exec.Cmd
	var in, stderr *bufio.Scanner
	if flag.NArg() > 0 {
		cmd = strings.Join(flag.Args(), " ")
		if *record == "" {
			*record = safeFileName(cmd)
		}
		proc, in, stderr = piper.MustPipe("/bin/sh", "-c", cmd)
		go func() {
			for stderr.Scan() {
				fmt.Println(stderr.Text())
			}
		}()
	} else {
		in = bufio.NewScanner(os.Stdin)
	}

	if *record != "" {
		content := readFile(*record)
		*duration, *tLines = readRecord(&content)
	}

	startTime = time.Now().Unix()
	if *title != "" {
		*title = chooseAndFormatTitle(*title, cmd)
		fanOut(*title)
	}
	var buf string
	for in.Scan() {
		buf = in.Text()
		nLines++
		if *echo {
			fanOut(buf + "\n")
		}
		if *quiet {
			continue
		}
		if (nLines - 1) % *freq != 0 {
			continue
		}

		for i := 0; i < lastLen; i++ {
			fanOut("\b")
		}

		str := getFanText(*duration, nLines, *tLines)
		fanOut(str)
		newLen := len(str)
		if newLen < lastLen {
			// Clear trailing garbage from previous output
			for i := newLen; i < lastLen; i++ {
				fanOut(" ")
			}
			for i := newLen; i < lastLen; i++ {
				fanOut("\b")
			}
		}

		lastLen = newLen
		if *echo {
			fanOut("\n")
		}
	}
	timeTaken := time.Now().Unix() - startTime

	if ! *quiet {
		if *clean {
			cleanFan(lastLen + len(*title))
		} else {
			fanOut("\n")
		}
	}

	if *record != "" {
		_, err := os.Stat(*record)
		if *saveNewRecord || os.IsNotExist(err) {
			createFanRecord(*record, timeTaken, nLines)
		}
	}

	ret := 0
	if proc != nil {
		if proc.Wait() != nil {
			ret = 1
		}
	}
	os.Exit(ret)
}

func chooseAndFormatTitle(titleArg, cmd string) string {
  if titleArg == "-" && cmd != "" {
    titleArg = cmd
  }
  return titleArg + " "
}

func safeFileName(cmd string) string {
	createDir(".linefan")
	fsSafe := func(r rune) rune {
		switch {
		case r >= ' ' && r <= '.':
		return r
		case r >= '0' && r <= '~':
		return r
		}
		return '_'
	}
	return strings.Join([]string{".linefan", strings.Map(fsSafe, cmd)}, "/")
}
func createDir(name string) {
	os.MkdirAll(name, os.ModeDir|os.ModePerm)
}

func fanOut(str string) {
	fmt.Fprint(os.Stderr, str)
}

func createFanRecord(record string, timeTaken int64, nLines int) {
	str := fmt.Sprintf("duration=%d\ntarget=%d\n", timeTaken, nLines)
	if err := ioutil.WriteFile(record, []byte(str), 0644); err != nil {
		fmt.Println("linefan:", err)
	}
}

func cleanFan(l int) {
	for i := 0; i < l; i++ {
		fanOut("\b \b")
	}
}

func getFanText(duration int64, nLines, tLines int) string {
	str := fmt.Sprintf("%c", fan[idx])
	lapsed := float32(time.Now().Unix() - startTime)

	if tLines > 0 {
		percent := float64(nLines * 100.0 / tLines)
		if percent <= 100 {
			str += fmt.Sprintf(" %3.0f%%", percent)
		} else {
			str += "   ?%"
		}
		if duration == 0 && lapsed > 0 {
			// estimate remaining time based on line velocity
			velocity := float32(nLines) / lapsed
			remLine := tLines - nLines
			// Add one second to offset the last second
			if velocity > 0 {
				eta := int64(float32(remLine) / velocity + 1)
				str = fmt.Sprintf("%s (%s)", str, textTime(eta))
			}
		}
	}

	if duration > 0 {
		// Calculate remaining time based on given duration
		remTime := duration - int64(lapsed)
		if remTime >= 0 {
			str += fmt.Sprintf(" %s", textTime(remTime))
		} else {
			str += " ?"
		}
	}

	idx++
	if idx >= len(fan) {
		idx = idx % len(fan);
	}

	return str
}

func textTime(delta int64) string {
	const year_text = 'y'
	const day_text = 'd'
	const hour_text = 'h'
	const minute_text = 'm'
	const second_text = 's'

	const SECONDS_PER_MINUTE = 60
	const SECONDS_PER_HOUR = 60 * SECONDS_PER_MINUTE
	const SECONDS_PER_DAY = 24 * SECONDS_PER_HOUR
	const SECONDS_PER_YEAR = 365 * SECONDS_PER_DAY

	if delta == 0 {
		return "0s"
	}

	years := delta / SECONDS_PER_YEAR
	delta = delta % SECONDS_PER_YEAR

	days := delta / SECONDS_PER_DAY
	delta = delta % SECONDS_PER_DAY

	hours := delta / SECONDS_PER_HOUR
	delta = delta % SECONDS_PER_HOUR

	minutes := delta / SECONDS_PER_MINUTE
	delta = delta % SECONDS_PER_MINUTE

	seconds := delta

	var timeChunk [5]string
	idx := 0

	if years > 0 {
		timeChunk[idx] = fmt.Sprintf("%d%c", years, year_text)
		idx++
	}
	if idx > 0 || days > 0 {
		timeChunk[idx] = fmt.Sprintf("%d%c", days, day_text)
		idx++
	}
	if idx > 0 || hours > 0 {
		timeChunk[idx] = fmt.Sprintf("%d%c", hours, hour_text)
		idx++
	}
	if idx > 0 || minutes > 0 {
		timeChunk[idx] = fmt.Sprintf("%2d%c", minutes, minute_text)
		idx++
	}
	if idx > 0 || seconds > 0 {
		fmtStr := "%2d%c"
		if idx == 0 {
			fmtStr = "%d%c"
		}
		timeChunk[idx] = fmt.Sprintf(fmtStr, seconds, second_text)
		idx++
	}

	return strings.Join(timeChunk[0:idx], " ")
}

func readRecord(content *string) (duration int64, nLines int) {
	duration, nLines = 0, 0
	for _, token := range strings.Fields(*content) {
		pair := strings.Split(token, "=")
		if len(pair) == 2 {
			if pair[0] == "duration" {
				d, err := strconv.Atoi(pair[1])
				if err == nil {
					duration = int64(d)
				} else {
					duration = 0
				}
			}
			if pair[0] == "target" {
				value, err := strconv.Atoi(pair[1])
				if err != nil {
					nLines = value
				}
			}
		}
	}
	return
}

func readFile(filename string) string {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return ""
	}
	return string(buf)
}

func docStr(text ...string) string {
	return strings.Join(text, "\n\t") + "\n"
}
