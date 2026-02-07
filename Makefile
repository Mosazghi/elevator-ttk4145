run:
	go run ./cmd/elevator --id=1 --port=15657

run-multi:
	go run ./cmd/elevator --id=1 --port=15657 &
	go run ./cmd/elevator --id=2 --port=15658 &
	go run ./cmd/elevator --id=3 --port=15659 &

test:
	go test ./... -v
race:
	go test -race ./...
