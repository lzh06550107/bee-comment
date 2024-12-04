package utils

import (
	"os"

	beeLogger "github.com/beego/bee/v2/logger"
)

// 这段代码提供了几个常用的文件系统操作函数，主要是检查文件或目录是否存在，并创建目录。

// Mkdir ... 该函数用于创建指定路径的目录。如果目录路径为空，则记录日志并返回 false；如果目录创建失败，也会记录错误日志并返回 false。如果创建成功，则记录成功日志并返回 true
func Mkdir(dir string) bool {
	if dir == "" {
		beeLogger.Log.Fatalf("The directory is empty")
		return false
	}
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		beeLogger.Log.Fatalf("Could not create the directory: %s", err)
		return false
	}

	beeLogger.Log.Infof("Create %s Success!", dir)
	return true
}

// IsDir ... 检查指定路径是否为目录。如果路径不存在或不是目录，则返回 false。如果是目录，则返回 true
func IsDir(dir string) bool {
	f, e := os.Stat(dir)
	if e != nil {
		return false
	}
	return f.IsDir()
}

// IsExist returns whether a file or directory exists.
// 检查指定路径是否存在（无论是文件还是目录）。如果路径存在，则返回 true；如果路径不存在，返回 false
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
