package main

import (
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

var storageStore = storage.NewDefaultStore()
var coreClient = core.NewCore(nil, storageStore)
