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
package bale

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/beego/bee/v2/cmd/commands"
	"github.com/beego/bee/v2/cmd/commands/version"
	"github.com/beego/bee/v2/config"
	beeLogger "github.com/beego/bee/v2/logger"
	"github.com/beego/bee/v2/utils"
)

// 这段代码实现了一个名为 bale 的命令，主要用于将静态资源文件（如 JS、CSS、图片等）打包成 Go 代码，并生成一个压缩后的二进制文件。
// 这样做的目的是在部署 Go 应用时，不需要额外携带静态资源文件，而是将它们直接打包进 Go 可执行文件中。

// 代码的主要功能：
// 1. 静态资源打包：将指定的静态资源（文件夹中的文件）压缩并转换为 Go 代码文件，最终生成一个包含所有资源的 Go 文件。
// 2. 生成自动解压功能：为每个静态资源文件生成一个对应的 Go 函数，这些函数用于在运行时解压资源文件并将其保存到本地磁盘。
// 3. 资源文件的压缩：所有资源文件在打包时都会被 gzip 压缩，减少二进制文件的体积。

var CmdBale = &commands.Command{
	UsageLine: "bale",
	Short:     "Transforms non-Go files to Go source files",
	Long: `Bale command compress all the static files in to a single binary file.

  This is useful to not have to carry static files including js, css, images and
  views when deploying a Web application.

  It will auto-generate an unpack function to the main package then run it during the runtime.
  This is mainly used for zealots who are requiring 100% Go code.
`,
	PreRun: func(cmd *commands.Command, args []string) { version.ShowShortVersionBanner() },
	Run:    runBale,
}

func init() {
	commands.AvailableCommands = append(commands.AvailableCommands, CmdBale)
}

func runBale(cmd *commands.Command, args []string) int {
	// 首先删除并创建一个名为 bale 的目录，然后遍历配置中的 Dirs 路径，将指定目录中的所有静态文件进行打包和压缩
	os.RemoveAll("bale")
	os.Mkdir("bale", os.ModePerm)

	// Pack and compress data 每个文件会被压缩成 gzip 格式，并生成对应的 Go 文件。这些 Go 文件包含了资源的解压和保存逻辑
	for _, p := range config.Conf.Bale.Dirs {
		if !utils.IsExist(p) {
			beeLogger.Log.Warnf("Skipped directory: %s", p)
			continue
		}
		beeLogger.Log.Infof("Packaging directory: %s", p)
		filepath.Walk(p, walkFn)
	}

	// Generate auto-uncompress function.
	buf := new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf(BaleHeader, config.Conf.Bale.Import,
		strings.Join(resFiles, "\",\n\t\t\""),
		strings.Join(resFiles, ",\n\t\tbale.R")))

	fw, err := os.Create("bale.go")
	if err != nil {
		beeLogger.Log.Fatalf("Failed to create file: %s", err)
	}
	defer fw.Close()

	_, err = fw.Write(buf.Bytes())
	if err != nil {
		beeLogger.Log.Fatalf("Failed to write data: %s", err)
	}

	beeLogger.Log.Success("Baled resources successfully!")
	return 0
}

const (
	// BaleHeader ...
	BaleHeader = `package main

import(
	"os"
	"strings"
	"path"

	"%s"
)

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func init() {
	files := []string{
		"%s",
	}

	funcs := []func() []byte{
		bale.R%s,
	}

	for i, f := range funcs {
		fp := getFilePath(files[i])
		if !isExist(fp) {
			saveFile(fp, f())
		}
	}
}

func getFilePath(name string) string {
	name = strings.Replace(name, "_4_", "/", -1)
	name = strings.Replace(name, "_3_", " ", -1)
	name = strings.Replace(name, "_2_", "-", -1)
	name = strings.Replace(name, "_1_", ".", -1)
	name = strings.Replace(name, "_0_", "_", -1)
	return name
}

func saveFile(filePath string, b []byte) (int, error) {
	os.MkdirAll(path.Dir(filePath), os.ModePerm)
	fw, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer fw.Close()
	return fw.Write(b)
}
`
)

var resFiles = make([]string, 0, 10)

/*
* walkFn 函数遍历每个静态文件，进行如下处理：
 1. 打开文件并压缩内容。
 2. 将文件路径转化为 Go 文件名的合法形式（替换文件路径中的分隔符等）。
 3. 为每个资源文件生成一个 Go 源文件，该文件包含解压缩资源的函数。
*/
func walkFn(resPath string, info os.FileInfo, _ error) error {
	if info.IsDir() || filterSuffix(resPath) {
		return nil
	}

	// Open resource files
	fr, err := os.Open(resPath)
	if err != nil {
		beeLogger.Log.Fatalf("Failed to read file: %s", err)
	}

	// Convert path
	resPath = strings.Replace(resPath, "_", "_0_", -1)
	resPath = strings.Replace(resPath, ".", "_1_", -1)
	resPath = strings.Replace(resPath, "-", "_2_", -1)
	resPath = strings.Replace(resPath, " ", "_3_", -1)
	sep := "/"
	if runtime.GOOS == "windows" {
		sep = "\\"
	}
	resPath = strings.Replace(resPath, sep, "_4_", -1)

	// Create corresponding Go source files
	os.MkdirAll(path.Dir(resPath), os.ModePerm)
	fw, err := os.Create("bale/" + resPath + ".go")
	if err != nil {
		beeLogger.Log.Fatalf("Failed to create file: %s", err)
	}
	defer fw.Close()

	// Write header
	fmt.Fprintf(fw, Header, resPath)

	// Copy and compress data
	gz := gzip.NewWriter(&ByteWriter{Writer: fw})
	io.Copy(gz, fr)
	gz.Close()

	// Write footer.
	fmt.Fprint(fw, Footer)

	resFiles = append(resFiles, resPath)
	return nil
}

// 用于过滤掉不需要打包的文件扩展名
func filterSuffix(name string) bool {
	for _, s := range config.Conf.Bale.IngExt {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

const (
	// Header ...
	Header = `package bale

import(
	"bytes"
	"compress/gzip"
	"io"
)

func R%s() []byte {
	gz, err := gzip.NewReader(bytes.NewBuffer([]byte{`
	// Footer ...
	Footer = `
	}))

	if err != nil {
		panic("Unpack resources failed: " + err.Error())
	}

	var b bytes.Buffer
	io.Copy(&b, gz)
	gz.Close()

	return b.Bytes()
}`
)

var newline = []byte{'\n'}

// ByteWriter ...
type ByteWriter struct {
	io.Writer
	c int
}

func (w *ByteWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	for n = range p {
		if w.c%12 == 0 {
			w.Writer.Write(newline)
			w.c = 0
		}
		fmt.Fprintf(w.Writer, "0x%02x,", p[n])
		w.c++
	}
	n++
	return
}
