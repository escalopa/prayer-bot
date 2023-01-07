package memory

type SubscriberRepository struct {
	s []int
}

func NewSubscriberRepository() *SubscriberRepository {
	return &SubscriberRepository{s: make([]int, 0)}
}

func (sr *SubscriberRepository) StoreSubscriber(id int) error {
	sr.s = append(sr.s, id)
	return nil
}

func (sr *SubscriberRepository) RemoveSubscribe(id int) error {
	for i, v := range sr.s {
		if v == id {
			sr.s = append(sr.s[:i], sr.s[i+1:]...)
			break
		}
	}
	return nil
}

func (sr *SubscriberRepository) GetSubscribers() ([]int, error) {
	return sr.s, nil
}
