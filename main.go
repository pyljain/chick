package main

import (
	"chick/pkg/csv"
	"chick/pkg/duckdb"
	"chick/pkg/parquet"
	"chick/pkg/rendering"
	"chick/pkg/schema"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"go.yaml.in/yaml/v3"
)

func main() {
	var loc string
	var dest string
	var schemaLoc string
	var query string
	var outputFormat string
	ctx := context.Background()
	// Take folder as input
	flag.StringVar(&loc, "location", "./samples", "Location of the CSV files to load")
	flag.StringVar(&dest, "destination", "./output", "Location where the generated parquet files will be stored")
	flag.StringVar(&schemaLoc, "schema", "", "Location where the schema YAML will available")
	flag.StringVar(&query, "query", "", "Query to run")
	flag.StringVar(&outputFormat, "format", "table", "Output format you'd like. Options are table (default) or JSON")
	flag.Parse()

	// Load CSVs and buffer into parquet files
	workerChan, err := csv.Collect(loc)
	if err != nil {
		log.Printf("Error reading CSVs %s", err)
		os.Exit(1)
	}

	// Load schema
	schema := schema.Schema{}
	schemaBytes, err := os.ReadFile(schemaLoc)
	if err != nil {
		log.Printf("Error reading Schema %s", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(schemaBytes, &schema)
	if err != nil {
		log.Printf("Error unmarshaling schema %s", err)
		os.Exit(1)
	}

	// Convert to parquet
	err = parquet.WriteParquet(workerChan, dest, &schema)
	if err != nil {
		log.Printf("Unable to write into the Parquet file %s", err)
		os.Exit(1)
	}

	// Load into DuckDB
	db, err := duckdb.NewDB(fmt.Sprintf("%s/chick.db", dest), dest)
	if err != nil {
		log.Printf("Unable to insert into the database %s", err)
		os.Exit(1)
	}

	// Query Result rows
	res, err := db.Query(ctx, query)
	if err != nil {
		log.Printf("Unable to fetch query result %s", err)
		os.Exit(1)
	}

	err = rendering.Format(outputFormat)(res)
	if err != nil {
		log.Printf("Unable to render output query result %s", err)
		fmt.Println(res)
		os.Exit(1)
	}

}
