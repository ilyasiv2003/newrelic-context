package nrmock

import "github.com/newrelic/go-agent/v3/newrelic"

type DatastoreSegment struct {
	*newrelic.DatastoreSegment
	Txn       *newrelic.Transaction
	StartTime newrelic.SegmentStartTime
	Finished  bool
}

func (m *DatastoreSegment) End() {
	m.Finished = true
}
