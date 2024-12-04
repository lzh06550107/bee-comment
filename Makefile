# 这一行通过 grep 命令查找文件 cmd/commands/version/version.go 中定义的 const version，然后通过 sed 命令提取出版本号（例如 v1.2.3），
# 并将其赋值给 VERSION 变量。VERSION 变量会用于构建版本目录和发布版本文件
VERSION = $(shell grep 'const version' cmd/commands/version/version.go | sed -E 's/.*"(.+)"$$/v\1/')

# 使用 .PHONY 声明目标为伪目标后，make 会忽略文件检查，始终执行目标对应的命令
.PHONY: all test clean build install

GOFLAGS ?= $(GOFLAGS:)

# 默认目标（如果你只运行 make，它会默认执行 all）
# all 目标依赖于 install 和 test，所以 make all 会先执行 install，然后执行 test
all: install test

# 使用 go build 命令构建 Go 项目，$(GOFLAGS) 用于传递构建时的额外选项，./... 表示构建当前目录及其子目录下的所有 Go 文件
build:
	go build $(GOFLAGS) ./...

# 使用 go get 安装所有依赖项，$(GOFLAGS) 用于传递 Go 构建时的参数，./... 表示获取当前目录及其子目录下所有模块的依赖
install:
	go get $(GOFLAGS) ./...

# 该目标依赖于 install，因此会先安装所有依赖
# go test 用于运行项目的测试
test: install
	go test $(GOFLAGS) ./...

# 该目标用于执行性能基准测试（Benchmark），它与 test 目标类似，但是使用了 -bench 标志来运行基准测试
# -run=NONE 表示跳过普通的测试，只执行基准测试
bench: install
	go test -run=NONE -bench=. $(GOFLAGS) ./...

# 该目标用于清理 Go 项目，使用 go clean 删除编译缓存和临时文件，-i 选项会移除安装的包
clean:
	go clean $(GOFLAGS) -i ./...

# 该目标负责发布构建好的项目
publish:
	# 创建一个以 $(VERSION) 为名的目录
	mkdir -p bin/$(VERSION)
	cd bin/$(VERSION)
	# 使用 xgo 构建跨平台的二进制文件，--targets 参数指定构建的目标平台（包括 Windows、Darwin（macOS）、Linux 和 ARM）
	xgo -v -x --targets="windows/*,darwin/*,linux/386,linux/amd64,linux/arm-5,linux/arm64" -out bee_$(VERSION) github.com/beego/bee/v2
	cd ..
	# 使用 ghr 将构建好的二进制文件上传到 GitHub Release 中
	ghr -u beego -r bee $(VERSION) $(VERSION)
