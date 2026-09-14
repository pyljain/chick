package csv

import (
	"encoding/csv"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const workerCount = 10

func Collect(loc string) (chan [][]string, error) {
	collectorChan := make(chan [][]string)
	workerChan := make(chan struct{}, workerCount)
	wg := sync.WaitGroup{}
	err := filepath.WalkDir(loc, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".csv" {
			return nil
		}

		wg.Add(1)

		go func() {
			workerChan <- struct{}{}
			defer func() {
				<-workerChan
				wg.Done()
			}()
			f, err := os.Open(path)
			if err != nil {
				log.Printf("Error opening file %s", err)
				return
			}
			defer f.Close()

			reader := csv.NewReader(f)
			records, err := reader.ReadAll()
			if err != nil {
				log.Printf("Error reading file %s", err)
				return
			}

			collectorChan <- records[1:]
		}()

		return nil

	})
	if err != nil {
		return nil, err
	}

	go func() {
		wg.Wait()
		close(collectorChan)
	}()

	return collectorChan, nil
}
