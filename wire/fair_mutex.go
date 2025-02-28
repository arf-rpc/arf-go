package wire

type FairMutex struct {
	queue chan struct{}
}

func NewFairMutex() *FairMutex {
	return &FairMutex{
		queue: make(chan struct{}, 1),
	}
}

func (m *FairMutex) Lock() {
	m.queue <- struct{}{}
}

func (m *FairMutex) Unlock() {
	<-m.queue
}
