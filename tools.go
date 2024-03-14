//go:build tools

package tools

// This file is used to tell `go mod` about development tools we require, but
// not necessarily imported into the main source code.
// Read more here: https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

import (
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
