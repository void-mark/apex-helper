package helper

import "fmt"

type Notifier interface {
	success(msg string)
	bad(msg string)
	normal(msg string)
}

type colorPrint struct {
	successColor string
	badColor     string
	reset        string
}

func (c *colorPrint) success(msg string) {
	fmt.Println(c.successColor, msg, c.reset)
}

func (c *colorPrint) bad(msg string) {
	fmt.Println(c.badColor, msg, c.reset)
}

func (c *colorPrint) normal(msg string) {
	fmt.Println(msg)
}

func NewColorNotifier() Notifier {
	return &colorPrint{
		successColor: "\033[32m",
		badColor:     "\033[31m",
		reset:        "\033[0m",
	}
}
