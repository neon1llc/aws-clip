.PHONY: build check clean generate check-models

GOCACHE ?= $(CURDIR)/.cache/go-build
export GOCACHE

build:
	mkdir -p bin
	go build -o bin/aws-clip ./cmd/aws-clip

check:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt required for:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	go vet ./...
	go test ./...

# Regenerate the sensitive-output list from botocore's service models, installed at the pinned release
BOTOCORE_DATA = $$(python3 -c 'import botocore, os; print(os.path.dirname(botocore.__file__))')/data

generate:
	python3 -m pip install -r tools/genops/requirements.txt
	go run ./tools/genops -models "$(BOTOCORE_DATA)"

check-models:
	go run ./tools/genops -models "$(BOTOCORE_DATA)" -check

clean:
	rm -f bin/aws-clip
