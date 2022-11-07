package webapp

import "embed"

//go:embed public/*
var Content embed.FS
