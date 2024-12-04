package utils

import "fmt"

// flag.Var(&myList, "list", "List of options")
// go run main.go -list="option1" -list="option2" -list="option3"
// [option1 option2 option3]
type ListOpts []string

func (opts *ListOpts) String() string {
	return fmt.Sprint(*opts)
}

func (opts *ListOpts) Set(value string) error {
	*opts = append(*opts, value)
	return nil
}
