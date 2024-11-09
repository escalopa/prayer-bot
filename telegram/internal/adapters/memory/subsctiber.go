package memory

import (
	"context"
	"sync"
)

type SubscriberRepository struct {
	ids map[int]struct{}
	mu  sync.RWMutex
}

func NewSubscriberRepository() *SubscriberRepository {
	return &SubscriberRepository{ids: make(map[int]struct{})}
}

func (sr *SubscriberRepository) StoreSubscriber(_ context.Context, id int) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	sr.ids[id] = struct{}{}

	return nil
}

func (sr *SubscriberRepository) RemoveSubscribe(_ context.Context, id int) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	delete(sr.ids, id)

	return nil
}

func (sr *SubscriberRepository) GetSubscribers(_ context.Context) ([]int, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	ids := make([]int, 0, len(sr.ids))
	for id := range sr.ids {
		ids = append(ids, id)
	}

	return ids, nil
}
