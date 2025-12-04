package compute

type Nanoseconds uint64

// CPUStats represents the statistics of a CPU
type CPUStats struct {
	// Name of the CPU
	Name string
	// User time is the time spent executing user code
	User   Nanoseconds
	// Nice time is the time spent servicing nice processes
	Nice   Nanoseconds
	// System time is the time spent executing system code
	System Nanoseconds
	// Idle time is the time spent idle
	Idle   Nanoseconds
	// IOWait time is the time spent waiting for I/O operations
	IOWait Nanoseconds
	// IRQ time is the time spent servicing hardware interrupts
	IRQ    Nanoseconds
	// SoftIRQ time is the time 
	// spent servicing software interrupts
	SoftIRQ Nanoseconds
	// Steal time is the time spent stealing CPU time
	// The hypervisor may steal CPU time 
	// from the guest OS to run other tasks
	Steal  Nanoseconds
}

// Total returns the total time spent executing code on the CPU.
func (c *CPUStats) Total() Nanoseconds {
	return c.User + c.Nice + c.System + c.Idle +
		c.IOWait + c.IRQ + c.SoftIRQ + c.Steal
}

// Active returns the total time spent executing code on the CPU.
// It excludes the idle time.
func (c *CPUStats) Active() Nanoseconds {
	return c.User + c.Nice + c.System +
		c.IOWait + c.IRQ + c.SoftIRQ + c.Steal
}

// Usage returns the CPU usage percentage based on active time vs total time.
// Note: This calculates usage from a single reading (cumulative since boot).
func (c *CPUStats) Usage() float64 {
	return float64(c.Active()) / float64(c.Total()) * 100
}
