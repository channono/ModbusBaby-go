# 大牛大巨婴 - ModbusBaby Go版本构建脚本
APP_NAME=ModbusBaby
VERSION=2.0.0

.PHONY: build run clean deps

# 安装依赖
deps:
	go mod tidy
	go mod download

# 构建应用
build: deps
	@echo "构建 ModbusBaby Go版本..."
	go build -ldflags "-X main.version=$(VERSION)" -o build/$(APP_NAME) main.go

# 运行应用
run: deps
	@echo "运行 ModbusBaby..."
	go run internal/main.go

# 跨平台构建
build-all: deps
	@echo "构建所有平台版本..."
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o build/$(APP_NAME)-windows.exe internal/main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o build/$(APP_NAME)-macos internal/main.go
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o build/$(APP_NAME)-linux internal/main.go

# 清理构建文件
clean:
	rm -rf build/
	go clean

# 测试
test:
	go test ./...

# 格式化代码
fmt:
	go fmt ./...

# 检查代码
vet:
	go vet ./...