package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"windows/internal/attachments"
	"windows/internal/audiostream"
	"windows/internal/ipfs"
	"windows/internal/localapi"
	"windows/internal/services"
	"windows/internal/store"
	"windows/internal/updates"
)

// Library exposes the local authoring functions to the application window.
type Library struct {
	once                sync.Once
	store               *store.Store
	err                 error
	publicMu            sync.Mutex
	feedMu              sync.Mutex
	nodeMu              sync.RWMutex
	node                *ipfs.Client
	provideMu           sync.Mutex
	provideCtx          context.Context
	provideCancel       context.CancelFunc
	provideWorkers      sync.WaitGroup
	providePollInterval time.Duration
	provideWaitLimit    time.Duration
	audioMu             sync.Mutex
	audio               *audiostream.Server
	audioTemp           string
	audioSnapshot       string
	audioFile           string
	editorMediaMu       sync.Mutex
	editorMedia         *audiostream.Server
	updateMu            sync.Mutex
	checkedUpdate       *updates.Release
	updateIdentity      string
	downloadedUpdate    string
	updateGeneration    uint64
	updateCancel        context.CancelFunc
	updateClosing       bool
	updateWorkers       sync.WaitGroup
	xPoster             services.XPoster
	telegramAPI         services.TelegramAPI
	apiMu               sync.Mutex
	apiServer           *localapi.Server
	activePublishes     atomic.Int32
	defaultDataDir      string
}

func (l *Library) open() (*store.Store, error) {
	l.once.Do(func() {
		var path string
		path, l.err = mygo.App.Path(mygo.PathUserData)
		if l.err == nil {
			l.store, l.err = store.Open(path)
			if l.err == nil {
				attachments.CleanupDraftFiles(path, l.store)
				attachments.CleanupArticleFiles(path, l.store)
			}
		}
	})
	return l.store, l.err
}
