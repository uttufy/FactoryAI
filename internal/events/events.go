package events

import "time"

type EventType int

const (
	EvtStationStarted EventType = iota
	EvtStationInspecting
	EvtStationDone
	EvtStationFailed
	EvtMerging
	EvtDone
)

type Event struct {
	Type        EventType
	Timestamp   time.Time
	LineName    string
	StationName string
	Duration    time.Duration
	Output      string
	Error       error
	Retries     int
	Passed      bool
	Reasoning   string
}

func StationStarted(lineName, stationName string) Event {
	return Event{
		Type:        EvtStationStarted,
		Timestamp:   time.Now(),
		LineName:    lineName,
		StationName: stationName,
	}
}

func StationInspecting(lineName, stationName string) Event {
	return Event{
		Type:        EvtStationInspecting,
		Timestamp:   time.Now(),
		LineName:    lineName,
		StationName: stationName,
	}
}

func StationDone(lineName, stationName string, duration time.Duration, output string, retries int) Event {
	return Event{
		Type:        EvtStationDone,
		Timestamp:   time.Now(),
		LineName:    lineName,
		StationName: stationName,
		Duration:    duration,
		Output:      output,
		Retries:     retries,
		Passed:      true,
	}
}

func StationFailed(lineName, stationName string, duration time.Duration, err error, retries int) Event {
	return Event{
		Type:        EvtStationFailed,
		Timestamp:   time.Now(),
		LineName:    lineName,
		StationName: stationName,
		Duration:    duration,
		Error:       err,
		Retries:     retries,
		Passed:      false,
	}
}

func Merging() Event {
	return Event{
		Type:      EvtMerging,
		Timestamp: time.Now(),
	}
}

func Done(output string) Event {
	return Event{
		Type:      EvtDone,
		Timestamp: time.Now(),
		Output:    output,
	}
}
