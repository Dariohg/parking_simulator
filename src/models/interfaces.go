package models

type Position struct {
	ID            int
	Spot          int
	Status        string // Examples: "waiting in queue", "parked", "left"
	QueuePosition int    // Optional: Position in queue
}

// Observer defines the observer behavior
type Observer interface {
	Update(pos Position)
}

// Subject defines the subject behavior
type Subject interface {
	Register(observer Observer)
	NotifyAll(pos Position)
}
