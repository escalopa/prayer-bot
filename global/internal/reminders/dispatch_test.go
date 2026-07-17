package reminders

import (
	"context"
	"testing"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/store"
)

type fakeDispatchStore struct {
	items []store.OutboxItem
}

func (f *fakeDispatchStore) ClaimDue(context.Context, time.Time, int) (int, error) {
	return 0, nil
}

func (f *fakeDispatchStore) PendingOutbox(context.Context, int) ([]store.OutboxItem, error) {
	return f.items, nil
}

func (f *fakeDispatchStore) MarkOutboxEnqueued(_ context.Context, id int64) error {
	for index := range f.items {
		if f.items[index].ID == id {
			f.items = append(f.items[:index], f.items[index+1:]...)
			break
		}
	}
	return nil
}

type enqueuedTask struct {
	key, endpoint string
	runAt         time.Time
	payload       []byte
}

type fakeTaskEnqueuer struct {
	tasks []enqueuedTask
}

func (f *fakeTaskEnqueuer) Enqueue(_ context.Context, key, endpoint string, runAt time.Time, payload []byte) error {
	f.tasks = append(f.tasks, enqueuedTask{key: key, endpoint: endpoint, runAt: runAt, payload: payload})
	return nil
}

func (f *fakeTaskEnqueuer) Close() error { return nil }

func TestDispatcherPreservesCleanupEndpointAndSchedule(t *testing.T) {
	runAt := time.Date(2026, time.July, 19, 21, 0, 0, 0, time.UTC)
	storage := &fakeDispatchStore{items: []store.OutboxItem{{
		ID: 7, DeliveryKey: "delete:42:100:expiry", Endpoint: "/tasks/delete",
		RunAt: runAt, Payload: []byte(`{"message_id":100}`),
	}}}
	enqueuer := &fakeTaskEnqueuer{}

	count, err := NewDispatcher(storage, enqueuer, 10).Run(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(enqueuer.tasks) != 1 {
		t.Fatalf("count=%d tasks=%d", count, len(enqueuer.tasks))
	}
	task := enqueuer.tasks[0]
	if task.endpoint != "/tasks/delete" || !task.runAt.Equal(runAt) || task.key != "delete:42:100:expiry" {
		t.Fatalf("unexpected enqueued task: %+v", task)
	}
	if len(storage.items) != 0 {
		t.Fatal("outbox item was not acknowledged")
	}
}
