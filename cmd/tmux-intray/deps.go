package main

import (
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

var fileStorage, _ = storage.NewFromConfig()
var coreClient = core.NewCore(nil, fileStorage)
