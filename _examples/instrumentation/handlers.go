package instrumentation

import (
	"log"
	"sort"
	"sync"

	"github.com/martinohmann/promise/instrumented"
)

// UUIDCollector is an example type for demonstrating custom instrumentation
// handlers.
type UUIDCollector struct {
	sync.Mutex
	uuidMap map[string]struct{}
}

func NewUUIDCollector() *UUIDCollector {
	return &UUIDCollector{
		uuidMap: make(map[string]struct{}),
	}
}

// CollectUUIDs is an instrumented.InstrumentationHandlerFunc.
func (c *UUIDCollector) CollectUUIDs(invocation *instrumented.Invocation) {
	c.Lock()
	defer c.Unlock()
	c.uuidMap[invocation.UUID] = struct{}{}
}

func (c *UUIDCollector) GetUUIDs() []string {
	uuids := make([]string, 0, len(c.uuidMap))
	for uuid := range c.uuidMap {
		uuids = append(uuids, uuid)
	}

	sort.Strings(uuids)
	return uuids
}

// LoggingHandler is an instrumented.InstrumentationHandlerFunc.
func LoggingHandler(invocation *instrumented.Invocation) {
	log.Printf(`
  Duration: %s
  UUID: %s
  Promise: %p
  CallerInfo:
    File: %s
    Line: %d
    Func: %s
  SubjectInfo:
    Subject: %s
    Arguments: %#v
    ReturnValues: %#v`,
		invocation.EndTime.Sub(invocation.StartTime),
		invocation.UUID,
		invocation.Promise,
		invocation.CallerInfo.File,
		invocation.CallerInfo.Line,
		invocation.CallerInfo.Func,
		invocation.SubjectInfo.Subject,
		invocation.SubjectInfo.Arguments,
		invocation.SubjectInfo.ReturnValues,
	)
}
