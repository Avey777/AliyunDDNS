package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

type Brush func(string) string

func NewBrush(color string) Brush {
	pre := "\033["
	reset := "\033[0m"
	return func(text string) string {
		return pre + color + "m" + text + reset
	}
}

//鉴于终端的通常使用习惯，一般白色和黑色字体是不可行的,所以30,37不可用，
var colors = []Brush{
	NewBrush("1;41"), // Emergency          红色底
	NewBrush("1;35"), // Alert              紫色
	NewBrush("1;34"), // Critical           蓝色
	NewBrush("1;31"), // Error              红色
	NewBrush("1;33"), // Warn               黄色
	NewBrush("1;36"), // Informational      天蓝色
	NewBrush("1;32"), // Debug              绿色
	NewBrush("1;32"), // Trace              绿色
}

type ConsoleLogger struct {
	sync.Mutex
	Level    string `json:"level"`
	Colorful bool   `json:"color"`
	LogLevel int
}

func (c *ConsoleLogger) Init(jsonConfig string) error {
	if len(jsonConfig) == 0 {
		return nil
	}
	if jsonConfig != "{}" {
		fmt.Fprintf(os.Stdout, "ConsoleLogger Init:%s\n", jsonConfig)
	}

	err := json.Unmarshal([]byte(jsonConfig), c)
	if runtime.GOOS == "windows" {
		c.Colorful = false
	}

	if l, ok := LevelMap[c.Level]; ok {
		c.LogLevel = l
		return nil
	}

	return err
}

func (c *ConsoleLogger) LogWrite(when time.Time, msgText interface{}, level int) error {
	if level > c.LogLevel {
		return nil
	}
	msg, ok := msgText.(string)
	if !ok {
		return nil
	}
	if c.Colorful {
		msg = colors[level](msg)
	}
	c.PrintlnConsole(when, msg)
	return nil
}

func (c *ConsoleLogger) Destroy() {

}

func (c *ConsoleLogger) PrintlnConsole(when time.Time, msg string) {
	c.Lock()
	defer c.Unlock()
	os.Stdout.Write(append([]byte(msg), '\n'))
}

func init() {
	Register(AdapterConsole, &ConsoleLogger{
		LogLevel: LevelDebug,
		Colorful: runtime.GOOS != "windows",
	})
}
