// Copyright 2017 bee authors
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
package utils

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"runtime"

	"github.com/beego/bee/v2/config"
)

// 代码实现了一个跨平台的通知系统，支持 macOS、Linux 和 Windows 系统，通过不同的通知工具进行操作

const appName = "Beego"

// 这个函数根据操作系统 (runtime.GOOS) 来调用相应的通知函数
func Notify(text, title string) {
	// 如果 config.Conf.EnableNotification 为 false，则不会发送通知
	if !config.Conf.EnableNotification {
		return
	}
	switch runtime.GOOS {
	case "darwin":
		osxNotify(text, title)
	case "linux":
		linuxNotify(text, title)
	case "windows":
		windowsNotify(text, title)
	}
}

// 根据系统上可用的工具来发送通知。首先检查 terminal-notifier 是否存在，
// 如果不存在，检查 macOS 版本并使用 osascript 发送通知。如果版本不支持，则使用 growlnotify。
func osxNotify(text, title string) {
	var cmd *exec.Cmd
	if existTerminalNotifier() {
		cmd = exec.Command("terminal-notifier", "-title", appName, "-message", text, "-subtitle", title)
	} else if MacOSVersionSupport() {
		notification := fmt.Sprintf("display notification \"%s\" with title \"%s\" subtitle \"%s\"", text, appName, title)
		cmd = exec.Command("osascript", "-e", notification)
	} else {
		cmd = exec.Command("growlnotify", "-n", appName, "-m", title)
	}
	cmd.Run()
}

// 使用 growlnotify 来发送通知。growlnotify 是一个适用于 Windows 系统的通知工具
func windowsNotify(text, title string) {
	exec.Command("growlnotify", "/i:", "", "/t:", title, text).Run()
}

// 使用 notify-send 来发送通知，这是 Linux 系统中常见的通知命令
func linuxNotify(text, title string) {
	exec.Command("notify-send", "-i", "", title, text).Run()
}

// 检查 terminal-notifier 是否存在，通过执行 which terminal-notifier 命令来判断。如果返回值为非空，则表示该工具已安装
func existTerminalNotifier() bool {
	cmd := exec.Command("which", "terminal-notifier")
	err := cmd.Start()
	if err != nil {
		return false
	}
	err = cmd.Wait()
	return err != nil
}

// 检查当前 macOS 的版本是否支持某些通知功能。只有在 macOS 10.9 或更高版本时，才使用 osascript 发送通知
func MacOSVersionSupport() bool {
	cmd := exec.Command("sw_vers", "-productVersion")
	check, _ := cmd.Output()
	version := strings.Split(string(check), ".")
	major, _ := strconv.Atoi(version[0])
	minor, _ := strconv.Atoi(version[1])
	if major < 10 || (major == 10 && minor < 9) {
		return false
	}
	return true
}
