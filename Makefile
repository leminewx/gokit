GO ?= go
PACKAGES ?= ./...
TEST_FLAGS ?= -count=1 -p 1

.PHONY: all fmt fmt-check vet build test race cover lint check clean

all: check

# 格式化整仓代码（统一为 LF 行尾）
fmt:
	gofmt -w .

# 校验格式，有未格式化文件则以非零退出，供 CI 使用
fmt-check:
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then \
		echo "以下文件未格式化，请执行 make fmt："; \
		echo "$$files"; \
		exit 1; \
	fi

vet:
	$(GO) vet -p 1 $(PACKAGES)

build:
	$(GO) build -p 1 $(PACKAGES)

test:
	$(GO) test $(TEST_FLAGS) $(PACKAGES)

race:
	$(GO) test $(TEST_FLAGS) -race $(PACKAGES)

cover:
	$(GO) test $(TEST_FLAGS) -coverprofile=coverage.out $(PACKAGES)
	$(GO) tool cover -func=coverage.out

lint:
	golangci-lint run

check: fmt-check vet build test

clean:
	rm -f coverage.out