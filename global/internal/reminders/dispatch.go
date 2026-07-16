package reminders

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	cloudtaskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/escalopa/prayer-bot/global/internal/store"
)

type DispatchStore interface {
	ClaimDue(context.Context, time.Time, int) (int, error)
	PendingOutbox(context.Context, int) ([]store.OutboxItem, error)
	MarkOutboxEnqueued(context.Context, int64) error
}

type TaskEnqueuer interface {
	Enqueue(context.Context, string, []byte) error
	Close() error
}

type Dispatcher struct {
	store    DispatchStore
	enqueuer TaskEnqueuer
	batch    int
}

func NewDispatcher(store DispatchStore, enqueuer TaskEnqueuer, batch int) *Dispatcher {
	return &Dispatcher{store: store, enqueuer: enqueuer, batch: batch}
}

func (d *Dispatcher) Run(ctx context.Context, now time.Time) (int, error) {
	if _, err := d.store.ClaimDue(ctx, now, d.batch); err != nil {
		return 0, fmt.Errorf("claim due reminders: %w", err)
	}
	items, err := d.store.PendingOutbox(ctx, d.batch)
	if err != nil {
		return 0, fmt.Errorf("load outbox: %w", err)
	}
	for index, item := range items {
		if err := d.enqueuer.Enqueue(ctx, item.DeliveryKey, item.Payload); err != nil {
			return index, fmt.Errorf("enqueue delivery: %w", err)
		}
		if err := d.store.MarkOutboxEnqueued(ctx, item.ID); err != nil {
			return index, fmt.Errorf("mark outbox: %w", err)
		}
	}
	return len(items), nil
}

type CloudTasksEnqueuer struct {
	client              *cloudtasks.Client
	queuePath           string
	senderURL           string
	serviceAccountEmail string
}

func NewCloudTasksEnqueuer(ctx context.Context, projectID, region, queue, senderURL, serviceAccountEmail string) (*CloudTasksEnqueuer, error) {
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &CloudTasksEnqueuer{
		client:              client,
		queuePath:           fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, region, queue),
		senderURL:           senderURL,
		serviceAccountEmail: serviceAccountEmail,
	}, nil
}

func (e *CloudTasksEnqueuer) Enqueue(ctx context.Context, deliveryKey string, payload []byte) error {
	digest := sha256.Sum256([]byte(deliveryKey))
	taskName := e.queuePath + "/tasks/" + hex.EncodeToString(digest[:])
	_, err := e.client.CreateTask(ctx, &cloudtaskspb.CreateTaskRequest{
		Parent: e.queuePath,
		Task: &cloudtaskspb.Task{
			Name: taskName,
			MessageType: &cloudtaskspb.Task_HttpRequest{HttpRequest: &cloudtaskspb.HttpRequest{
				HttpMethod: cloudtaskspb.HttpMethod_POST,
				Url:        e.senderURL + "/tasks/send",
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       payload,
				AuthorizationHeader: &cloudtaskspb.HttpRequest_OidcToken{OidcToken: &cloudtaskspb.OidcToken{
					ServiceAccountEmail: e.serviceAccountEmail,
					Audience:            e.senderURL,
				}},
			}},
		},
	})
	if status.Code(err) == codes.AlreadyExists {
		return nil
	}
	return err
}

func (e *CloudTasksEnqueuer) Close() error { return e.client.Close() }
