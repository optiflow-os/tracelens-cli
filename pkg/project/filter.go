package project

import (
	"context"
	"fmt"
	"os"

	"github.com/optiflow-os/tracelens-cli/pkg/heartbeat"
	"github.com/optiflow-os/tracelens-cli/pkg/log"
)

// FilterConfig contains project filtering configurations.
type FilterConfig struct {
	// ExcludeUnknownProject determines if heartbeat should be skipped when the project cannot be detected.
	ExcludeUnknownProject bool
}

// WithFiltering initializes and returns a heartbeat handle option, which
// can be used in a heartbeat processing pipeline to filter heartbeats following
// the provided configurations.
func WithFiltering(config FilterConfig) heartbeat.HandleOption {
	return func(next heartbeat.Handle) heartbeat.Handle {
		return func(ctx context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
			logger := log.Extract(ctx)
			logger.Debugln("execute project filtering")

			var filtered []heartbeat.Heartbeat

			for _, h := range hh {
				err := Filter(h, config)
				if err != nil {
					logger.Debugln(err.Error())

					if h.LocalFileNeedsCleanup {
						err = os.Remove(h.LocalFile)
						if err != nil {
							logger.Warnf("unable to delete tmp file: %s", err)
						}
					}

					continue
				}

				filtered = append(filtered, h)
			}

			return next(ctx, filtered)
		}
	}
}

// Filter determines, following the passed in configurations, if a heartbeat
// should be skipped.
func Filter(h heartbeat.Heartbeat, config FilterConfig) error {
	// exclude unknown project
	if config.ExcludeUnknownProject && (h.Project == nil || *h.Project == "") {
		return fmt.Errorf("skipping because of unknown project")
	}

	return nil
}
