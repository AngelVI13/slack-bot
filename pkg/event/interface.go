package event

type EventType int

type Event interface {
	Type() EventType
	String() string
}

// this is set to a very high value so that no matter, how many
// other events are added - this should always be unique
const AnyEvent = 1_000_000
