build:
	go build -o ./bin/chick

clean:
	rm -rf ./output

run: build clean
	./bin/chick --schema=./samples/schema.yaml \
		--format=table \
		--query="SELECT request_id, topic, model, region FROM DATA WHERE REGION='us-east-1';"