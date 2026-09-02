package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"charm.land/lipgloss/v2"
)

type globalLogConfig struct {
	logChan    chan LogMsg
	doneChan   chan struct{}
	isReady    bool
	logLevel   LogLevel
	logger     *log.Logger
	fileHandle *os.File
	fileName   string
	filePath   string
	mux        sync.Mutex
}

var logConfig = globalLogConfig{
	logLevel: None,
	fileName: "koshime.log",
	filePath: "",
}

// var (
// 	logChan         chan LogMsg
// 	doneChan        chan struct{}
// 	isReady         = false
// 	logLevel        = None
// 	stdLogger       *log.Logger
// 	fileHandle      *os.File
// 	defaultFileName = "log.txt"
// 	filePath        = ""
// 	mux             sync.Mutex
// )

const timeFormat = "03:04:05.000000 PM Z07:00"

const (
	Insane = LogLevel(iota)
	Hot
	Debug
	Info
	Attention
	Error
	None
)

var levelTags = []string{
	"∞∞∞", "HOT", "DBG", "NFO", "ATN", "ERR", "None",
}

type LogLevel int

func (ll LogLevel) String() string {
	return levelTags[ll]
}

func (ll LogLevel) IsValid() bool {
	if ll < Insane || ll > None {
		return false
	}
	return true
}

type LogMsg struct {
	msg   func() string
	level LogLevel
	file  string
	line  int
	vars  []any
}

func Log(ll LogLevel, msg string, vars ...any) {
	if logConfig.logLevel == None || ll < logConfig.logLevel {
		return
	}

	tryInitLog()

	_, file, line, _ := runtime.Caller(1)
	logConfig.logChan <- LogMsg{func() string { return msg }, ll, file, line, vars}
}

// LogFunc executes the msgFn and passes its result to the
// default log func, if the specified log level is active.
//
// This is useful if a log requires some heavier processing
// but you don't want it to affect the runtime of your
// application. The processing will happen inside the
// logger thread instead.
func LogFunc(ll LogLevel, msgFn func() string, vars ...any) {
	if logConfig.logLevel == None || ll < logConfig.logLevel {
		return
	}

	tryInitLog()

	if ll < logConfig.logLevel {
		return
	}

	_, file, line, _ := runtime.Caller(1)
	logConfig.logChan <- LogMsg{msgFn, ll, file, line, vars}
}

func LogFatal(errMsg string, description string, vars ...any) {
	defer func() {
		err := CloseLog()
		if err != nil {
			// Should effectively never happen
			panic(err)
		}
		os.Exit(1)
	}()

	maxDisplayWidth := 60

	errMsgStyle := lipgloss.NewStyle().
		Width(maxDisplayWidth).
		PaddingTop(1).
		PaddingLeft(1).
		Foreground(lipgloss.Red)

	description = strings.TrimSpace(description)
	if len(description) > 0 {
		paragraphs := strings.Split(description, "\n\n")
		for i, p := range paragraphs {
			paragraphs[i] = lipgloss.NewStyle().
				PaddingTop(1).
				Width(maxDisplayWidth).
				Render(strings.ReplaceAll(p, "\n", " "))
		}
		description = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingBottom(1).
			Foreground(lipgloss.Yellow).
			Render(lipgloss.JoinVertical(lipgloss.Left, paragraphs...))
	}

	if len(vars) > 0 {
		errMsg = errMsgStyle.Render(fmt.Sprintf(errMsg, vars...))
	} else {
		errMsg = errMsgStyle.Render(errMsg)
	}

	fmt.Print(
		lipgloss.JoinVertical(
			lipgloss.Left,
			errMsg,
			description,
			getStack(),
		) + "\n",
	)
}

func Reset() error {
	logConfig.mux.Lock()
	ll := logConfig.logLevel
	// Do not allow any calls to log during reset
	logConfig.logLevel = None

	// Init should be called once reset is done
	logConfig.isReady = false
	logConfig.mux.Unlock()

	err := CloseLog()
	if err != nil {
		return err
	}
	logConfig.logLevel = ll
	return nil
}

func SetLogLevel(ll LogLevel) error {
	if !ll.IsValid() {
		return fmt.Errorf("invalid log level: %d", ll)
	}
	logConfig.logLevel = ll
	return nil
}

func GetLogLevel() LogLevel {
	return logConfig.logLevel
}

func GetTimeFormat() string {
	return timeFormat
}

func SetFilePath(path string) {
	logConfig.filePath = path
}

func CloseLog() error {
	if logConfig.logLevel == None {
		panic("log was never initialized with a log level")
	}
	close(logConfig.logChan)
	<-logConfig.doneChan
	return logConfig.fileHandle.Close()
}

func tryInitLog() {
	if logConfig.isReady {
		return
	}

	if logConfig.filePath == "" {
		wd, err := os.Getwd()
		if err != nil {
			panic(err) // This should never happen
		}
		logConfig.filePath = filepath.Join(wd, logConfig.fileName)
	}

	logConfig.logChan = make(chan LogMsg, 50)
	logConfig.doneChan = make(chan struct{})

	var err error
	logConfig.fileHandle, err = os.OpenFile(
		logConfig.filePath,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0o600,
	)
	if err != nil {
		panic(err)
	}

	logConfig.logger = log.New(logConfig.fileHandle, "", 0)
	go logMessages()
	logConfig.isReady = true
}

func logMessages() {
	for log := range logConfig.logChan {
		msg := fmt.Sprintf(
			"%s [%s] [%s:%d]: %s\n",
			time.Now().Format(timeFormat),
			log.level,
			filepath.Base(log.file), log.line,
			log.msg(),
		)
		if len(log.vars) == 0 {
			logConfig.logger.Print(msg)
		} else {
			logConfig.logger.Printf(msg, log.vars...)
		}
	}
	close(logConfig.doneChan)
}

func getStack() string {
	pc := make([]uintptr, 4)
	n := runtime.Callers(3, pc)
	if n == 0 {
		return ""
	}

	pc = pc[:n]
	frames := runtime.CallersFrames(pc)

	var funcBuilder strings.Builder
	var fileBuilder strings.Builder
	for {
		frame, more := frames.Next()
		file := filepath.Base(frame.File)
		function := filepath.Base(frame.Function)
		fmt.Fprintf(&funcBuilder, "\t%s(): \n", function)
		fmt.Fprintf(&fileBuilder, "%s:%d\n", file, frame.Line)
		if !more {
			break
		}
	}
	funcStyle := lipgloss.NewStyle().
		Foreground(lipgloss.BrightBlack).
		Align(lipgloss.Right).
		Render(funcBuilder.String())
	display := lipgloss.JoinHorizontal(lipgloss.Top, funcStyle, fileBuilder.String())
	return display
}
