// Package schedule is the homepage module's schedule adapter: it registers the
// module's periodic jobs on the shared scheduler handed in by the composition
// root. No business logic lives here.
package schedule

import (
	"time"

	appschedule "github.com/fatkulnurk/go-project-starter/internal/application/schedule"
)

// Register registers every periodic job of the homepage module. Today that is
// the demo tick job that logs the current time every minute; replace it with
// real recurring work as the module grows.
func Register(r appschedule.Registrar) {
	r.Register(appschedule.Job{
		Name:     "homepage.tick",
		Interval: time.Minute,
		Handler:  tick,
	})
}
