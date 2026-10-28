package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/grafana/loki/v3/pkg/logproto"
	"github.com/grafana/loki/v3/pkg/storage/config"
	"github.com/grafana/loki/v3/pkg/storage/stores/shipper/indexshipper/tsdb/index"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

type chunkID struct {
	fingerprint uint64
	from        int64
	through     int64
	checksum    uint32
}

func runChunks(r *index.Reader, tenantID string, schemaFrom string) error {
	start, err := time.Parse("2006-01-02", schemaFrom)
	if err != nil {
		return fmt.Errorf("invalid schema start date: %w", err)
	}

	schema := config.SchemaConfig{
		Configs: []config.PeriodConfig{
			{
				From:   config.DayTime{Time: model.TimeFromUnix(start.Unix())},
				Schema: "v13",
			},
		},
	}

	names, err := r.LabelNames()
	if err != nil {
		return fmt.Errorf("read label names: %w", err)
	}

	seen := make(map[chunkID]struct{})

	for _, name := range names {
		values, err := r.LabelValues(name)
		if err != nil {
			return fmt.Errorf("read values for %q: %w", name, err)
		}

		p, err := r.Postings(name, nil, values...)
		if err != nil {
			return fmt.Errorf("read postings for %q: %w", name, err)
		}

		for p.Next() {
			seriesRef := p.At()

			var lbls labels.Labels
			var metas []index.ChunkMeta

			fingerprint, err := r.Series(
				seriesRef,
				math.MinInt64,
				math.MaxInt64,
				&lbls,
				&metas,
			)
			if err != nil {
				return fmt.Errorf("read series %d: %w", seriesRef, err)
			}

			for _, m := range metas {
				id := chunkID{
					fingerprint: fingerprint,
					from:        m.MinTime,
					through:     m.MaxTime,
					checksum:    m.Checksum,
				}

				if _, exists := seen[id]; exists {
					continue
				}
				seen[id] = struct{}{}

				ref := logproto.ChunkRef{
					UserID:      tenantID,
					Fingerprint: fingerprint,
					From:        model.Time(m.MinTime),
					Through:     model.Time(m.MaxTime),
					Checksum:    m.Checksum,
				}

				fmt.Println(schema.ExternalKey(ref))
			}
		}

		if err := p.Err(); err != nil {
			return fmt.Errorf("iterate postings for %q: %w", name, err)
		}
	}

	return nil
}

func runLabels(r *index.Reader) error {
	names, err := r.LabelNames()
	if err != nil {
		return fmt.Errorf("read label names: %w", err)
	}

	for _, name := range names {
		values, err := r.LabelValues(name)
		if err != nil {
			return fmt.Errorf("read values for %q: %w", name, err)
		}

		for _, value := range values {
			fmt.Printf("%s=%s\n", name, value)
		}
	}

	return nil
}

func main() {
	mode := flag.String(
		"mode",
		"chunks",
		"Operation mode: chunks or labels",
	)
	tenantID := flag.String(
		"tenant",
		"fake",
		"Loki tenant ID, used in chunks mode",
	)
	schemaFrom := flag.String(
		"schema-from",
		"2024-01-01",
		"Schema start date (YYYY-MM-DD), used in chunks mode",
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: %s [options] <index-file>\n\nOptions:\n",
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	if *mode != "chunks" && *mode != "labels" {
		fmt.Fprintln(os.Stderr, "Error: -mode must be chunks or labels")
		os.Exit(2)
	}

	r, err := index.NewFileReader(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening index:", err)
		os.Exit(1)
	}
	defer r.Close()

	switch *mode {
	case "chunks":
		err = runChunks(r, *tenantID, *schemaFrom)
	case "labels":
		err = runLabels(r)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
