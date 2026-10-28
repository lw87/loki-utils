package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/grafana/loki/v3/pkg/storage/chunk"

	// Register Loki's chunk encoding implementations.
	_ "github.com/grafana/loki/v3/pkg/chunkenc"
)

func run(filePath, objectKey string) error {
	// Expected schema v13 key:
	// <tenant>/<fingerprint>/<from>:<through>:<checksum>
	parts := strings.SplitN(objectKey, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid S3 object key: %q", objectKey)
	}

	tenantID := parts[0]

	// Parse the expected chunk metadata and checksum from the key.
	ch, err := chunk.ParseExternalKey(tenantID, objectKey)
	if err != nil {
		return fmt.Errorf("parse S3 object key: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read chunk file: %w", err)
	}

	// Loki validates the stored bytes against the parsed reference.
	if err := ch.Decode(chunk.NewDecodeContext(), data); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	fmt.Printf("OK;%s;%08x\n", objectKey, ch.Checksum)
	return nil
}

func main() {
	filePath := flag.String("file", "", "Local chunk file path")
	objectKey := flag.String("key", "", "Full S3 object key")
	flag.Parse()

	if *filePath == "" || *objectKey == "" {
		fmt.Fprintf(os.Stderr,
			"Usage: %s --file <path> --key <s3-object-key>\n",
			os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}

	if err := run(*filePath, *objectKey); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
