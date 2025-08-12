package clustering

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/llm-aware-gateway/pkg/interfaces"
	"github.com/llm-aware-gateway/pkg/types"
	"github.com/llm-aware-gateway/pkg/utils"
)

// clusteringEngine 聚类引擎实现
type clusteringEngine struct {
	config            *types.ClusteringConfig
	embeddingService  interfaces.EmbeddingService
	vectorDB          interfaces.VectorDB
	clusters          map[string]*types.Cluster
	memberToCluster   map[string]string // 成员ID到簇ID的映射
	mutex             sync.RWMutex
	stopCh            chan struct{}
	reclusterTicker   *time.Ticker
}

// NewClusteringEngine 创建聚类引擎
func NewClusteringEngine(
	config *types.ClusteringConfig,
	embeddingService interfaces.EmbeddingService,
	vectorDB interfaces.VectorDB,
) interfaces.ClusteringEngine {
	return &clusteringEngine{
		config:           config,
		embeddingService: embeddingService,
		vectorDB:         vectorDB,
		clusters:         make(map[string]*types.Cluster),
		memberToCluster:  make(map[string]string),
		stopCh:           make(chan struct{}),
	}
}

// ProcessErrorEvent 处理错误事件
func (ce *clusteringEngine) ProcessErrorEvent(event *types.ErrorEvent) error {
	// 构建错误特征文本
	errorText := ce.buildErrorSignature(event)

	// 生成向量
	vector, err := ce.embeddingService.EmbedText(errorText)
	if err != nil {
		return fmt.Errorf("failed to embed text: %v", err)
	}

	// 查找最相似的簇
	clusterID, similarity, err := ce.FindMostSimilarCluster(vector)
	if err != nil {
		return fmt.Errorf("failed to find similar cluster: %v", err)
	}

	// 判断是否创建新簇或加入现有簇
	if clusterID == "" || similarity < ce.config.SimilarityThreshold {
		// 创建新簇
		newClusterID, err := ce.CreateNewCluster(event, vector)
		if err != nil {
			return fmt.Errorf("failed to create new cluster: %v", err)
		}
		event.ClusterID = newClusterID
		log.Printf("Created new cluster %s for error event %s", newClusterID, event.EventID)
	} else {
		// 加入现有簇
		err := ce.addEventToCluster(clusterID, event, vector)
		if err != nil {
			return fmt.Errorf("failed to add event to cluster: %v", err)
		}
		event.ClusterID = clusterID
		log.Printf("Added event %s to existing cluster %s (similarity: %.4f)", event.EventID, clusterID, similarity)
	}

	return nil
}

// FindMostSimilarCluster 查找最相似的簇
func (ce *clusteringEngine) FindMostSimilarCluster(vector []float32) (string, float64, error) {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()

	var bestClusterID string
	var bestSimilarity float64

	for clusterID, cluster := range ce.clusters {
		if len(cluster.Centroid) == 0 {
			continue
		}

		similarity := utils.CosineSimilarity(vector, cluster.Centroid)
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestClusterID = clusterID
		}
	}

	return bestClusterID, bestSimilarity, nil
}

// CreateNewCluster 创建新簇
func (ce *clusteringEngine) CreateNewCluster(event *types.ErrorEvent, vector []float32) (string, error) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	// 检查簇数量限制
	if len(ce.clusters) >= ce.config.MaxClusters {
		return "", fmt.Errorf("maximum number of clusters (%d) reached", ce.config.MaxClusters)
	}

	clusterID := utils.GenerateClusterID()

	cluster := &types.Cluster{
		ID:          clusterID,
		Centroid:    make([]float32, len(vector)),
		Members:     []string{event.EventID},
		ErrorCount:  1,
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
		Severity:    0.0, // 初始严重度为0
		Description: ce.generateClusterDescription(event),
	}

	copy(cluster.Centroid, vector)

	// 存储簇信息
	ce.clusters[clusterID] = cluster
	ce.memberToCluster[event.EventID] = clusterID

	// 将向量存储到向量数据库
	if err := ce.vectorDB.AddVector(event.EventID, vector); err != nil {
		log.Printf("Failed to store vector in database: %v", err)
	}

	return clusterID, nil
}

// GetCluster 获取簇信息
func (ce *clusteringEngine) GetCluster(clusterID string) (*types.Cluster, error) {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()

	cluster, exists := ce.clusters[clusterID]
	if !exists {
		return nil, fmt.Errorf("cluster not found: %s", clusterID)
	}

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

	return clusterCopy, nil
}

// GetAllClusters 获取所有簇
func (ce *clusteringEngine) GetAllClusters() (map[string]*types.Cluster, error) {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()

	clusters := make(map[string]*types.Cluster)

	for clusterID, cluster := range ce.clusters {
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

		clusters[clusterID] = clusterCopy
	}

	return clusters, nil
}

// ReCluster 重新聚类
func (ce *clusteringEngine) ReCluster() error {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	log.Println("Starting re-clustering process...")

	// 收集所有向量
	var vectors [][]float32
	var eventIDs []string

	for _, cluster := range ce.clusters {
		for _, memberID := range cluster.Members {
			if vector, err := ce.vectorDB.GetVector(memberID); err == nil {
				vectors = append(vectors, vector)
				eventIDs = append(eventIDs, memberID)
			}
		}
	}

	if len(vectors) < ce.config.MinClusterSize {
		log.Printf("Not enough vectors (%d) for re-clustering, minimum required: %d", len(vectors), ce.config.MinClusterSize)
		return nil
	}

	// 使用K-means算法重新聚类
	newClusters := ce.kMeansCluster(vectors, eventIDs, len(ce.clusters))

	// 更新簇信息
	ce.clusters = newClusters
	ce.memberToCluster = make(map[string]string)

	for clusterID, cluster := range ce.clusters {
		for _, memberID := range cluster.Members {
			ce.memberToCluster[memberID] = clusterID
		}
	}

	log.Printf("Re-clustering completed: %d clusters", len(ce.clusters))
	return nil
}

// Start 启动聚类引擎
func (ce *clusteringEngine) Start() error {
	// 启动定期重聚类
	ce.reclusterTicker = time.NewTicker(ce.config.ReclusteringInterval)

	go func() {
		for {
			select {
			case <-ce.reclusterTicker.C:
				if err := ce.ReCluster(); err != nil {
					log.Printf("Re-clustering failed: %v", err)
				}
			case <-ce.stopCh:
				return
			}
		}
	}()

	log.Println("Clustering engine started")
	return nil
}

// Stop 停止聚类引擎
func (ce *clusteringEngine) Stop() error {
	close(ce.stopCh)

	if ce.reclusterTicker != nil {
		ce.reclusterTicker.Stop()
	}

	log.Println("Clustering engine stopped")
	return nil
}

// addEventToCluster 将事件添加到簇
func (ce *clusteringEngine) addEventToCluster(clusterID string, event *types.ErrorEvent, vector []float32) error {
	cluster, exists := ce.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster not found: %s", clusterID)
	}

	// 添加成员
	cluster.Members = append(cluster.Members, event.EventID)
	cluster.ErrorCount++
	cluster.UpdateTime = time.Now()

	// 更新质心
	ce.updateCentroid(cluster, vector)

	// 更新映射
	ce.memberToCluster[event.EventID] = clusterID

	// 存储向量
	if err := ce.vectorDB.AddVector(event.EventID, vector); err != nil {
		log.Printf("Failed to store vector in database: %v", err)
	}

	return nil
}

// updateCentroid 更新簇质心
func (ce *clusteringEngine) updateCentroid(cluster *types.Cluster, newVector []float32) {
	if len(cluster.Centroid) != len(newVector) {
		return
	}

	// 增量更新质心
	n := float32(len(cluster.Members))
	for i := range cluster.Centroid {
		cluster.Centroid[i] = (cluster.Centroid[i]*(n-1) + newVector[i]) / n
	}
}

// buildErrorSignature 构建错误特征
func (ce *clusteringEngine) buildErrorSignature(event *types.ErrorEvent) string {
	signature := fmt.Sprintf("service:%s method:%s path:%s error:%s",
		event.ServiceName,
		event.Method,
		event.RequestPath,
		event.ErrorMessage,
	)

	// 添加堆栈信息前两帧
	if len(event.StackTrace) > 0 {
		signature += " stack:" + event.StackTrace[0]
		if len(event.StackTrace) > 1 {
			signature += " " + event.StackTrace[1]
		}
	}

	return signature
}

// generateClusterDescription 生成簇描述
func (ce *clusteringEngine) generateClusterDescription(event *types.ErrorEvent) string {
	return fmt.Sprintf("Service: %s, Method: %s, Error: %s",
		event.ServiceName,
		event.Method,
		utils.Truncate(event.ErrorMessage, 100),
	)
}

// kMeansCluster K-means聚类算法
func (ce *clusteringEngine) kMeansCluster(vectors [][]float32, eventIDs []string, k int) map[string]*types.Cluster {
	if k <= 0 || len(vectors) == 0 {
		return make(map[string]*types.Cluster)
	}

	// 简化的K-means实现
	// 初始化质心
	centroids := make([][]float32, k)
	for i := 0; i < k; i++ {
		centroids[i] = make([]float32, len(vectors[0]))
		copy(centroids[i], vectors[i%len(vectors)])
	}

	// 迭代优化
	maxIterations := 10
	for iter := 0; iter < maxIterations; iter++ {
		// 分配点到最近的质心
		assignments := make([]int, len(vectors))
		for i, vector := range vectors {
			bestCluster := 0
			bestDistance := utils.EuclideanDistance(vector, centroids[0])

			for j := 1; j < k; j++ {
				distance := utils.EuclideanDistance(vector, centroids[j])
				if distance < bestDistance {
					bestDistance = distance
					bestCluster = j
				}
			}

			assignments[i] = bestCluster
		}

		// 更新质心
		newCentroids := make([][]float32, k)
		counts := make([]int, k)

		for i := 0; i < k; i++ {
			newCentroids[i] = make([]float32, len(vectors[0]))
		}

		for i, vector := range vectors {
			clusterIdx := assignments[i]
			counts[clusterIdx]++
			for j := range vector {
				newCentroids[clusterIdx][j] += vector[j]
			}
		}

		for i := 0; i < k; i++ {
			if counts[i] > 0 {
				for j := range newCentroids[i] {
					newCentroids[i][j] /= float32(counts[i])
				}
				centroids[i] = newCentroids[i]
			}
		}
	}

	// 构建簇
	clusters := make(map[string]*types.Cluster)
	for i := 0; i < k; i++ {
		clusterID := utils.GenerateClusterID()
		cluster := &types.Cluster{
			ID:         clusterID,
			Centroid:   centroids[i],
			Members:    []string{},
			ErrorCount: 0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
			Severity:   0.0,
		}

		// 添加属于这个簇的成员
		for j, vector := range vectors {
			bestCluster := 0
			bestDistance := utils.EuclideanDistance(vector, centroids[0])

			for l := 1; l < k; l++ {
				distance := utils.EuclideanDistance(vector, centroids[l])
				if distance < bestDistance {
					bestDistance = distance
					bestCluster = l
				}
			}

			if bestCluster == i {
				cluster.Members = append(cluster.Members, eventIDs[j])
				cluster.ErrorCount++
			}
		}

		if len(cluster.Members) > 0 {
			clusters[clusterID] = cluster
		}
	}

	return clusters
}
