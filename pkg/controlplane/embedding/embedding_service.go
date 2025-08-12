package embedding

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/llm-aware-gateway/pkg/interfaces"
	"github.com/llm-aware-gateway/pkg/types"
	"github.com/llm-aware-gateway/pkg/utils"
)

// embeddingService 嵌入服务实现
type embeddingService struct {
	config    *types.EmbeddingConfig
	cache     interfaces.Cache
	model     *MockBGEModel // 使用模拟模型
	batchSize int
	mutex     sync.RWMutex
}

// MockBGEModel 模拟BGE模型
type MockBGEModel struct {
	dimension int
}

// NewEmbeddingService 创建嵌入服务
func NewEmbeddingService(config *types.EmbeddingConfig) interfaces.EmbeddingService {
	cache := utils.NewCache(config.CacheSize)

	model := &MockBGEModel{
		dimension: config.Dimension,
	}

	return &embeddingService{
		config:    config,
		cache:     cache,
		model:     model,
		batchSize: config.BatchSize,
	}
}

// EmbedText 文本向量化
func (es *embeddingService) EmbedText(text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	// 检查缓存
	cacheKey := fmt.Sprintf("embed:%s", text)
	if cached, found := es.cache.Get(cacheKey); found {
		if vector, ok := cached.([]float32); ok {
			return vector, nil
		}
	}

	// 预处理文本
	processedText := es.PreprocessText(text)

	// 生成向量
	vector, err := es.model.Encode(processedText)
	if err != nil {
		return nil, err
	}

	// 缓存结果
	es.cache.Set(cacheKey, vector, 300) // TTL 5分钟

	return vector, nil
}

// EmbedBatch 批量向量化
func (es *embeddingService) EmbedBatch(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	vectors := make([][]float32, len(texts))

	// 分批处理
	for i := 0; i < len(texts); i += es.batchSize {
		end := i + es.batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		batchVectors, err := es.processBatch(batch)
		if err != nil {
			return nil, err
		}

		copy(vectors[i:end], batchVectors)
	}

	return vectors, nil
}

// PreprocessText 预处理文本
func (es *embeddingService) PreprocessText(text string) string {
	if text == "" {
		return text
	}

	// 转换为小写
	text = strings.ToLower(text)

	// 模板化处理：将变量替换为占位符
	patterns := map[string]string{
		`\b\d{11}\b`:                                              "[PHONE]",
		`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`:    "[EMAIL]",
		`\b[A-Za-z0-9]{20,}\b`:                                    "[TOKEN]",
		`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`:                 "[IP]",
		`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`: "[UUID]",
		`\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`:                 "[CARD]",
		`\b\d+\b`:                                                 "[NUMBER]",
		`/[a-zA-Z0-9/._-]+`:                                       "[PATH]",
	}

	for pattern, replacement := range patterns {
		re := regexp.MustCompile(pattern)
		text = re.ReplaceAllString(text, replacement)
	}

	// 清理多余空格
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

// processBatch 处理批次
func (es *embeddingService) processBatch(texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))

	for i, text := range texts {
		vector, err := es.EmbedText(text)
		if err != nil {
			return nil, err
		}
		vectors[i] = vector
	}

	return vectors, nil
}

// Encode 模拟BGE模型编码
func (m *MockBGEModel) Encode(text string) ([]float32, error) {
	// 这是一个简化的模拟实现
	// 实际应用中应该使用真正的BGE模型

	vector := make([]float32, m.dimension)

	// 基于文本内容生成伪向量
	hash := 0
	for _, r := range text {
		hash = hash*31 + int(r)
	}

	for i := 0; i < m.dimension; i++ {
		// 生成[-1, 1]范围内的值
		value := float32((hash+i)%200-100) / 100.0
		vector[i] = value
		hash = hash*17 + i
	}

	// 归一化向量
	vector = utils.NormalizeVector(vector)

	log.Printf("Generated vector for text: %s (dim: %d)", utils.Truncate(text, 50), m.dimension)

	return vector, nil
}

// EncodeBatch 批量编码
func (m *MockBGEModel) EncodeBatch(texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))

	for i, text := range texts {
		vector, err := m.Encode(text)
		if err != nil {
			return nil, err
		}
		vectors[i] = vector
	}

	return vectors, nil
}
