run:
	go run ./cmd/elevator --id=1 --port=12345

run-multi:
	go run ./cmd/elevator --id=1 --port=12345 &
	go run ./cmd/elevator --id=2 --port=12346 &
	go run ./cmd/elevator --id=3 --port=12347 &

test:
	go test ./... -v
race:
	go test -race ./...
