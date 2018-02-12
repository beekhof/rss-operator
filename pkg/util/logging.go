// Copyright 2017 The etcd-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"encoding/json"
	"strings"

	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	//	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

type RssLogger struct {
	*logrus.Entry
}

// PresentIn returns true if the given string is part of an array of strings
func PresentIn(a string, list []string) bool {
	for _, l := range list {
		if a == l {
			return true
		}
	}
	return false
}

func LogOutput(logger *RssLogger, level logrus.Level, id string, result string) {
	if result != "" {
		lines := strings.Split(result, "\n")
		for n, l := range lines {
			switch level {
			case logrus.DebugLevel:
				logger.WithField("pod", id).Debugf("[%v][%v]", n, l)
			case logrus.InfoLevel:
				logger.WithField("pod", id).Infof("[%v][%v]", n, l)
			case logrus.WarnLevel:
				logger.WithField("pod", id).Warnf("[%v][%v]", n, l)
			case logrus.ErrorLevel:
				logger.WithField("pod", id).Errorf("[%v][%v]", n, l)
			case logrus.FatalLevel:
				logger.WithField("pod", id).Fatalf("[%v][%v]", n, l)
			case logrus.PanicLevel:
				logger.WithField("pod", id).Panicf("[%v][%v]", n, l)
			}

		}
	}
}

var (
	rootLogger  = logrus.New()
	dummyLogger *RssLogger
	once        sync.Once
)

func GetLogger(component string) *RssLogger {
	once.Do(func() {
		f := new(TextFormatter)
		f.ForceFormatting = true
		f.FullTimestamp = true
		f.ForceColors = true

		rootLogger.Formatter = f
		rootLogger.Level = logrus.DebugLevel

		dummyLogger = &RssLogger{logrus.New().WithField("c", "default")}
		dummyLogger.Level = logrus.PanicLevel
	})

	return &RssLogger{rootLogger.WithField("c", component)}
}

func (entry *RssLogger) WithField(name, value string) *RssLogger {
	return &RssLogger{entry.Entry.WithField(name, value)}
}

func (entry *RssLogger) Debug(args ...interface{}) {
	if entry.Data["c"] == "default" {
		return
	}
	entry.V(1).Entry.Debug(args...)
}

func (entry *RssLogger) Debugf(format string, args ...interface{}) {
	if entry.Data["c"] == "default" {
		return
	}
	entry.V(1).Entry.Debug(fmt.Sprintf(format, args...))
}

func (entry *RssLogger) V(level int) *RssLogger {
	if !flag.Parsed() {
		flag.Parse()
	}
	//	logrus.Infof("log check %v %v %v", glog.V(glog.Level(level)), rootLogger.Level, dummyLogger.Level)

	if flag.Parsed() && glog.V(glog.Level(level)) == glog.Verbose(true) {
		return entry
	}
	return dummyLogger
}

func JsonLogObject(logger *RssLogger, spec interface{}, text string) {
	specBytes, err := json.MarshalIndent(spec, "", "    ")
	if err != nil {
		logger.Errorf("failed to marshal spec for '%v': %v", text, err)
	}

	logger.Debug(text)
	for _, m := range strings.Split(string(specBytes), "\n") {
		logger.Debug(m)
	}
}

const defaultTimestampFormat = time.RFC3339

var (
	baseTimestamp      time.Time    = time.Now()
	defaultColorScheme *ColorScheme = &ColorScheme{
		InfoLevelStyle:  "black",
		WarnLevelStyle:  "yellow",
		ErrorLevelStyle: "red",
		FatalLevelStyle: "red",
		PanicLevelStyle: "red",
		DebugLevelStyle: "blue",
		PrefixStyle:     "cyan",
		TimestampStyle:  "black+h",
	}
	noColorsColorScheme *compiledColorScheme = &compiledColorScheme{
		InfoLevelColor:  ansi.ColorFunc(""),
		WarnLevelColor:  ansi.ColorFunc(""),
		ErrorLevelColor: ansi.ColorFunc(""),
		FatalLevelColor: ansi.ColorFunc(""),
		PanicLevelColor: ansi.ColorFunc(""),
		DebugLevelColor: ansi.ColorFunc(""),
		PrefixColor:     ansi.ColorFunc(""),
		TimestampColor:  ansi.ColorFunc(""),
	}
	defaultCompiledColorScheme *compiledColorScheme = compileColorScheme(defaultColorScheme)
)

func miniTS() int {
	return int(time.Since(baseTimestamp) / time.Second)
}

type ColorScheme struct {
	InfoLevelStyle  string
	WarnLevelStyle  string
	ErrorLevelStyle string
	FatalLevelStyle string
	PanicLevelStyle string
	DebugLevelStyle string
	PrefixStyle     string
	TimestampStyle  string
}

type compiledColorScheme struct {
	InfoLevelColor  func(string) string
	WarnLevelColor  func(string) string
	ErrorLevelColor func(string) string
	FatalLevelColor func(string) string
	PanicLevelColor func(string) string
	DebugLevelColor func(string) string
	PrefixColor     func(string) string
	TimestampColor  func(string) string
}

type TextFormatter struct {
	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors bool

	// Force disabling colors. For a TTY colors are enabled by default.
	DisableColors bool

	// Force formatted layout, even for non-TTY output.
	ForceFormatting bool

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Disable the conversion of the log levels to uppercase
	DisableUppercase bool

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool

	// Timestamp format to use for display when a full timestamp is printed.
	TimestampFormat string

	// The fields are sorted by default for a consistent output. For applications
	// that log extremely frequently and don't use the JSON formatter this may not
	// be desired.
	DisableSorting bool

	// Wrap empty fields in quotes if true.
	QuoteEmptyFields bool

	// Can be set to the override the default quoting character "
	// with something else. For example: ', or `.
	QuoteCharacter string

	// Pad msg field with spaces on the right for display.
	// The value for this parameter will be the size of padding.
	// Its default value is zero, which means no padding will be applied for msg.
	SpacePadding int

	// Color scheme to use.
	colorScheme *compiledColorScheme

	// Whether the logger's out is to a terminal.
	isTerminal bool

	sync.Once
}

func getCompiledColor(main string, fallback string) func(string) string {
	var style string
	if main != "" {
		style = main
	} else {
		style = fallback
	}
	return ansi.ColorFunc(style)
}

func compileColorScheme(s *ColorScheme) *compiledColorScheme {
	return &compiledColorScheme{
		InfoLevelColor:  getCompiledColor(s.InfoLevelStyle, defaultColorScheme.InfoLevelStyle),
		WarnLevelColor:  getCompiledColor(s.WarnLevelStyle, defaultColorScheme.WarnLevelStyle),
		ErrorLevelColor: getCompiledColor(s.ErrorLevelStyle, defaultColorScheme.ErrorLevelStyle),
		FatalLevelColor: getCompiledColor(s.FatalLevelStyle, defaultColorScheme.FatalLevelStyle),
		PanicLevelColor: getCompiledColor(s.PanicLevelStyle, defaultColorScheme.PanicLevelStyle),
		DebugLevelColor: getCompiledColor(s.DebugLevelStyle, defaultColorScheme.DebugLevelStyle),
		PrefixColor:     getCompiledColor(s.PrefixStyle, defaultColorScheme.PrefixStyle),
		TimestampColor:  getCompiledColor(s.TimestampStyle, defaultColorScheme.TimestampStyle),
	}
}

func (f *TextFormatter) init(entry *logrus.Entry) {
	if len(f.QuoteCharacter) == 0 {
		f.QuoteCharacter = "\""
	}
	if entry.Logger != nil {
		f.isTerminal = f.checkIfTerminal(entry.Logger.Out)
	}
}

func (f *TextFormatter) checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}

func (f *TextFormatter) SetColorScheme(colorScheme *ColorScheme) {
	f.colorScheme = compileColorScheme(colorScheme)
}

func (f *TextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	var keys []string = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	lastKeyIdx := len(keys) - 1

	if !f.DisableSorting {
		sort.Strings(keys)
	}
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	prefixFieldClashes(entry.Data)

	f.Do(func() { f.init(entry) })

	isFormatted := f.ForceFormatting || f.isTerminal

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}
	if isFormatted {
		isColored := (f.ForceColors || f.isTerminal) && !f.DisableColors
		var colorScheme *compiledColorScheme
		if isColored {
			if f.colorScheme == nil {
				colorScheme = defaultCompiledColorScheme
			} else {
				colorScheme = f.colorScheme
			}
		} else {
			colorScheme = noColorsColorScheme
		}
		f.printColored(b, entry, keys, timestampFormat, colorScheme)
	} else {
		if !f.DisableTimestamp {
			f.appendKeyValue(b, "time", entry.Time.Format(timestampFormat), true)
		}
		f.appendKeyValue(b, "level", entry.Level.String(), true)
		for i, key := range keys {
			f.appendKeyValue(b, key, entry.Data[key], lastKeyIdx != i)
		}
		if entry.Message != "" {
			f.appendKeyValue(b, "msg", entry.Message, lastKeyIdx >= 0)
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func (f *TextFormatter) printColored(b *bytes.Buffer, entry *logrus.Entry, keys []string, timestampFormat string, colorScheme *compiledColorScheme) {
	var levelColor func(string) string
	var levelText string
	switch entry.Level {
	case logrus.InfoLevel:
		levelColor = colorScheme.InfoLevelColor
	case logrus.WarnLevel:
		levelColor = colorScheme.WarnLevelColor
	case logrus.ErrorLevel:
		levelColor = colorScheme.ErrorLevelColor
	case logrus.FatalLevel:
		levelColor = colorScheme.FatalLevelColor
	case logrus.PanicLevel:
		levelColor = colorScheme.PanicLevelColor
	default:
		levelColor = colorScheme.DebugLevelColor
	}

	if entry.Level != logrus.WarnLevel {
		levelText = entry.Level.String()
	} else {
		levelText = "warn"
	}

	if !f.DisableUppercase {
		levelText = strings.ToUpper(levelText)
	}

	level := levelColor(fmt.Sprintf("%5s", levelText))
	prefix := ""
	message := entry.Message

	if prefixValue, ok := entry.Data["prefix"]; ok {
		prefix = colorScheme.PrefixColor(" " + prefixValue.(string) + ":")
	} else {
		prefixValue, trimmedMsg := extractPrefix(entry.Message)
		if len(prefixValue) > 0 {
			prefix = colorScheme.PrefixColor(" " + prefixValue + ":")
			message = trimmedMsg
		}
	}

	messageFormat := "%s"
	if f.SpacePadding != 0 {
		messageFormat = fmt.Sprintf("%%-%ds", f.SpacePadding)
	}

	if f.DisableTimestamp {
		fmt.Fprintf(b, "%s%s "+messageFormat, level, prefix, message)
	} else {
		var timestamp string
		if !f.FullTimestamp {
			timestamp = fmt.Sprintf("[%04d]", miniTS())
		} else {
			timestamp = fmt.Sprintf("[%s]", entry.Time.Format(timestampFormat))
		}
		fmt.Fprintf(b, "%s %s%s ", colorScheme.TimestampColor(timestamp), level, prefix)
	}
	for _, k := range keys {
		if k != "prefix" {
			v := entry.Data[k]
			fmt.Fprintf(b, " %s=%+v", k, levelColor(fmt.Sprintf("%v", v)))
		}
	}
	fmt.Fprintf(b, ":\t %s", levelColor(fmt.Sprintf(""+messageFormat, message)))
}

func (f *TextFormatter) needsQuoting(text string) bool {
	if f.QuoteEmptyFields && len(text) == 0 {
		return true
	}
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return true
		}
	}
	return false
}

func extractPrefix(msg string) (string, string) {
	prefix := ""
	regex := regexp.MustCompile(`^[(.*?)]`)
	if regex.MatchString(msg) {
		match := regex.FindString(msg)
		prefix, msg = match[1:len(match)-1], strings.TrimSpace(msg[len(match):])
	}
	return prefix, msg
}

func (f *TextFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}, appendSpace bool) {
	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)

	if appendSpace {
		b.WriteByte(' ')
	}
}

func (f *TextFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	switch value := value.(type) {
	case string:
		if !f.needsQuoting(value) {
			b.WriteString(value)
		} else {
			fmt.Fprintf(b, "%s%v%s", f.QuoteCharacter, value, f.QuoteCharacter)
		}
	case error:
		errmsg := value.Error()
		if !f.needsQuoting(errmsg) {
			b.WriteString(errmsg)
		} else {
			fmt.Fprintf(b, "%s%v%s", f.QuoteCharacter, errmsg, f.QuoteCharacter)
		}
	default:
		fmt.Fprint(b, value)
	}
}

// This is to not silently overwrite `time`, `msg` and `level` fields when
// dumping it. If this code wasn't there doing:
//
//  logrus.WithField("level", 1).Info("hello")
//
// would just silently drop the user provided level. Instead with this code
// it'll be logged as:
//
//  {"level": "info", "fields.level": 1, "msg": "hello", "time": "..."}
func prefixFieldClashes(data logrus.Fields) {
	if t, ok := data["time"]; ok {
		data["fields.time"] = t
	}

	if m, ok := data["msg"]; ok {
		data["fields.msg"] = m
	}

	if l, ok := data["level"]; ok {
		data["fields.level"] = l
	}
}
