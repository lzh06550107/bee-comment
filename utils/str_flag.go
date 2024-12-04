package utils

import "fmt"

// The string flag list, implemented flag.Value interface
// 该实现非常适用于需要通过命令行接受多个值的标志。例如，你可能需要处理像 -file 或 -list 这样的标志，可以通过 StrFlags 类型来接收多个文件名或多个选项
// go run main.go -file="file1.txt" -file="file2.txt" -file="file3.txt"
type StrFlags []string

func (s *StrFlags) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *StrFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}
