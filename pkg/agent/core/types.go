package core

// TelemetryEvent is the standard format for logs, metrics
type TelemetryEvent struct {
    // ProbeName is the name of the probe that generated the event.
    ProbeName string 
    // Timestamp is the timestamp of the event.(Unix timestamp)
    Timestamp int64
    // Data is the data of the event.
    // TODO: Change data type for more flexible data
    Data string
}