// Copyright 2017 bee authors
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

// Package rs ...
package rs

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"strings"

	"github.com/beego/bee/v2/cmd/commands"
	"github.com/beego/bee/v2/cmd/commands/version"
	"github.com/beego/bee/v2/config"
	"github.com/beego/bee/v2/logger"
	"github.com/beego/bee/v2/logger/colors"
	"github.com/beego/bee/v2/utils"
)

var cmdRs = &commands.Command{
	UsageLine: "rs",
	Short:     "Run customized scripts",
	Long: `Run script allows you to run arbitrary commands using Bee.
  Custom commands are provided from the "scripts" object inside bee.json or Beefile.

  To run a custom command, use: {{"$ bee rs mycmd ARGS" | bold}}
  {{if len .}}
{{"AVAILABLE SCRIPTS"|headline}}{{range $cmdName, $cmd := .}}
  {{$cmdName | bold}}
      {{$cmd}}{{end}}{{end}}
`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    runScript,
}

func init() {
	// 加载 bee.json 或 Beefile 配置文件，获取 scripts 配置并生成命令帮助信息
	config.LoadConfig()
	cmdRs.Long = utils.TmplToString(cmdRs.Long, config.Conf.Scripts)
	commands.AvailableCommands = append(commands.AvailableCommands, cmdRs)
}

func runScript(cmd *commands.Command, args []string) int {
	if len(args) == 0 {
		cmd.Usage()
	}

	start := time.Now()
	// 从传入的 args 中提取出第一个参数作为脚本的名称（script），其余部分作为参数列表（args）
	script, args := args[0], args[1:]

	// 检查 script 是否在 bee.json 或 Beefile 配置中的 Scripts 字段中定义。如果定义了，c 是脚本对应的命令字符串。
	if c, exist := config.Conf.Scripts[script]; exist {
		command := customCommand{
			Name:    script,
			Command: c,
			Args:    args,
		}
		if err := command.run(); err != nil {
			beeLogger.Log.Error(err.Error())
		}
	} else {
		beeLogger.Log.Errorf("Command '%s' not found in Beefile/bee.json", script)
	}
	elapsed := time.Since(start) // 通过 time.Since(start) 计算脚本执行的时间，并输出信息
	fmt.Println(colors.GreenBold(fmt.Sprintf("Finished in %s.", elapsed)))
	return 0
}

type customCommand struct {
	Name    string   // 脚本名称
	Command string   // 脚本要执行的命令字符串
	Args    []string // 传递给脚本的参数
}

func (c *customCommand) run() error {
	beeLogger.Log.Info(colors.GreenBold(fmt.Sprintf("Running '%s'...", c.Name)))
	var cmd *exec.Cmd
	switch runtime.GOOS {
	// 如果是 macOS 或 Linux 系统，使用 sh -c 来执行命令。这是因为 sh 是 Unix-like 系统的默认命令解释器，-c 参数表示运行一个命令字符串。
	// 通过 strings.Join(args, " ") 将命令和参数拼接成一个完整的字符串
	case "darwin", "linux":
		args := append([]string{c.Command}, c.Args...)
		cmd = exec.Command("sh", "-c", strings.Join(args, " "))
	case "windows":
		// 如果是 Windows 系统，使用 cmd /C 来执行命令。/C 参数表示执行完命令后关闭命令提示符窗口，同样将命令和参数拼接成一个字符串
		args := append([]string{c.Command}, c.Args...)
		cmd = exec.Command("cmd", "/C", strings.Join(args, " "))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
