.PHONY: check fmt integration-test test vet

fmt:
	gofmt -w .

test:
	go test -race -cover ./...

vet:
	go vet ./...

integration-test:
	bash integration/run.sh

check: test vet integration-test
