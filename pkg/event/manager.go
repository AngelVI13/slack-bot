package event

type Consumer interface {
	Consume(Event)
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
