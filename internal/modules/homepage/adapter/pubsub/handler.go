// Package pubsub contains the homepage module's pub/sub handlers. It
// demonstrates the publish/subscribe pipeline: the schedule tick publishes
// demo events to PingTopic, and this adapter subscribes to them and logs their
// arrival (running in cmd/subscriber).
package pubsub

import (
	"context"
	"log/slog"
)

// PingTopic is the demo topic the homepage module uses to prove the pipeline:
// cmd/scheduler publishes it once a minute, cmd/subscriber logs it.
const PingTopic = "app.demo.ping"

// onPing logs an arriving demo event. Replace it with real broadcast work as
// the module grows.
func onPing(ctx context.Context, topic string, payload []byte) error {
	slog.Info("pubsub received", "topic", topic, "payload", string(payload))
	return nil
}
