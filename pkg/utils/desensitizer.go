package utils

import (
	"regexp"
	"sync"

	"github.com/llm-aware-gateway/pkg/interfaces"
)

// desensitizer 脱敏器实现
type desensitizer struct {
	patterns map[string]*patternInfo
	mutex    sync.RWMutex
}

// patternInfo 模式信息
type patternInfo struct {
	regex       *regexp.Regexp
	replacement string
}

// NewDesensitizer 创建脱敏器
func NewDesensitizer() interfaces.Desensitizer {
	d := &desensitizer{
		patterns: make(map[string]*patternInfo),
	}

	// 添加默认脱敏规则
	d.AddPattern("phone", `\b\d{11}\b`, "[PHONE]")
	d.AddPattern("email", `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, "[EMAIL]")
	d.AddPattern("token", `\b[A-Za-z0-9]{20,}\b`, "[TOKEN]")
	d.AddPattern("ip", `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`, "[IP]")
	d.AddPattern("uuid", `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`, "[UUID]")
	d.AddPattern("creditcard", `\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`, "[CARD]")

	return d
}

// Desensitize 脱敏文本
func (d *desensitizer) Desensitize(text string) string {
	if text == "" {
		return text
	}

	d.mutex.RLock()
	defer d.mutex.RUnlock()

	result := text
	for _, pattern := range d.patterns {
		result = pattern.regex.ReplaceAllString(result, pattern.replacement)
	}

	return result
}

// AddPattern 添加脱敏规则
func (d *desensitizer) AddPattern(name string, pattern string, replacement string) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return // 忽略无效的正则表达式
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.patterns[name] = &patternInfo{
		regex:       regex,
		replacement: replacement,
	}
}