run:
	go run main.go

build:
	go build -o app main.go

swagger:
	swag init --dir ./src/interface/fiber_server --output ./src/interface/fiber_server/docs

unit-test:
	go test -v ./src/...

coverage-test:
	go test -coverprofile cover.out ./src/...

coverage-test-html: coverage-test
	go tool cover -html=cover.out

benchmark-test:
	go test -bench=. -benchtime=10s -count 3 ./src/...
