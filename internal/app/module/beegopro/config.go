package beegopro

import (
	"io/ioutil"

	"github.com/beego/bee/v2/internal/pkg/utils"
	beeLogger "github.com/beego/bee/v2/logger"
)

// 这段代码的作用是生成一个 beegopro.toml 配置文件。它检查文件是否已经存在，如果存在则输出日志并停止执行；如果文件不存在，则创建一个新的配置文件并写入默认内容

// 这是一个字符串数组，列出了在某些比较操作中需要排除的字段。例如，可能在生成配置或版本比较时，@BeeGenerateTime 这个字段应该被排除在外
var CompareExcept = []string{"@BeeGenerateTime"}

// 这个方法用于生成配置文件
func (c *Container) GenConfig() {
	// 检查 c.BeegoProFile 所指定的文件是否存在。如果文件已存在，日志记录 "beego pro toml exist" 并返回，停止进一步的操作。
	if utils.IsExist(c.BeegoProFile) {
		beeLogger.Log.Fatalf("beego pro toml exist")
		return
	}

	// 如果文件不存在，使用 ioutil.WriteFile 创建一个新的文件，并写入 BeegoToml 字符串的内容。
	// 文件权限设置为 0644，意味着所有用户都可以读取文件，但只有文件的所有者可以写入。
	err := ioutil.WriteFile("beegopro.toml", []byte(BeegoToml), 0644)
	if err != nil {
		beeLogger.Log.Fatalf("write beego pro toml err: %s", err)
		return
	}
}
