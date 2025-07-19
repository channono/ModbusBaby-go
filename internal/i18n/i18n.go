package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localeFiles embed.FS

// I18n 国际化管理器
type I18n struct {
	currentLang string
	messages    map[string]string
	fallback    map[string]string
}

// NewI18n 创建国际化管理器
func NewI18n() *I18n {
	i18n := &I18n{
		messages: make(map[string]string),
		fallback: make(map[string]string),
	}
	
	// 检测系统语言
	systemLang := detectSystemLanguage()
	i18n.SetLanguage(systemLang)
	
	return i18n
}

// SetLanguage 设置语言
func (i *I18n) SetLanguage(lang string) error {
	// 标准化语言代码
	lang = normalizeLanguageCode(lang)
	
	// 加载回退语言（英文）
	if err := i.loadLanguageFile("en", &i.fallback); err != nil {
		log.Printf("警告: 无法加载回退语言文件: %v", err)
	}
	
	// 加载目标语言
	if err := i.loadLanguageFile(lang, &i.messages); err != nil {
		log.Printf("警告: 无法加载语言文件 %s: %v", lang, err)
		// 如果目标语言加载失败，使用英文
		lang = "en"
		i.messages = make(map[string]string)
		for k, v := range i.fallback {
			i.messages[k] = v
		}
	}
	
	i.currentLang = lang
	log.Printf("语言设置为: %s", lang)
	return nil
}

// T 翻译文本
func (i *I18n) T(key string, args ...interface{}) string {
	// 首先尝试当前语言
	if text, exists := i.messages[key]; exists {
		if len(args) > 0 {
			return fmt.Sprintf(text, args...)
		}
		return text
	}
	
	// 回退到英文
	if text, exists := i.fallback[key]; exists {
		if len(args) > 0 {
			return fmt.Sprintf(text, args...)
		}
		return text
	}
	
	// 如果都没有找到，返回键名
	log.Printf("警告: 未找到翻译键: %s", key)
	return key
}

// GetCurrentLanguage 获取当前语言
func (i *I18n) GetCurrentLanguage() string {
	return i.currentLang
}

// GetAvailableLanguages 获取可用语言列表
func (i *I18n) GetAvailableLanguages() []string {
	return []string{"en", "zh-CN", "zh-TW", "ja"}
}

// loadLanguageFile 加载语言文件
func (i *I18n) loadLanguageFile(lang string, target *map[string]string) error {
	filename := fmt.Sprintf("locales/%s.json", lang)
	
	data, err := localeFiles.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取语言文件失败: %v", err)
	}
	
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("解析语言文件失败: %v", err)
	}
	
	*target = messages
	return nil
}

// detectSystemLanguage 检测系统语言
func detectSystemLanguage() string {
	// 尝试从环境变量获取语言设置
	if lang := os.Getenv("LANG"); lang != "" {
		return normalizeLanguageCode(lang)
	}
	
	if lang := os.Getenv("LC_ALL"); lang != "" {
		return normalizeLanguageCode(lang)
	}
	
	if lang := os.Getenv("LC_MESSAGES"); lang != "" {
		return normalizeLanguageCode(lang)
	}
	
	// 使用golang.org/x/text/language检测
	tags, _, _ := language.ParseAcceptLanguage(os.Getenv("ACCEPT_LANGUAGE"))
	if len(tags) > 0 {
		return normalizeLanguageCode(tags[0].String())
	}
	
	// 默认返回英文
	return "en"
}

// normalizeLanguageCode 标准化语言代码
func normalizeLanguageCode(lang string) string {
	// 移除编码信息 (如 zh_CN.UTF-8 -> zh_CN)
	if idx := strings.Index(lang, "."); idx != -1 {
		lang = lang[:idx]
	}
	
	// 转换下划线为连字符
	lang = strings.ReplaceAll(lang, "_", "-")
	
	// 转换为小写
	lang = strings.ToLower(lang)
	
	// 处理特殊情况
	switch {
	case strings.HasPrefix(lang, "zh-cn") || strings.HasPrefix(lang, "zh-hans"):
		return "zh-CN"
	case strings.HasPrefix(lang, "zh-tw") || strings.HasPrefix(lang, "zh-hant"):
		return "zh-TW"
	case strings.HasPrefix(lang, "zh"):
		return "zh"
	case strings.HasPrefix(lang, "en"):
		return "en"
	case strings.HasPrefix(lang, "ja"):
		return "ja"
	case strings.HasPrefix(lang, "ko"):
		return "ko"
	default:
		return "en" // 默认英文
	}
}

// 全局实例
var globalI18n *I18n

// Init 初始化全局国际化实例
func Init() {
	globalI18n = NewI18n()
}

// T 全局翻译函数
func T(key string, args ...interface{}) string {
	if globalI18n == nil {
		Init()
	}
	return globalI18n.T(key, args...)
}

// SetLanguage 设置全局语言
func SetLanguage(lang string) error {
	if globalI18n == nil {
		Init()
	}
	return globalI18n.SetLanguage(lang)
}

// GetCurrentLanguage 获取当前语言
func GetCurrentLanguage() string {
	if globalI18n == nil {
		Init()
	}
	return globalI18n.GetCurrentLanguage()
}

// GetAvailableLanguages 获取可用语言
func GetAvailableLanguages() []string {
	if globalI18n == nil {
		Init()
	}
	return globalI18n.GetAvailableLanguages()
}
