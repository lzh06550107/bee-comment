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

package run

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/beego/bee/v2/config"
	beeLogger "github.com/beego/bee/v2/logger"
	"github.com/beego/bee/v2/logger/colors"
	"github.com/beego/bee/v2/utils"
	"github.com/fsnotify/fsnotify"
)

var (
	cmd                 *exec.Cmd
	state               sync.Mutex
	eventTime           = make(map[string]int64)
	scheduleTime        time.Time
	watchExts           = config.Conf.WatchExts
	watchExtsStatic     = config.Conf.WatchExtsStatic
	ignoredFilesRegExps = []string{
		`.#(\w+).go$`,
		`.(\w+).go.swp$`,
		`(\w+).go~$`,
		`(\w+).tmp$`,
		`commentsRouter_controllers.go$`,
	}
)

// NewWatcher starts an fsnotify Watcher on the specified paths
// 用于初始化文件系统监控器并监控指定路径的文件变化。当监测到文件变动时，触发自动构建或重新加载操作
func NewWatcher(paths []string, files []string, isgenerate bool) {
	// 使用 fsnotify 库监控文件系统的变化，特别是指定的目录和文件。当监控到文件变化时，会根据配置自动执行构建或刷新操作
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		beeLogger.Log.Fatalf("Failed to create watcher: %s", err)
	}

	// 启动 Goroutine 监听文件变化
	go func() {
		for {
			select {
			case e := <-watcher.Events: // 文件系统发生变化时触发的事件
				// 当监控到文件系统的变化时，会进入 watcher.Events 通道并触发此代码块
				isBuild := true

				// 检查文件是否是静态文件。如果是静态文件，并且配置中启用了自动刷新（EnableReload），则调用 sendReload 发送重新加载信号
				if ifStaticFile(e.Name) && config.Conf.EnableReload {
					sendReload(e.String())
					continue
				}
				// Skip ignored files
				// 如果该文件被标记为忽略文件，则跳过该文件
				if shouldIgnoreFile(e.Name) {
					continue
				}
				// 检查文件扩展名是否符合监控条件，若不符合，则跳过
				if !shouldWatchFileWithExtension(e.Name) {
					continue
				}

				// 防止重复构建
				mt := utils.GetFileModTime(e.Name) // 获取文件的修改时间
				// 如果文件的修改时间与上次记录的时间相同，表示文件未发生实际变化，因此跳过该文件的构建（isBuild = false）
				if t := eventTime[e.Name]; mt == t {
					beeLogger.Log.Hintf(colors.Bold("Skipping: ")+"%s", e.String())
					isBuild = false
				}

				// 如果文件发生变化，则更新记录的时间
				eventTime[e.Name] = mt

				// 如果 isBuild 为 true，表示文件发生了变化且需要进行构建
				if isBuild {
					beeLogger.Log.Hintf("Event fired: %s", e)
					go func() {
						// Wait 1s before autobuild until there is no file change.
						// 在 1 秒钟的延时后调用 AutoBuild(files, isgenerate) 执行自动构建。延时是为了等待其他文件变化，避免多次触发构建
						scheduleTime = time.Now().Add(1 * time.Second)
						time.Sleep(time.Until(scheduleTime))
						// 负责根据文件变化自动构建应用。files 是需要构建的文件列表，isgenerate 标志是否生成文档等
						AutoBuild(files, isgenerate)

						// 如果启用了 EnableReload，则再延迟 100 毫秒后发送重新加载信号（sendReload），通知浏览器刷新
						if config.Conf.EnableReload {
							// Wait 100ms more before refreshing the browser
							time.Sleep(100 * time.Millisecond)
							sendReload(e.String()) // 向前端发送重新加载的请求，以便在文件变化后刷新浏览器
						}
					}()
				}
			case err := <-watcher.Errors: // 监控器发生错误时触发的事件
				beeLogger.Log.Warnf("Watcher error: %s", err.Error()) // No need to exit here
			}
		}
	}()

	beeLogger.Log.Info("Initializing watcher...")
	for _, path := range paths {
		beeLogger.Log.Hintf(colors.Bold("Watching: ")+"%s", path)
		err = watcher.Add(path) // 添加路径到监控列表
		if err != nil {
			beeLogger.Log.Fatalf("Failed to watch directory: %s", err)
		}
	}
}

// AutoBuild builds the specified set of files
// 它的主要任务是根据指定的文件集和构建标志进行自动构建，并在必要时生成文档
// AutoBuild 函数用于构建指定的 Go 应用程序。它会根据配置和条件执行以下操作：
//
// 1. 如果启用了文档生成，生成应用文档。
// 2. 使用 go install 或 go build 构建应用程序。
// 3. 在构建成功后，调用 Restart 函数重启应用。
func AutoBuild(files []string, isgenerate bool) {
	state.Lock()
	defer state.Unlock()

	// 将当前工作目录更改为 currpath，确保构建命令在正确的目录下执行
	os.Chdir(currpath)

	// 设置使用的命令行工具为 go，即使用 Go 命令进行构建
	cmdName := "go"

	var (
		err    error
		stderr bytes.Buffer
	)
	// For applications use full import path like "github.com/.../.."
	// are able to use "go install" to reduce build time.
	// 执行 go install
	// 如果配置中启用了 GoInstall，则通过 go install 命令安装应用程序，减少构建时间。-v 参数会显示安装过程中的详细信息
	if config.Conf.GoInstall {
		icmd := exec.Command(cmdName, "install", "-v")
		icmd.Stdout = os.Stdout
		icmd.Stderr = os.Stderr
		icmd.Env = append(os.Environ(), "GOGC=off") // 设置 GOGC=off 环境变量，禁用 Go 的垃圾回收，以提高构建性能
		icmd.Run()
	}

	// 生成文档
	// 如果 isgenerate 为 true，表示需要生成文档，调用 bee generate docs 命令生成文档
	if isgenerate {
		beeLogger.Log.Info("Generating the docs...")
		icmd := exec.Command("bee", "generate", "docs")
		icmd.Env = append(os.Environ(), "GOGC=off")
		err = icmd.Run()
		if err != nil {
			utils.Notify("", "Failed to generate the docs.")
			beeLogger.Log.Errorf("Failed to generate the docs.")
			return
		}
		beeLogger.Log.Success("Docs generated!")
	}

	// 构建应用程序
	appName := appname
	if err == nil {
		// 设置构建目标文件名 appName，如果是 Windows 系统，会加上 .exe 扩展名
		if runtime.GOOS == "windows" {
			appName += ".exe"
		}

		args := []string{"build"}
		args = append(args, "-o", appName) // 指定输出文件名
		if buildTags != "" {
			args = append(args, "-tags", buildTags) // 指定构建时使用的构建标签
		}
		if buildLDFlags != "" {
			args = append(args, "-ldflags", buildLDFlags) // 指定链接器标志
		}
		args = append(args, files...) // 构建指定的 Go 源代码文件

		bcmd := exec.Command(cmdName, args...)
		bcmd.Env = append(os.Environ(), "GOGC=off")
		bcmd.Stderr = &stderr
		err = bcmd.Run()
		if err != nil {
			utils.Notify(stderr.String(), "Build Failed")
			beeLogger.Log.Errorf("Failed to build the application: %s", stderr.String())
			return
		}
	}

	beeLogger.Log.Success("Built Successfully!")
	Restart(appName) // 构建成功后重启应用
}

// Kill kills the running command process
func Kill() {
	defer func() {
		if e := recover(); e != nil {
			beeLogger.Log.Infof("Kill recover: %s", e)
		}
	}()
	if cmd != nil && cmd.Process != nil {
		// Windows does not support Interrupt
		if runtime.GOOS == "windows" {
			cmd.Process.Signal(os.Kill)
		} else {
			cmd.Process.Signal(os.Interrupt)
		}

		ch := make(chan struct{}, 1)
		go func() {
			cmd.Wait()
			ch <- struct{}{}
		}()

		select {
		case <-ch:
			return
		case <-time.After(10 * time.Second):
			beeLogger.Log.Info("Timeout. Force kill cmd process")
			err := cmd.Process.Kill()
			if err != nil {
				beeLogger.Log.Errorf("Error while killing cmd process: %s", err)
			}
			return
		}
	}
}

// Restart kills the running command process and starts it again
func Restart(appname string) {
	beeLogger.Log.Debugf("Kill running process", utils.FILE(), utils.LINE())
	Kill()
	go Start(appname)
}

// Start starts the command process
func Start(appname string) {
	beeLogger.Log.Infof("Restarting '%s'...", appname)
	if !strings.Contains(appname, "./") {
		appname = "./" + appname
	}

	cmd = exec.Command(appname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if runargs != "" {
		r := regexp.MustCompile("'.+'|\".+\"|\\S+")
		m := r.FindAllString(runargs, -1)
		cmd.Args = append([]string{appname}, m...)
	} else {
		cmd.Args = append([]string{appname}, config.Conf.CmdArgs...)
	}
	cmd.Env = append(os.Environ(), config.Conf.Envs...)

	go cmd.Run()
	beeLogger.Log.Successf("'%s' is running...", appname)
	started <- true
}

func ifStaticFile(filename string) bool {
	for _, s := range watchExtsStatic {
		if strings.HasSuffix(filename, s) {
			return true
		}
	}
	return false
}

// shouldIgnoreFile ignores filenames generated by Emacs, Vim or SublimeText.
// It returns true if the file should be ignored, false otherwise.
func shouldIgnoreFile(filename string) bool {
	for _, regex := range ignoredFilesRegExps {
		r, err := regexp.Compile(regex)
		if err != nil {
			beeLogger.Log.Fatalf("Could not compile regular expression: %s", err)
		}
		if r.MatchString(filename) {
			return true
		}
		continue
	}
	return false
}

// shouldWatchFileWithExtension returns true if the name of the file
// hash a suffix that should be watched.
func shouldWatchFileWithExtension(name string) bool {
	for _, s := range watchExts {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}
