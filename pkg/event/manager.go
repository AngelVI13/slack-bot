package event

import (
	"log"
	"slices"
)

type Consumer interface {
	Consume(Event)
}

type ConsumerWithContext interface {
	Consumer
	Context() string
}

const NoContext = ""

type ConsumerNoContext struct {
	Consumer
}

func (c *ConsumerNoContext) Context() string {
	return NoContext
}

type EventManager struct {
	events      chan Event
	subscribers map[EventType][]ConsumerWithContext
}

func NewEventManager() *EventManager {
	return &EventManager{
		events:      make(chan Event),
		subscribers: map[EventType][]ConsumerWithContext{},
	}
}

// Subscribe Subscribe for events of the chosen type without considering any context
func (em *EventManager) Subscribe(consumer Consumer, eventTypes ...EventType) {
	em.subscribe(&ConsumerNoContext{consumer}, eventTypes...)
}

func (em *EventManager) subscribe(consumer ConsumerWithContext, eventTypes ...EventType) {
	if slices.Contains(eventTypes, AnyEvent) && len(eventTypes) > 1 {
		log.Fatalf(
			"Can't mix specific events with AnyEvent type: %s :: %v",
			consumer.Context(),
			eventTypes,
		)
	}
	for _, eventType := range eventTypes {
		if _, ok := em.subscribers[eventType]; !ok {
			em.subscribers[eventType] = []ConsumerWithContext{}
		}

		em.subscribers[eventType] = append(em.subscribers[eventType], consumer)
	}
}

// SubscribeWithContext Subscribe for events of the chosen type provided they also
// contain the expected context (Context()).
func (em *EventManager) SubscribeWithContext(
	consumer ConsumerWithContext,
	eventTypes ...EventType,
) {
	em.subscribe(consumer, eventTypes...)
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
				// NOTE: if subscribed with context then check if the event contains
				// this context and only then forward it to the subscriber
				// if subscribed without context -> forward to subscriber
				if MatchesContext(event, sub) {
					go sub.Consume(event)
				}
			}
		}

		// Send events to subscribers that listen to ALL events
		subs, ok = em.subscribers[AnyEvent]
		if ok {
			for _, sub := range subs {
				if MatchesContext(event, sub) {
					go sub.Consume(event)
				}
			}
		}
	}
}

func MatchesContext(event Event, sub ConsumerWithContext) bool {
	return (sub.Context() != "" && event.HasContext(sub.Context())) || sub.Context() == ""
}
