// Copyright 2013 bee authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package apiapp

import (
	"net/http"

	beeLogger "github.com/beego/bee/v2/logger"

	"os"

	"github.com/beego/bee/v2/cmd/commands"
	"github.com/beego/bee/v2/cmd/commands/version"
	"github.com/beego/bee/v2/utils"
)

var CmdServer = &commands.Command{
	// CustomFlags: true,
	UsageLine: "server [port]",
	Short:     "serving static content over HTTP on port",
	Long: `
  The command 'server' creates a Beego API application.
`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    createAPI,
}

var (
	a utils.DocValue
	p utils.DocValue
	f utils.DocValue
)

func init() {
	CmdServer.Flag.Var(&a, "a", "Listen address")
	CmdServer.Flag.Var(&p, "p", "Listen port")
	CmdServer.Flag.Var(&f, "f", "Static files fold")
	commands.AvailableCommands = append(commands.AvailableCommands, CmdServer)
}

func createAPI(cmd *commands.Command, args []string) int {
	// 首先检查 args 中是否包含参数，如果有，则使用 cmd.Flag.Parse 来解析命令行参数并将其传递给相应的标志（例如，-a、-p 和 -f）
	if len(args) > 0 {
		err := cmd.Flag.Parse(args[1:])
		if err != nil {
			beeLogger.Log.Error(err.Error())
		}
	}
	// 设置默认值
	if a == "" { // 如果 a（监听地址）为空，则默认为 127.0.0.1
		a = "127.0.0.1"
	}
	if p == "" { // 如果 p（监听端口）为空，则默认为 8080
		p = "8080"
	}
	if f == "" { // 如果 f（静态文件目录）为空，则使用当前工作目录作为静态文件目录
		cwd, _ := os.Getwd()
		f = utils.DocValue(cwd)
	}
	beeLogger.Log.Infof("Start server on http://%s:%s, static file %s", a, p, f)
	// 创建一个文件服务器，它会从指定目录 f 提供文件
	err := http.ListenAndServe(string(a)+":"+string(p), http.FileServer(http.Dir(f)))
	if err != nil {
		beeLogger.Log.Error(err.Error())
	}
	return 0
}
