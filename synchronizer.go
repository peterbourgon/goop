package main

const (
	Mod = "mod"
)

func ModEvent(i int) Event { return Event{Mod, float32(i), nil} }

// A synchronizer buffers upstream Events, and releases them
// downstream only when an appropriate Tick is received.
type Synchronizer struct {
	nodeName
	singleAncestry

	eventIn chan Event
	buffer  []Event
	mod     int
}

func NewSynchronizer(name string) *Synchronizer {
	s := &Synchronizer{
		nodeName: nodeName(name),

		eventIn: make(chan Event),
		buffer:  []Event{},
		mod:     1,
	}
	go s.loop()
	return s
}

func NewSynchronizerNode(name string) Node { return Node(NewSynchronizer(name)) }

// Kind satisfies the Typed interface for Synchronizer.
func (s *Synchronizer) Kind() string { return "synchronizer" }

// Events satisfies the Node interface for Synchronizer.
func (s *Synchronizer) Events() chan<- Event { return s.eventIn }

func (s *Synchronizer) loop() {
	for {
		select {
		case ev := <-s.eventIn:
			switch ev.Type {
			case Tick:
				if int(ev.Value)%s.mod != 0 {
					break
				}
				if s.ChildNode != nilNode {
					receiver := s.ChildNode.Events()
					for _, ev := range s.buffer {
						receiver <- ev
					}
				}
				s.buffer = []Event{}

			case Mod:
				i := int(ev.Value)
				if i <= 0 || i > 100 {
					D("%s: invalid Mod %.2f (%d)", s.Name(), ev.Value, i)
					break
				}
				s.mod = i

			case Kill:
				return

			case Connect, Disconnect, Connection, Disconnection:
				s.singleAncestry.processEvent(ev, s)

			default:
				s.buffer = append(s.buffer, ev)
			}
		}
	}
}
