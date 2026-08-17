package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	apppubsub "github.com/fatkulnurk/go-project-starter/internal/application/pubsub"
	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage/adapter/pubsub"
)

// tick is the default scheduled job: it logs the current time so a running
// scheduler visibly proves the pipeline works, then broadcasts a demo pub/sub
// event on PingTopic for cmd/subscriber to receive. A nil publisher skips the
// publish and only logs.
func tick(ctx context.Context, pub apppubsub.Publisher) error {
	now := time.Now().Format(time.RFC3339)
	slog.Info("schedule tick", "time", now)
	if pub == nil {
		return nil
	}
	return pub.Publish(ctx, apppubsub.Message{
		Topic:   pubsub.PingTopic,
		Payload: []byte(fmt.Sprintf(`{"at":%q}`, now)),
	})
}
