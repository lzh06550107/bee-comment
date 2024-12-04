package utils

import (
	"errors"
	"net/url"
	"strings"
)

// 这段代码是一个用于解析数据源名称（DSN）的工具，它能够将 DSN 字符串解析成结构体 DSN，并且支持从 DSN 中提取出用户名、密码、网络类型、地址、数据库名称以及额外的连接参数。

// DSN ... DSN 结构体用于保存 DSN 中的各个组件
type DSN struct {
	User   string            // Username
	Passwd string            // Password (requires User)
	Net    string            // Network type
	Addr   string            // Network address (requires Net)
	DBName string            // Database name
	Params map[string]string // Connection parameters
}

// 这三个错误定义用于在解析过程中返回错误
var (
	// 如果 URL 中的参数值没有正确转义
	errInvalidDSNUnescaped = errors.New("invalid DSN: did you forget to escape a param value?")
	// 如果网络地址没有正确关闭括号
	errInvalidDSNAddr = errors.New("invalid DSN: network address not terminated (missing closing brace)")
	// 如果 DSN 中缺少 /，即数据库名称部分前没有分隔符
	errInvalidDSNNoSlash = errors.New("invalid DSN: missing the slash separating the database name")
)

// ParseDSN parses the DSN string to a Config，负责将 DSN 字符串解析成 DSN 结构体
func ParseDSN(dsn string) (cfg *DSN, err error) {
	// New config with some default values
	cfg = new(DSN)

	// [user[:password]@][net[(addr)]]/dbname[?param1=value1&paramN=valueN]
	// Find the last '/' (since the password or the net addr might contain a '/')
	foundSlash := false
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '/' {
			foundSlash = true
			var j, k int

			// left part is empty if i <= 0
			if i > 0 {
				// [username[:password]@][protocol[(address)]]
				// Find the last '@' in dsn[:i]
				for j = i; j >= 0; j-- {
					if dsn[j] == '@' {
						// username[:password]
						// Find the first ':' in dsn[:j]
						for k = 0; k < j; k++ {
							if dsn[k] == ':' {
								cfg.Passwd = dsn[k+1 : j]
								break
							}
						}
						cfg.User = dsn[:k]

						break
					}
				}

				// [protocol[(address)]]
				// Find the first '(' in dsn[j+1:i]
				for k = j + 1; k < i; k++ {
					if dsn[k] == '(' {
						// dsn[i-1] must be == ')' if an address is specified
						if dsn[i-1] != ')' {
							if strings.ContainsRune(dsn[k+1:i], ')') {
								return nil, errInvalidDSNUnescaped
							}
							return nil, errInvalidDSNAddr
						}
						cfg.Addr = dsn[k+1 : i-1]
						break
					}
				}
				cfg.Net = dsn[j+1 : k]
			}

			// dbname[?param1=value1&...&paramN=valueN]
			// Find the first '?' in dsn[i+1:]
			for j = i + 1; j < len(dsn); j++ {
				if dsn[j] == '?' {
					if err = parseDSNParams(cfg, dsn[j+1:]); err != nil {
						return
					}
					break
				}
			}
			cfg.DBName = dsn[i+1 : j]

			break
		}
	}
	if !foundSlash && len(dsn) > 0 {
		return nil, errInvalidDSNNoSlash
	}
	return
}

// 该函数用于解析 DSN 中的查询参数部分（例如 ?param1=value1&param2=value2），并将其存储在 DSN 结构体的 Params 字段中
func parseDSNParams(cfg *DSN, params string) (err error) {
	for _, v := range strings.Split(params, "&") {
		param := strings.SplitN(v, "=", 2)
		if len(param) != 2 {
			continue
		}
		// lazy init
		if cfg.Params == nil {
			cfg.Params = make(map[string]string)
		}
		value := param[1]
		if cfg.Params[param[0]], err = url.QueryUnescape(value); err != nil {
			return
		}
	}
	return
}
