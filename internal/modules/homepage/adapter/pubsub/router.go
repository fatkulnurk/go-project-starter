package pubsub

import (
	"github.com/fatkulnurk/go-project-starter/internal/application/pubsub"
)

// Register subscribes every topic handler of the homepage module. Today that
// is the demo PingTopic handler; replace it with real broadcast consumers as
// the module grows.
func Register(r pubsub.Registrar) {
	r.Subscribe(PingTopic, onPing)
}
