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

package dev

import (
	"os"

	beeLogger "github.com/beego/bee/v2/logger"
)

// 这段代码的作用是初始化一个 Git 的 pre-commit 钩子，用于在提交代码之前自动运行一些代码质量工具，确保代码符合格式和质量要求。
// 具体来说，它设置了 goimports、ineffassign 和 staticcheck 工具，并且会在 Git 提交时执行这些工具来检查 Go 代码

// 使用 goimports 自动格式化 Go 代码，并将修改写回文件
// 使用 ineffassign 查找并报告 Go 代码中无用的赋值操作
// 运行 staticcheck 静态分析工具，使用特定的配置，禁用了某些检查（通过 -show-ignored 参数显示被忽略的检查，-checks 参数禁用一些检查）
var preCommit = `
goimports -w -format-only ./ \
ineffassign . \
staticcheck -show-ignored -checks "-ST1017,-U1000,-ST1005,-S1034,-S1012,-SA4006,-SA6005,-SA1019,-SA1024" ./ \
`

// for now, we simply override pre-commit file
// 这个函数的作用是设置 Git 的 pre-commit 钩子，把 preCommit 内容写入到 .git/hooks/pre-commit 文件中
func initGitHook() {
	// pcf => pre-commit file
	pcfPath := "./.git/hooks/pre-commit"
	pcf, err := os.OpenFile(pcfPath, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		beeLogger.Log.Errorf("try to create or open file failed: %s, cause: %s", pcfPath, err.Error())
		return
	}

	defer pcf.Close()
	_, err = pcf.Write(([]byte)(preCommit))

	if err != nil {
		beeLogger.Log.Errorf("could not init githooks: %s", err.Error())
	} else {
		beeLogger.Log.Successf("The githooks has been added, the content is:\n %s ", preCommit)
	}
}
