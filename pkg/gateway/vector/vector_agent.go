package vector

import (
	"fmt"
	"log"
	"sync"

	"github.com/llm-aware-gateway/pkg/interfaces"
	"github.com/llm-aware-gateway/pkg/types"
	"github.com/llm-aware-gateway/pkg/utils"
)

// vectorAgent 向量代理实现
type vectorAgent struct {
	embeddingService interfaces.EmbeddingService
	clusters         map[string]*types.Cluster
	cache            interfaces.Cache
	similarityThreshold float64
	mutex            sync.RWMutex
}

// NewVectorAgent 创建向量代理
func NewVectorAgent(embeddingService interfaces.EmbeddingService, cache interfaces.Cache) interfaces.VectorAgent {
	return &vectorAgent{
		embeddingService:    embeddingService,
		clusters:           make(map[string]*types.Cluster),
		cache:              cache,
		similarityThreshold: 0.82, // 默认相似度阈值
	}
}

// IdentifyCluster 识别错误所属的簇
func (va *vectorAgent) IdentifyCluster(errorSignature string) (string, error) {
	if errorSignature == "" {
		return "", nil
	}

	// 首先检查缓存
	if cached, found := va.cache.Get(errorSignature); found {
		if clusterID, ok := cached.(string); ok {
			return clusterID, nil
		}
	}

	// 生成错误签名的向量
	vector, err := va.GenerateVector(errorSignature)
	if err != nil {
		return "", fmt.Errorf("failed to generate vector: %v", err)
	}

	// 查找最相似的簇
	clusterID := va.findMostSimilarCluster(vector)

	// 缓存结果（TTL 5分钟）
	if clusterID != "" {
		va.cache.Set(errorSignature, clusterID, 300)
	}

	return clusterID, nil
}

// GenerateVector 生成文本向量
func (va *vectorAgent) GenerateVector(text string) ([]float32, error) {
	if va.embeddingService == nil {
		return nil, fmt.Errorf("embedding service not available")
	}

	// 预处理文本
	processedText := va.embeddingService.PreprocessText(text)

	// 生成向量
	vector, err := va.embeddingService.EmbedText(processedText)
	if err != nil {
		return nil, err
	}

	return vector, nil
}

// UpdateClusters 更新簇信息
func (va *vectorAgent) UpdateClusters(clusters map[string]*types.Cluster) error {
	va.mutex.Lock()
	defer va.mutex.Unlock()

	// 更新簇信息
	va.clusters = make(map[string]*types.Cluster)
	for clusterID, cluster := range clusters {
		// 深拷贝簇信息
		clusterCopy := &types.Cluster{
			ID:          cluster.ID,
			Centroid:    make([]float32, len(cluster.Centroid)),
			Members:     make([]string, len(cluster.Members)),
			ErrorCount:  cluster.ErrorCount,
			CreateTime:  cluster.CreateTime,
			UpdateTime:  cluster.UpdateTime,
			Severity:    cluster.Severity,
			Description: cluster.Description,
		}

		copy(clusterCopy.Centroid, cluster.Centroid)
		copy(clusterCopy.Members, cluster.Members)

		va.clusters[clusterID] = clusterCopy
	}

	// 清空缓存，强制重新计算
	va.cache.Clear()

	log.Printf("Updated %d clusters in vector agent", len(clusters))
	return nil
}

// findMostSimilarCluster 查找最相似的簇
func (va *vectorAgent) findMostSimilarCluster(vector []float32) string {
	va.mutex.RLock()
	defer va.mutex.RUnlock()

	var bestClusterID string
	var bestSimilarity float64

	for clusterID, cluster := range va.clusters {
		if len(cluster.Centroid) == 0 {
			continue
		}

		similarity := utils.CosineSimilarity(vector, cluster.Centroid)
		if similarity > bestSimilarity && similarity >= va.similarityThreshold {
			bestSimilarity = similarity
			bestClusterID = clusterID
		}
	}

	if bestClusterID != "" {
		log.Printf("Found similar cluster: %s (similarity: %.4f)", bestClusterID, bestSimilarity)
	}

	return bestClusterID
}

// getClusterCount 获取簇数量
func (va *vectorAgent) getClusterCount() int {
	va.mutex.RLock()
	defer va.mutex.RUnlock()
	return len(va.clusters)
}

// setSimilarityThreshold 设置相似度阈值
func (va *vectorAgent) setSimilarityThreshold(threshold float64) {
	va.mutex.Lock()
	defer va.mutex.Unlock()
	va.similarityThreshold = threshold
}

// getSimilarityThreshold 获取相似度阈值
func (va *vectorAgent) getSimilarityThreshold() float64 {
	va.mutex.RLock()
	defer va.mutex.RUnlock()
	return va.similarityThreshold
}
