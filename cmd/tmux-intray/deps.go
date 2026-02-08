package main

import (
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

var fileStorage, _ = storage.NewFileStorage()
var coreClient = core.NewCore(nil, fileStorage)
