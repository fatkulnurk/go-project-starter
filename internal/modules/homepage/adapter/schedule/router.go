// Package schedule is the homepage module's schedule adapter: it registers the
// module's periodic jobs on the shared scheduler handed in by the composition
// root. No business logic lives here.
package schedule

import (
	"context"

	apppubsub "github.com/fatkulnurk/go-project-starter/internal/application/pubsub"
	appschedule "github.com/fatkulnurk/go-project-starter/internal/application/schedule"
)

// Register registers every periodic job of the homepage module. pub broadcasts
// the demo pub/sub events; it may be nil to only log. Today that is the demo
// tick job that logs the current time every minute; replace it with real
// recurring work as the module grows.
func Register(r appschedule.Registrar, pub apppubsub.Publisher) {
	r.Register(appschedule.Job{
		Name:    "homepage.tick",
		Cron:    "* * * * *",
		Handler: func(ctx context.Context) error { return tick(ctx, pub) },
	})
}
