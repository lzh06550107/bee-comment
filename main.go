// Copyright 2013 bee authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/beego/bee/v2/cmd"
	"github.com/beego/bee/v2/cmd/commands"
	"github.com/beego/bee/v2/config"
	"github.com/beego/bee/v2/utils"
)

func main() {
	// 检查是否需要更新 bee 工具
	utils.NoticeUpdateBee()
	flag.Usage = cmd.Usage // 用于输出命令行参数的使用说明
	flag.Parse()           // 解析命令行传入的参数
	// 用于设置日志的格式化选项。在 Go 的 log 包中，默认情况下，日志会显示时间、日期、文件名和行号等信息。
	// 这里将 SetFlags 设置为 0，表示不显示任何额外的日志信息，仅输出日志消息。
	log.SetFlags(0)

	// flag.Args() 返回一个字符串切片，包含解析命令行参数后，flag 包中没有被处理的所有额外参数。
	// 通常，flag 包用于处理命令行中的标志（如 -flag=value），而 flag.Args() 返回的是这些标志之外的其他命令行参数
	// 例如，假设你在命令行中输入了 ./program -flag1 value1 -flag2 value2 arg1 arg2，则 flag.Args() 会返回 ["arg1", "arg2"]
	args := flag.Args()

	// 如果没有子命令，则打印帮助信息并退出
	if len(args) < 1 {
		cmd.Usage()
		os.Exit(2)
		return
	}

	// 如果命令是 help，则打印帮助信息
	if args[0] == "help" {
		cmd.Help(args[1:])
		return
	}

	// 遍历 commands.AvailableCommands，寻找与用户输入相匹配的子命令
	for _, c := range commands.AvailableCommands {
		// 如果找到命令，解析参数并执行
		if c.Name() == args[0] && c.Run != nil {
			// 在 Go 的 flag 包中，FlagSet 结构体包含了一个 Usage 函数，用来输出该命令的使用帮助信息。
			// 在这里，我们将其指向 c.Usage，因此如果该命令的标志未正确解析，会调用 c.Usage() 输出命令的用法
			c.Flag.Usage = func() { c.Usage() }
			// 如果命令定义了自定义标志解析（CustomFlags 为 true），则跳过对标志的默认解析，只将参数 args 的第一个元素（命令名）去掉，剩下的部分直接作为命令的参数传递给 Run 函数
			if c.CustomFlags {
				args = args[1:]
			} else {
				// 如果命令没有自定义标志解析（CustomFlags 为 false），则使用 c.Flag.Parse(args[1:]) 来解析命令行的标志（flags）。
				// args[1:] 是去除命令名后的剩余部分（即命令行中标志和参数）。
				// Parse 会处理这些标志，并将它们存储在 Flag 对象中。然后，c.Flag.Args() 会返回剩余的非标志参数，这些参数会传递给命令的 Run 函数
				c.Flag.Parse(args[1:])
				args = c.Flag.Args()
			}

			// 如果命令有 PreRun，则在运行主逻辑前调用
			// PreRun 通常用于执行一些命令的前置操作，例如参数验证、环境准备等
			if c.PreRun != nil {
				c.PreRun(c, args)
			}

			// 加载配置文件（config.LoadConfig()）
			config.LoadConfig() // 加载一些全局配置，确保命令执行时使用正确的配置信息
			// 执行 Run 方法，完成命令逻辑
			os.Exit(c.Run(c, args))
			return
		}
	}

	// 如果找不到匹配的命令，调用 utils.PrintErrorAndExit 打印错误并退出
	utils.PrintErrorAndExit("Unknown subcommand", cmd.ErrorTemplate)
}
