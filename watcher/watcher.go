package watcher

import (
	"log"
	"sync"
	"time"

	"github.com/congregalis/stock-ping/config"
	"github.com/fsnotify/fsnotify"
)

// ConfigWatcher watches a config file for changes and calls a callback on reload
type ConfigWatcher struct {
	path     string
	watcher  *fsnotify.Watcher
	callback func(*config.Config)
	mu       sync.Mutex
	done     chan struct{}
}

// NewConfigWatcher creates a new config file watcher
func NewConfigWatcher(path string, callback func(*config.Config)) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	cw := &ConfigWatcher{
		path:     path,
		watcher:  watcher,
		callback: callback,
		done:     make(chan struct{}),
	}

	if err := watcher.Add(path); err != nil {
		watcher.Close()
		return nil, err
	}

	return cw, nil
}

// Start begins watching for file changes
func (cw *ConfigWatcher) Start() {
	go cw.watchLoop()
}

// Stop stops watching and cleans up resources
func (cw *ConfigWatcher) Stop() {
	close(cw.done)
	cw.watcher.Close()
}

func (cw *ConfigWatcher) watchLoop() {
	// Debounce to avoid multiple reloads on rapid file changes
	var debounceTimer *time.Timer
	debounceDuration := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			// Only react to write/create events
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				cw.mu.Lock()
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDuration, func() {
					cw.reload()
				})
				cw.mu.Unlock()
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Config watcher error: %v", err)

		case <-cw.done:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return
		}
	}
}

func (cw *ConfigWatcher) reload() {
	cfg, err := config.LoadFrom(cw.path)
	if err != nil {
		log.Printf("Failed to reload config: %v", err)
		return
	}
	cw.callback(cfg)
}
