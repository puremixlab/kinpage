package nativeui

import (
	"slices"

	"windows/internal/store"
)

type sitePublishRequest struct {
	siteID, siteName string
}

type sitePublishTask struct {
	sitePublishRequest
	done chan publishModalState
}

func (a *App) publishInProgress(siteID string) bool {
	return (a.publication != nil && a.publication.siteID == siteID) ||
		slices.ContainsFunc(a.publicationQueue, func(p sitePublishRequest) bool { return p.siteID == siteID })
}

// HasPendingPublications is used by the window's exit action on the UI thread.
func (a *App) HasPendingPublications() bool {
	return a.publication != nil || len(a.publicationQueue) > 0
}

func (a *App) triggerPublishSite(site store.Site) {
	if a.backend == nil || a.publishInProgress(site.ID) || a.taskContext().Err() != nil {
		return
	}
	name := a.fallbackTitle(site.Name)
	a.publicationQueue = append(a.publicationQueue, sitePublishRequest{siteID: site.ID, siteName: name})
	if a.publication == nil {
		a.startNextPublication()
	} else {
		a.setStatus(a.tf("status.sitePublishQueued", name))
	}
}

func (a *App) startNextPublication() {
	if len(a.publicationQueue) == 0 || a.taskContext().Err() != nil {
		return
	}
	task := &sitePublishTask{sitePublishRequest: a.publicationQueue[0], done: make(chan publishModalState, 1)}
	a.publicationQueue = a.publicationQueue[1:]
	a.publication = task
	a.setStatus(a.tf("status.sitePublishing", task.siteName))
	backend, ctx, invalidate := a.backend, a.taskContext(), a.invalidateWindow
	go func() {
		result, err := backend.PublishSiteContext(ctx, task.siteID)
		outcome := publishModalState{siteName: task.siteName, result: result}
		if err != nil {
			outcome.err = err.Error()
		}
		task.done <- outcome
		if invalidate != nil && ctx.Err() == nil {
			invalidate()
		}
	}()
}

func (a *App) finishSitePublication() {
	task := a.publication
	if task == nil {
		return
	}
	select {
	case outcome := <-task.done:
		a.publication = nil
		if sites, err := a.backend.Sites(); err == nil {
			a.sites = sites
			if site := a.findSiteByID(task.siteID); site != nil && site.ContentRevision > outcome.result.Revision {
				outcome.result.Pending = true
			}
		}
		if dashboard, err := a.backend.Dashboard(); err == nil {
			a.dashboard = dashboard
		}
		if a.publicationResults == nil {
			a.publicationResults = make(map[string]publishModalState)
		}
		a.publicationResults[task.siteID] = outcome
		if a.selectedSiteID == task.siteID && a.mode == viewSite && !a.editing && !a.anyModalOpen() {
			outcome.open = true
			a.publishModal = outcome
		}
		switch {
		case outcome.err != "":
			a.setError(a.tf("publishModal.failed", outcome.err))
		case outcome.result.HasWarnings():
			a.setStatus(a.tf("status.sitePublishWarnings", task.siteName))
		case outcome.result.Pending:
			a.setStatus(a.tf("status.sitePublishedPending", task.siteName))
		default:
			a.setStatus(a.tf("status.sitePublished", task.siteName))
		}
		a.startNextPublication()
	default:
	}
}
