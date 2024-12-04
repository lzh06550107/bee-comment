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
package run

import (
	"io/ioutil"
	"os"
	path "path/filepath"
	"runtime"
	"strings"

	"github.com/beego/bee/v2/cmd/commands"
	"github.com/beego/bee/v2/cmd/commands/version"
	"github.com/beego/bee/v2/config"
	beeLogger "github.com/beego/bee/v2/logger"
	"github.com/beego/bee/v2/utils"
)

// 这段代码实现了 Beego 框架的 run 命令，用于启动本地开发服务器并监控文件变化。它在开发过程中自动重新编译和重启应用。

var CmdRun = &commands.Command{
	UsageLine: "run [appname] [watchall] [-main=*.go] [-downdoc=true]  [-gendoc=true] [-vendor=true] [-e=folderToExclude] [-ex=extraPackageToWatch] [-tags=goBuildTags] [-runmode=BEEGO_RUNMODE]",
	Short:     "Run the application by starting a local development server",
	Long: `
Run command will supervise the filesystem of the application for any changes, and recompile/restart it.

`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    RunApp,
}

var (
	mainFiles utils.ListOpts
	downdoc   utils.DocValue
	gendoc    utils.DocValue
	// The flags list of the paths excluded from watching
	excludedPaths utils.StrFlags
	// Pass through to -tags arg of "go build"
	buildTags string
	// Pass through to -ldflags arg of "go build"
	buildLDFlags string
	// Application path
	currpath string
	// Application name
	appname string
	// Channel to signal an Exit
	exit chan bool
	// Flag to watch the vendor folder
	vendorWatch bool
	// Current user workspace
	currentGoPath string
	// Current runmode
	runmode string
	// Extra args to run application
	runargs string
	// Extra directories
	extraPackages utils.StrFlags
)
var started = make(chan bool)

func init() {
	// main: 指定要监控的主 Go 文件
	CmdRun.Flag.Var(&mainFiles, "main", "Specify main go files.")
	// gendoc: 是否启用自动生成文档
	CmdRun.Flag.Var(&gendoc, "gendoc", "Enable auto-generate the docs.")
	// downdoc: 是否启用自动下载 Swagger 文件
	CmdRun.Flag.Var(&downdoc, "downdoc", "Enable auto-download of the swagger file if it does not exist.")
	// e: 排除某些路径
	CmdRun.Flag.Var(&excludedPaths, "e", "List of paths to exclude.")
	// vendor: 是否监控 vendor 文件夹
	CmdRun.Flag.BoolVar(&vendorWatch, "vendor", false, "Enable watch vendor folder.")
	// tags: 传递给 go build 的 build 标签
	CmdRun.Flag.StringVar(&buildTags, "tags", "", "Set the build tags. See: https://golang.org/pkg/go/build/")
	// 定义一个名为 -ldflags 的命令行标志，它允许用户为 go build 命令指定 ldflags 参数
	CmdRun.Flag.StringVar(&buildLDFlags, "ldflags", "", "Set the build ldflags. See: https://golang.org/pkg/go/build/")
	// runmode: 设置 Beego 运行模式（如 dev, prod）
	CmdRun.Flag.StringVar(&runmode, "runmode", "", "Set the Beego run mode.")
	// runargs: 启动应用时的额外参数
	CmdRun.Flag.StringVar(&runargs, "runargs", "", "Extra args to run application")
	// ex: 要额外监控的包
	CmdRun.Flag.Var(&extraPackages, "ex", "List of extra package to watch.")
	exit = make(chan bool)
	commands.AvailableCommands = append(commands.AvailableCommands, CmdRun)
}

// RunApp locates files to watch, and starts the beego application
// 执行命令时调用 RunApp 函数，启动应用并开始监控文件变化
func RunApp(cmd *commands.Command, args []string) int {
	// The default app path is the current working directory
	appPath, _ := os.Getwd() // 默认应用路径为当前工作目录

	// If an argument is presented, we use it as the app path
	// 如果传入了参数并且参数不是 watchall，那么根据传入的路径来确定应用路径。如果路径是相对路径，拼接成绝对路径
	if len(args) != 0 && args[0] != "watchall" {
		if path.IsAbs(args[0]) {
			appPath = args[0]
		} else {
			appPath = path.Join(appPath, args[0])
		}
	}

	// 判断是否在 GOPATH 中
	if utils.IsInGOPATH(appPath) {
		// 如果在 GOPATH 中，设置 appPath 为 GOPATH 下的路径
		if found, _gopath, _path := utils.SearchGOPATHs(appPath); found {
			appPath = _path
			appname = path.Base(appPath)
			currentGoPath = _gopath
		} else {
			beeLogger.Log.Fatalf("No application '%s' found in your GOPATH", appPath)
		}
		if strings.HasSuffix(appname, ".go") && utils.IsExist(appPath) {
			beeLogger.Log.Warnf("The appname is in conflict with file's current path. Do you want to build appname as '%s'", appname)
			beeLogger.Log.Info("Do you want to overwrite it? [yes|no] ")
			if !utils.AskForConfirmation() {
				return 0
			}
		}
	} else {
		beeLogger.Log.Warn("Running application outside of GOPATH")
		appname = path.Base(appPath)
		currentGoPath = appPath
	}

	beeLogger.Log.Infof("Using '%s' as 'appname'", appname)

	beeLogger.Log.Debugf("Current path: %s", utils.FILE(), utils.LINE(), appPath)

	// 设置运行模式
	// 根据传入的 runmode 参数（例如 prod 或 dev）设置 Beego 的运行模式。如果没有传入 runmode，则会使用环境变量 BEEGO_RUNMODE 中的值
	if runmode == "prod" || runmode == "dev" {
		os.Setenv("BEEGO_RUNMODE", runmode)
		beeLogger.Log.Infof("Using '%s' as 'runmode'", os.Getenv("BEEGO_RUNMODE"))
	} else if runmode != "" {
		os.Setenv("BEEGO_RUNMODE", runmode)
		beeLogger.Log.Warnf("Using '%s' as 'runmode'", os.Getenv("BEEGO_RUNMODE"))
	} else if os.Getenv("BEEGO_RUNMODE") != "" {
		beeLogger.Log.Warnf("Using '%s' as 'runmode'", os.Getenv("BEEGO_RUNMODE"))
	}

	// 读取应用程序目录，并开始监控文件
	var paths []string
	readAppDirectories(appPath, &paths)

	// Because monitor files has some issues, we watch current directory
	// and ignore non-go files.
	for _, p := range config.Conf.DirStruct.Others {
		paths = append(paths, strings.Replace(p, "$GOPATH", currentGoPath, -1))
	}

	// 如果有额外指定的包路径（extraPackages），则会查找这些路径并将其加入到需要监控的路径中
	if len(extraPackages) > 0 {
		// get the full path
		for _, packagePath := range extraPackages {
			if found, _, _fullPath := utils.SearchGOPATHs(packagePath); found {
				readAppDirectories(_fullPath, &paths)
			} else {
				beeLogger.Log.Warnf("No extra package '%s' found in your GOPATH", packagePath)
			}
		}
		// let paths unique
		strSet := make(map[string]struct{})
		for _, p := range paths {
			strSet[p] = struct{}{}
		}
		paths = make([]string, len(strSet))
		index := 0
		for i := range strSet {
			paths[index] = i
			index++
		}
	}

	files := []string{}
	for _, arg := range mainFiles {
		if len(arg) > 0 {
			files = append(files, arg)
		}
	}
	// 如果启用了 downdoc（下载文档），并且应用目录下没有 Swagger 文档，则会从指定的 URL 下载 Swagger 文档，并解压到本地
	if downdoc == "true" {
		if _, err := os.Stat(path.Join(appPath, "swagger", "index.html")); err != nil {
			if os.IsNotExist(err) {
				downloadFromURL(swaggerlink, "swagger.zip")
				unzipAndDelete("swagger.zip")
			}
		}
	}

	// Start the Reload server (if enabled) 如果启用了热重载（EnableReload），则启动重载服务器
	if config.Conf.EnableReload {
		startReloadServer()
	}
	// 开始监控文件并自动构建
	if gendoc == "true" { // 如果启用了文档生成（gendoc），则监控文件并启用自动构建
		NewWatcher(paths, files, true)
		AutoBuild(files, true)
	} else {
		// 否则，仅监控文件，但不进行自动构建
		NewWatcher(paths, files, false)
		AutoBuild(files, false)
	}

	// 进入一个无限循环，等待退出信号。一旦接收到退出信号，调用 runtime.Goexit() 退出当前 Goroutine
	for {
		<-exit
		runtime.Goexit()
	}
}

// 递归读取目录：这个函数会递归遍历指定目录，查找 Go 文件或其他需要监控的文件，并将其添加到监控路径列表中
// 该函数遍历指定的目录 directory，并根据一定的规则（如文件类型、是否是目录等）将符合条件的目录路径添加到 paths 列表中
func readAppDirectories(directory string, paths *[]string) {
	// 读取指定目录下的文件和子目录
	fileInfos, err := ioutil.ReadDir(directory)
	if err != nil {
		return
	}

	// 遍历当前目录中的每个文件或子目录。fileInfo 是每个文件或子目录的元数据（包含文件名、是否为目录等信息）
	useDirectory := false
	for _, fileInfo := range fileInfos {
		// 如果文件或目录名以 docs 或 swagger 结尾，则跳过这些文件或目录，不予处理
		if strings.HasSuffix(fileInfo.Name(), "docs") {
			continue
		}
		if strings.HasSuffix(fileInfo.Name(), "swagger") {
			continue
		}

		// 如果 vendorWatch 标志为 false 且当前目录是 vendor 目录，则跳过该目录。vendor 目录通常包含依赖项，可能不需要被监视
		if !vendorWatch && strings.HasSuffix(fileInfo.Name(), "vendor") {
			continue
		}

		// 调用 isExcluded 函数检查当前目录是否应该被排除。如果是，则跳过该目录
		if isExcluded(path.Join(directory, fileInfo.Name())) {
			continue
		}

		// 如果当前条目是一个子目录（fileInfo.IsDir()），并且目录名称不以 . 开头（即不是隐藏目录），则递归调用 readAppDirectories 继续遍历该子目录
		if fileInfo.IsDir() && fileInfo.Name()[0] != '.' {
			readAppDirectories(directory+"/"+fileInfo.Name(), paths)
			continue
		}

		// 如果当前目录已经被加入 paths 列表（useDirectory 标志为 true），则跳过
		if useDirectory {
			continue
		}

		// 如果文件是 Go 文件（扩展名为 .go），或者是符合某些条件的静态文件（通过 ifStaticFile 判断，且 config.Conf.EnableReload 为 true），
		// 则将当前目录路径添加到 paths 列表
		if path.Ext(fileInfo.Name()) == ".go" || (ifStaticFile(fileInfo.Name()) && config.Conf.EnableReload) {
			*paths = append(*paths, directory)
			useDirectory = true
		}
	}
}

// If a file is excluded
func isExcluded(filePath string) bool {
	for _, p := range excludedPaths {
		absP, err := path.Abs(p)
		if err != nil {
			beeLogger.Log.Errorf("Cannot get absolute path of '%s'", p)
			continue
		}
		absFilePath, err := path.Abs(filePath)
		if err != nil {
			beeLogger.Log.Errorf("Cannot get absolute path of '%s'", filePath)
			break
		}
		if strings.HasPrefix(absFilePath, absP) {
			beeLogger.Log.Infof("'%s' is not being watched", filePath)
			return true
		}
	}
	return false
}
