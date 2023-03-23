package memory

import "context"

type SubscriberRepository struct {
	s []int
}

func NewSubscriberRepository() *SubscriberRepository {
	return &SubscriberRepository{s: make([]int, 0)}
}

func (sr *SubscriberRepository) StoreSubscriber(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sr.s = append(sr.s, id)
	return nil
}

func (sr *SubscriberRepository) RemoveSubscribe(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for i, v := range sr.s {
		if v == id {
			sr.s = append(sr.s[:i], sr.s[i+1:]...)
			break
		}
	}
	return nil
}

func (sr *SubscriberRepository) GetSubscribers(ctx context.Context) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return sr.s, nil
}
