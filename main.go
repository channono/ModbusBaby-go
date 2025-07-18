// 大牛大巨婴 - ModbusBaby Go版本 (完美布局)
// Big Giant Baby - ModbusBaby Go Edition (Perfect Layout)
package main

import (
	"log"
	"modbusbaby/internal/config"
	"modbusbaby/internal/gui"
	"modbusbaby/internal/logger"
)

var (
	version = "2.0.0"
	author  = "Daniel BigGiantBaby (大牛大巨婴)"
)

func main() {
	// 初始化日志系统
	logger.Init()

	log.Printf("ModbusBaby v%s - by %s", version, author)
	log.Println("Starting ModbusBaby Go Edition (Perfect Layout)...")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Printf("配置加载失败，使用默认配置: %v", err)
		cfg = config.Default()
	}

	// 创建并运行完美布局GUI应用
	app := gui.NewAppRefined(cfg, version, author)
	app.ShowAndRun()
}