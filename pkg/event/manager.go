package event

type Consumer interface {
	Consume(Event)
}

type ConsumerWithContext interface {
	Consumer
	Context() string
}

type EventManager struct {
	events      chan Event
	subscribers map[EventType][]Consumer
}

func NewEventManager() *EventManager {
	return &EventManager{
		events:      make(chan Event),
		subscribers: map[EventType][]Consumer{},
	}
}

func (em *EventManager) Subscribe(consumer Consumer, eventTypes ...EventType) {
	for _, eventType := range eventTypes {
		if _, ok := em.subscribers[eventType]; !ok {
			em.subscribers[eventType] = []Consumer{}
		}

		em.subscribers[eventType] = append(em.subscribers[eventType], consumer)
	}
}

// SubscribeWithContext The same as Subscribe(...) but indicates that context will be
// used to determine if event is forwarded to a given subscriber
func (em *EventManager) SubscribeWithContext(consumer ConsumerWithContext, eventTypes ...EventType) {
	em.Subscribe(consumer, eventTypes...)
}

func (em *EventManager) Publish(event Event) {
	em.events <- event
}

func (em *EventManager) ManageEvents() {
	for {
		event := <-em.events
		// Send events to subscribers that listen to a specific event
		eventType := event.Type()
		subs, ok := em.subscribers[eventType]
		if ok {
			// TODO: if subscriber implements Context() then check if the event contains
			// this context and only then forwards it to the subscriber
			for _, sub := range subs {
				go sub.Consume(event)
			}
		}

		// Send events to subscribers that listen to ALL events
		subs, ok = em.subscribers[AnyEvent]
		if ok {
			for _, sub := range subs {
				go sub.Consume(event)
			}
		}
	}
}
