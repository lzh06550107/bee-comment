package utils

import "fmt"

type DocValue string

func (d *DocValue) String() string {
	return fmt.Sprint(*d)
}

func (d *DocValue) Set(value string) error {
	// 将 value 转换为 DocValue 类型，并赋值给 *d
	*d = DocValue(value)
	return nil
}
