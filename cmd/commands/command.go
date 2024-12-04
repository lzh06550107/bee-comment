package commands

import (
	"flag"
	"io"
	"os"
	"strings"

	"github.com/beego/bee/v2/logger/colors"
	"github.com/beego/bee/v2/utils"
)

// 定义了一个 Command 类型和一些与命令相关的操作，用于命令行工具中的命令管理

// Command is the unit of execution
// Command 结构体表示一个命令，包含了命令执行所需的各种信息
type Command struct {
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(cmd *Command, args []string) int // 执行命令的函数

	// PreRun performs an operation before running the command
	PreRun func(cmd *Command, args []string) // 命令执行前的预处理函数

	// UsageLine is the one-line Usage message.
	// The first word in the line is taken to be the command name.
	UsageLine string // 命令的用法描述

	// Short is the short description shown in the 'go help' output.
	Short string // 命令的简短描述

	// Long is the long message shown in the 'go help <this-command>' output.
	Long string // 命令的详细描述

	// Flag is a set of flags specific to this command.
	Flag flag.FlagSet // 命令的标志集合

	// CustomFlags indicates that the command will do its own
	// flag parsing.
	CustomFlags bool // 是否自定义标志解析

	// output out writer if set in SetOutput(w)
	output *io.Writer // 输出流
}

// AvailableCommands 是一个全局变量，保存所有可用的命令。命令在运行时会被注册到这个切片中，用于后续的处理和调用
var AvailableCommands = []*Command{}
var cmdUsage = `Use {{printf "bee help %s" .Name | bold}} for more information.{{endline}}`

// Name returns the command's name: the first word in the Usage line.
// 命令名称通常是 UsageLine 中的第一个单词。如果 UsageLine 包含空格，它会提取第一个单词作为命令名
func (c *Command) Name() string {
	name := c.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

// SetOutput sets the destination for Usage and error messages.
// If output is nil, os.Stderr is used.
// 用于设置命令的输出流，允许用户指定命令执行时将结果输出到哪里。如果未调用此方法，默认输出流是 os.Stderr
func (c *Command) SetOutput(output io.Writer) {
	c.output = &output
}

// Out returns the out writer of the current command.
// If cmd.output is nil, os.Stderr is used.
// 返回当前命令的输出流。如果 output 字段被设置，则返回 output；否则返回一个默认的 colors.NewColorWriter(os.Stderr)，用于输出带颜色的日志或消息
func (c *Command) Out() io.Writer {
	if c.output != nil {
		return *c.output
	}

	return colors.NewColorWriter(os.Stderr)
}

// Usage puts out the Usage for the command.
// 显示命令的使用信息。它调用了 utils.Tmpl() 来输出格式化的命令用法，并在完成后退出程序（os.Exit(2)）。
// 退出代码 2 表示程序因使用错误退出，通常用于指示命令行参数不正确
func (c *Command) Usage() {
	utils.Tmpl(cmdUsage, c)
	os.Exit(2)
}

// Runnable reports whether the command can be run; otherwise
// it is a documentation pseudo-command such as import path.
// 方法返回一个布尔值，表示命令是否可以执行。如果命令的 Run 函数被定义，返回 true，否则返回 false。这是用来判断命令是否可执行的
func (c *Command) Runnable() bool {
	return c.Run != nil
}

// 返回命令的所有标志（flags）的描述，包括标志的默认值和用途
func (c *Command) Options() map[string]string {
	options := make(map[string]string)
	// c.Flag.VisitAll 遍历所有标志，将标志名称和它们的用途描述收集到一个映射中
	c.Flag.VisitAll(func(f *flag.Flag) {
		defaultVal := f.DefValue
		// 如果标志有默认值，将 flag.Name=defaultValue 作为键，flag.Usage 作为值；否则，只记录 flag.Name 和 flag.Usage
		if len(defaultVal) > 0 {
			options[f.Name+"="+defaultVal] = f.Usage
		} else {
			options[f.Name] = f.Usage
		}
	})
	return options
}
