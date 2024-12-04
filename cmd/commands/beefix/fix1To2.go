// Copyright 2020
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package beefix

import (
	"os"
	"os/exec"

	beeLogger "github.com/beego/bee/v2/logger"
)

// 主要功能是将 Beego 从 1.x 版本升级到 2.x 版本，并且对项目中的 Go 文件进行必要的修改。
// 它通过执行 shell 命令来更新依赖并替换源代码中的旧的 import 路径

// 该函数的作用是通过执行 Shell 命令来完成 Beego 项目的升级
func fix1To2() int {
	beeLogger.Log.Info("Upgrading the application...")

	// 使用 go get 命令获取 Beego v2 的最新版本
	cmdStr := `go get -u github.com/beego/beego/v2@master`
	err := runShell(cmdStr)
	if err != nil {
		beeLogger.Log.Error(err.Error())
		beeLogger.Log.Error(`fetch v2.0.1 failed. Please try to run: export GO111MODULE=on
and if your network is not stable, please try to use proxy, for example: export GOPROXY=https://goproxy.cn;'
`)
		return 1
	}

	// 更新代码中的 Beego import 路径： 使用 find 命令配合 sed 命令，更新所有 Go 文件中的 Beego import 路径
	// 将 github.com/astaxie/beego 替换为 github.com/beego/beego/v2/adapter
	cmdStr = `find ./ -name '*.go' -type f -exec sed -i '' -e 's/github.com\/astaxie\/beego/github.com\/beego\/beego\/v2\/adapter/g' {} \;`
	err = runShell(cmdStr)
	if err != nil {
		beeLogger.Log.Error(err.Error())
		return 1
	}
	// 确保 Beego 的 import 声明符合新的格式。
	// find 命令会遍历当前目录下的所有 .go 文件，并对每个文件执行 sed 替换操作
	cmdStr = `find ./ -name '*.go' -type f -exec sed -i '' -e 's/"github.com\/beego\/beego\/v2\/adapter"/beego "github.com\/beego\/beego\/v2\/adapter"/g' {} \;`
	err = runShell(cmdStr)
	if err != nil {
		beeLogger.Log.Error(err.Error())
		return 1
	}
	return 0
}

// 用于执行一个 Shell 命令，并输出命令的标准输出。如果命令执行失败，会返回错误。
func runShell(cmdStr string) error {
	// 启动一个新的 Shell 进程来执行给定的命令
	c := exec.Command("sh", "-c", cmdStr)
	c.Stdout = os.Stdout
	err := c.Run()
	if err != nil {
		beeLogger.Log.Errorf("execute command [%s] failed: %s", cmdStr, err.Error())
		return err
	}
	return nil
}
