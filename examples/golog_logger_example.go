package main

import (
	"os"

	"github.com/kataras/golog"
	"github.com/smallnest/langgraphgo/log"
)

func main() {
	// 示例 1: 使用默认的 golog logger
	defaultLogger := golog.Default
	logger1 := log.NewGologLogger(defaultLogger)
	logger1.Info("使用默认 golog logger")
	logger1.SetLevel(log.LogLevelDebug)
	logger1.Debug("调试信息")

	// 示例 2: 创建自定义的 golog logger
	customLogger := golog.New()
	customLogger.SetPrefix("[ MyApp ] ")
	customLogger.SetOutput(os.Stdout)
	logger2 := log.NewGologLogger(customLogger)
	logger2.SetLevel(log.LogLevelInfo)
	logger2.Info("使用自定义 golog logger")

	// 示例 3: 使用不同的 golog 配置
	errorLogger := golog.New()
	errorLogger.SetLevel("error")
	errorLogger.SetPrefix("[ ERROR ] ")
	logger3 := log.NewGologLogger(errorLogger)
	logger3.Debug("这条不会显示")
	logger3.Error("错误信息会显示")
}
