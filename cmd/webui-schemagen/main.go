// Command webui-schemagen generates the webui's service constants and
// descriptors from the woodhouse-api service schema
// (github.com/jimjibone/woodhouse-api/schema).
//
//	webui-schemagen -out webui/src/lib/schema/zz_services.ts
//
// Run it with "task webui:schema".
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jimjibone/woodhouse-api/schema"
)

func main() {
	out := flag.String("out", "webui/src/lib/schema/zz_services.ts", "output file")
	flag.Parse()

	src, err := generate(schema.MustLoad())
	if err != nil {
		fail("generate: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fail("mkdir: %v", err)
	}
	if err := os.WriteFile(*out, src, 0o644); err != nil {
		fail("write: %v", err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "webui-schemagen: "+format+"\n", args...)
	os.Exit(1)
}
