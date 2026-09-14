package parquet

import (
	"chick/pkg/schema"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/snappy"
	"github.com/parquet-go/parquet-go/compress/zstd"
)

const batchCount = 5000

func WriteParquet(incomingCSVChan chan [][]string, destination string, parquetSchema *schema.Schema) error {

	err := os.MkdirAll(destination, os.ModePerm)
	if err != nil {
		return err
	}

	// Cast schema into Parquet node format
	parquetFields := make(map[string]parquet.Node, len(parquetSchema.Spec.Fields))
	for _, f := range parquetSchema.Spec.Fields {
		switch f.Type {
		case "string":
			parquetFields[f.Name] = encode(parquet.String(), f.Encoding)
		case "integer":
			parquetFields[f.Name] = encode(parquet.Int(32), f.Encoding)
		case "datetime":
			parquetFields[f.Name] = encode(parquet.Timestamp(parquet.Millisecond), f.Encoding)
		case "float":
			parquetFields[f.Name] = encode(parquet.Leaf(parquet.FloatType), f.Encoding)
		case "boolean":
			parquetFields[f.Name] = encode(parquet.Leaf(parquet.BooleanType), f.Encoding)
		case "date":
			parquetFields[f.Name] = encode(parquet.Date(), f.Encoding)
		case "double":
			parquetFields[f.Name] = encode(parquet.Leaf(parquet.DoubleType), f.Encoding)
		case "default":
			return fmt.Errorf("Type not supported")
		}
	}

	schema := parquet.NewSchema("generic", parquet.Group(parquetFields))

	var count = 0
	currentFileWriter, closeFn, err := createFile(destination, schema, parquetSchema)
	if err != nil {
		return err
	}

	for records := range incomingCSVChan {
		for _, record := range records {
			if count >= batchCount {
				count = 0
				closeFn()
				currentFileWriter, closeFn, err = createFile(destination, schema, parquetSchema)
				if err != nil {
					return err
				}
			}

			columns := make([][]parquet.Value, len(schema.Columns()))
			for i, field := range record {
				fieldName := parquetSchema.Spec.Fields[i].Name
				leafIdx, ok := schema.Lookup(fieldName)
				if !ok {
					continue
				}

				pqVal := parquet.ValueOf(convertField(field, parquetSchema.Spec.Fields[i].Type)).Level(0, 1, leafIdx.ColumnIndex)

				columns[leafIdx.ColumnIndex] = []parquet.Value{pqVal}
			}

			row := parquet.MakeRow(columns...)

			_, err = currentFileWriter.WriteRows([]parquet.Row{row})
			if err != nil {
				return err
			}

			count += 1
		}

	}
	closeFn()

	return nil
}

func createFile(destination string, schema *parquet.Schema, parquetSchema *schema.Schema) (*parquet.GenericWriter[any], func(), error) {
	n := time.Now()
	filename := fmt.Sprintf("output_%d_%d_%d_%d_%d_%d_%d.parquet", n.Day(), n.Month(), n.Year(), n.Hour(), n.Minute(), n.Second(), n.Nanosecond())
	fileLoc := filepath.Join(destination, filename)
	f, err := os.Create(fileLoc)
	if err != nil {
		return nil, nil, err
	}

	writerOptions := []parquet.WriterOption{
		schema,
	}

	switch parquetSchema.Spec.Compression {
	case "zstd":
		writerOptions = append(writerOptions, parquet.Compression(&zstd.Codec{
			Level: zstd.SpeedDefault,
		}))
	case "snappy":
		writerOptions = append(writerOptions, parquet.Compression(&snappy.Codec{}))
	}

	writer := parquet.NewGenericWriter[any](f, writerOptions...)

	return writer, func() {
		writer.Close()
		f.Close()
	}, nil
}

func convertField(field string, schemaType string) any {
	switch schemaType {
	case "boolean":
		return field == "true"
	case "string":
		return field
	case "integer":
		i, _ := strconv.Atoi(field)
		return i
	case "float":
		f, _ := strconv.ParseFloat(field, 32)
		return float32(f)
	case "datetime":
		// TODO: Figure out date time conversion
		parsedTime, _ := time.Parse(time.RFC3339Nano, field)
		return parsedTime.UnixMilli()
	}

	return nil
}

func encode(f parquet.Node, encodingType string) parquet.Node {
	switch encodingType {
	case "delta":
		return parquet.Encoded(f, &parquet.DeltaBinaryPacked)
	case "prefix":
		return parquet.Encoded(f, &parquet.DeltaByteArray)
	case "dictionary":
		return parquet.Encoded(f, &parquet.RLEDictionary)
	default:
		return f
	}
}
