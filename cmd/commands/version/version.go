package version

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"runtime"

	"github.com/beego/bee/v2/cmd/commands"
	"github.com/beego/bee/v2/config"
	beeLogger "github.com/beego/bee/v2/logger"
	"github.com/beego/bee/v2/logger/colors"
)

// 这段代码定义了 version 包，提供了一个用于显示当前 Bee 项目的版本信息的命令。
// 该命令可以以不同的输出格式（如 JSON、YAML 或标准文本）输出版本信息、Go 环境信息以及平台的相关信息。

// 用于详细版本信息的显示，包括 Bee 项目的版本、Go 版本、操作系统、CPU 信息等
const verboseVersionBanner string = `%s%s______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v{{ .BeeVersion }}%s
%s%s
├── GoVersion : {{ .GoVersion }}
├── GOOS      : {{ .GOOS }}
├── GOARCH    : {{ .GOARCH }}
├── NumCPU    : {{ .NumCPU }}
├── GOPATH    : {{ .GOPATH }}
├── GOROOT    : {{ .GOROOT }}
├── Compiler  : {{ .Compiler }}
└── Date      : {{ Now "Monday, 2 Jan 2006" }}%s
`

// 是一个简化的版本信息显示，仅显示 Bee 项目的版本号
const shortVersionBanner = `______
| ___ \
| |_/ /  ___   ___
| ___ \ / _ \ / _ \
| |_/ /|  __/|  __/
\____/  \___| \___| v{{ .BeeVersion }}
`

// CmdVersion 命令定义
// CmdVersion 是一个用于显示版本信息的命令，命令的执行逻辑由 versionCmd 函数处理
var CmdVersion = &commands.Command{
	UsageLine: "version",
	Short:     "Prints the current Bee version",
	Long: `
Prints the current Bee, Beego and Go version alongside the platform information.
`,
	Run: versionCmd,
}

// 用于控制输出格式，允许用户通过 -o 参数指定输出为 json 或 yaml
var outputFormat string

const version = config.Version

// 在 init 函数中，定义了一个标志（flag）用于接收命令行参数
func init() {
	// 这里使用 flag.NewFlagSet 创建一个新的标志集（FlagSet），名称为 "version"，并指定错误处理策略为 flag.ContinueOnError，
	// 即遇到解析错误时不会直接退出程序，而是继续执行后续的命令行解析
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	// 定义了一个名为 -o 的命令行标志，用于指定输出格式
	fs.StringVar(&outputFormat, "o", "", "Set the output format. Either json or yaml.")
	// 将创建的标志集 fs 绑定到 CmdVersion 命令的 Flag 字段上，意味着这个命令会使用该标志集来解析命令行参数
	CmdVersion.Flag = *fs
	// 这行代码将 CmdVersion 命令添加到 commands.AvailableCommands 中
	commands.AvailableCommands = append(commands.AvailableCommands, CmdVersion)
}

func versionCmd(cmd *commands.Command, args []string) int {

	// 该函数首先解析命令行参数
	cmd.Flag.Parse(args)
	stdout := cmd.Out()

	// 如果 outputFormat 被设置为 json 或 yaml，则会构建一个 RuntimeInfo 结构体，
	// 其中包含当前 Go 环境和 Bee 项目的版本信息。然后，选择合适的格式（JSON 或 YAML）输出。
	if outputFormat != "" {
		runtimeInfo := RuntimeInfo{
			GoVersion:  runtime.Version(),
			GOOS:       runtime.GOOS,
			GOARCH:     runtime.GOARCH,
			NumCPU:     runtime.NumCPU(),
			GOPATH:     os.Getenv("GOPATH"),
			GOROOT:     runtime.GOROOT(),
			Compiler:   runtime.Compiler,
			BeeVersion: version,
		}
		switch outputFormat {
		case "json":
			{
				b, err := json.MarshalIndent(runtimeInfo, "", "    ")
				if err != nil {
					beeLogger.Log.Error(err.Error())
				}
				fmt.Println(string(b))
				return 0
			}
		case "yaml":
			{
				b, err := yaml.Marshal(&runtimeInfo)
				if err != nil {
					beeLogger.Log.Error(err.Error())
				}
				fmt.Println(string(b))
				return 0
			}
		}
	}

	coloredBanner := fmt.Sprintf(verboseVersionBanner, "\x1b[35m", "\x1b[1m",
		"\x1b[0m", "\x1b[32m", "\x1b[1m", "\x1b[0m")
	// 函数用于将版本信息格式化并输出到标准输出
	InitBanner(stdout, bytes.NewBufferString(coloredBanner))
	return 0
}

// ShowShortVersionBanner prints the short version banner.
func ShowShortVersionBanner() {
	output := colors.NewColorWriter(os.Stdout)
	InitBanner(output, bytes.NewBufferString(colors.MagentaBold(shortVersionBanner)))
}
