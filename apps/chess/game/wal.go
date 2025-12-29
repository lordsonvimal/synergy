package game

type WALEvent struct {
	Seq       uint64
	MoveUCI   string
	ServerNs  int64
	LagCompNs int64
	WRem      int64
	BRem      int64
}

type WAL struct {
	Events []WALEvent
}

func (w *WAL) Append(e WALEvent) {
	w.Events = append(w.Events, e)
}
