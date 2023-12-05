package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/logrusorgru/aurora/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"io"
	"strings"
	"sync"
	"time"
)

type logger struct {
	writer   io.Writer
	level    Level
	queryMap sync.Map
	pretty   bool
	color    bool
}

func New(opt ...Options) *event.CommandMonitor {
	opts := optionsDefault(opt...)

	l := &logger{
		writer: opts.Writer,
		level:  opts.Level,
	}
	return &event.CommandMonitor{
		Started:   l.handleStartedEvent,
		Succeeded: l.handleSucceedEvent,
		Failed:    l.handleFailedEvent,
	}
}

func (l *logger) storeQuery(requestId int64, query bson.Raw) {
	var queryString string
	if l.pretty {
		queryByte, err := json.MarshalIndent(query, "", "  ")
		if err != nil {
			panic(err)
		}
		queryString = string(queryByte)
	} else {
		queryString = fmt.Sprint(query)
	}

	l.queryMap.Store(requestId, queryString)
}

func (l *logger) getQuery(requestId int64) string {
	val, ok := l.queryMap.Load(requestId)
	if ok {
		l.queryMap.Delete(requestId)
		return val.(string)
	}
	return ""
}

func (l *logger) timeNow() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func (l *logger) printSuccessQuery(requestId int64, method string, duration time.Duration) {
	if l.level == LevelError || l.level == LevelSilent {
		return
	}

	var text string
	if l.color {
		queryString := l.getQuery(requestId)
		timeStr := aurora.Green("[%s]").String()
		ms := aurora.Cyan("[%s] ").String()
		info := aurora.Yellow("[%s]\n").String()
		query := aurora.BrightGreen("%s").String()
		text = aurora.Sprintf(timeStr+ms+info+query, l.timeNow(), l.formatDuration(duration), strings.ToUpper(method), queryString)
	} else {
		queryString := l.getQuery(requestId)
		timeStr := "[%s]"
		ms := "[%s] "
		info := "[%s]\n"
		query := "%s"
		text = aurora.Sprintf(timeStr+ms+info+query, l.timeNow(), l.formatDuration(duration), strings.ToUpper(method), queryString)
	}

	_, err := fmt.Fprintln(l.writer, text)
	if err != nil {
		panic(err)
	}
}
func (l *logger) printFailedQuery(requestId int64, method, failure string, duration time.Duration) {
	if l.level == LevelSilent {
		return
	}

	var text string
	if l.color {
		queryString := l.getQuery(requestId)
		timeStr := aurora.Green("[%s]").String()
		ms := aurora.Cyan("[%s] ").String()
		info := aurora.Yellow("[%s] ").String()
		debug := aurora.BrightGreen("%s\n").String()
		query := aurora.Red("%s").String()
		text = aurora.Sprintf(timeStr+ms+info+debug+query, l.timeNow(), l.formatDuration(duration), strings.ToUpper(method), failure, queryString)
	} else {
		queryString := l.getQuery(requestId)
		timeStr := "[%s]"
		ms := "[%s] "
		info := "[%s] "
		debug := "%s\n"
		query := "%s"
		text = aurora.Sprintf(timeStr+ms+info+debug+query, l.timeNow(), l.formatDuration(duration), strings.ToUpper(method), failure, queryString)
	}

	_, err := fmt.Fprintln(l.writer, text)
	if err != nil {
		panic(err)
	}
}

func (l *logger) handleStartedEvent(_ context.Context, evt *event.CommandStartedEvent) {
	l.storeQuery(evt.RequestID, evt.Command)
	return
}

func (l *logger) handleFailedEvent(_ context.Context, evt *event.CommandFailedEvent) {
	l.printFailedQuery(evt.RequestID, evt.CommandName, evt.Failure, evt.Duration)
	return
}

func (l *logger) handleSucceedEvent(_ context.Context, evt *event.CommandSucceededEvent) {
	l.printSuccessQuery(evt.RequestID, evt.CommandName, evt.Duration)
	return
}

func (l *logger) formatDuration(duration time.Duration) string {
	if duration.Nanoseconds() < 1000000 {
		return fmt.Sprintf("%dns", duration.Nanoseconds())
	}
	if duration.Milliseconds() < 1000 {
		return fmt.Sprintf("%dms", duration.Milliseconds())
	}
	if duration.Seconds() < 60.0 {
		return fmt.Sprintf("%ds", int64(duration.Seconds()))
	}
	if duration.Minutes() < 60.0 {
		return fmt.Sprintf("%dm", int64(duration.Minutes()))
	}
	if duration.Hours() < 24.0 {
		return fmt.Sprintf("%dh", int64(duration.Hours()))
	}

	return fmt.Sprintf("%dd", int64(duration.Hours()/24))
}