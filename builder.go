package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Tonkpils/snag/vow"
	"github.com/shiena/ansicolor"
	fsn "gopkg.in/fsnotify.v1"
)

var mtimes = map[string]time.Time{}
var clearBuffer = func() {
	fmt.Print("\033c")
}

type Bob struct {
	w        *fsn.Watcher
	mtx      sync.RWMutex
	curVow   *vow.Vow
	done     chan struct{}
	watching map[string]struct{}
	watchDir string

	cmds         [][]string
	ignoredItems []string

	verbose bool
}

func NewBuilder(c config) (*Bob, error) {
	w, err := fsn.NewWatcher()
	if err != nil {
		return nil, err
	}

	cmds := make([][]string, len(c.Script))
	for i, s := range c.Script {
		cmds[i] = strings.Split(s, " ")
	}

	return &Bob{
		w:            w,
		done:         make(chan struct{}),
		watching:     map[string]struct{}{},
		cmds:         cmds,
		ignoredItems: c.IgnoredItems,
		verbose:      c.Verbose,
	}, nil
}

func (b *Bob) Close() error {
	close(b.done)
	return b.w.Close()
}

func (b *Bob) Watch(path string) error {
	b.watchDir = path
	b.watch(path)
	b.execute()

	for {
		select {
		case ev := <-b.w.Events:
			var queueBuild bool
			switch {
			case isCreate(ev.Op):
				queueBuild = b.watch(ev.Name)
			case isDelete(ev.Op):
				if _, ok := b.watching[ev.Name]; ok {
					b.w.Remove(ev.Name)
					delete(b.watching, ev.Name)
				}
				queueBuild = true
			case isModify(ev.Op):
				queueBuild = true
			}
			if queueBuild {
				b.maybeQueue(ev.Name)
			}
		case err := <-b.w.Errors:
			log.Println("error:", err)
		case <-b.done:
			return nil
		}
	}
}

func (b *Bob) maybeQueue(path string) {
	if b.isExcluded(path) {
		return
	}

	stat, err := os.Stat(path)
	if err != nil {
		// we couldn't find the file
		// most likely a deletion
		delete(mtimes, path)
		b.execute()
		return
	}

	mtime := stat.ModTime()
	lasttime := mtimes[path]
	if !mtime.Equal(lasttime) {
		// the file has been modified and the
		// file system event wasn't bogus
		mtimes[path] = mtime
		b.execute()
	}
}

func (b *Bob) stopCurVow() {
	if b.curVow != nil {
		b.mtx.Lock()
		b.curVow.Stop()
		b.mtx.Unlock()
	}
}

func (b *Bob) execute() {
	b.stopCurVow()

	clearBuffer()
	b.mtx.Lock()

	// setup the first command
	firstCmd := b.cmds[0]
	b.curVow = vow.To(firstCmd[0], firstCmd[1:]...)

	// setup the remaining commands
	for i := 1; i < len(b.cmds); i++ {
		cmd := b.cmds[i]
		b.curVow = b.curVow.Then(cmd[0], cmd[1:]...)
	}
	b.curVow.Verbose = b.verbose
	go b.curVow.Exec(ansicolor.NewAnsiColorWriter(os.Stdout))

	b.mtx.Unlock()
}

func (b *Bob) watch(path string) bool {
	var shouldBuild bool
	if _, ok := b.watching[path]; ok {
		return false
	}
	filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if fi == nil {
			return filepath.SkipDir
		}

		if !fi.IsDir() {
			shouldBuild = true
			return nil
		}

		if b.isExcluded(p) {
			return filepath.SkipDir
		}

		if err := b.w.Add(p); err != nil {
			return err
		}
		b.watching[p] = struct{}{}

		return nil
	})
	return shouldBuild
}

func (b *Bob) isExcluded(path string) bool {
	// get the relative path
	path = strings.TrimPrefix(path, b.watchDir+string(filepath.Separator))

	for _, p := range b.ignoredItems {
		if p == path {
			return true
		}
	}
	return false
}

func isCreate(op fsn.Op) bool {
	return op&fsn.Create == fsn.Create
}

func isDelete(op fsn.Op) bool {
	return op&fsn.Remove == fsn.Remove
}

func isModify(op fsn.Op) bool {
	return op&fsn.Write == fsn.Write ||
		op&fsn.Rename == fsn.Rename
}
