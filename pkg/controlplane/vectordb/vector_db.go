package vectordb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	_ "github.com/lib/pq"

	"github.com/llm-aware-gateway/pkg/interfaces"
	"github.com/llm-aware-gateway/pkg/types"
	"github.com/llm-aware-gateway/pkg/utils"
)

// vectorDB 向量数据库实现
type vectorDB struct {
	config      *types.VectorDBConfig
	pgConn      *sql.DB
	cache       interfaces.Cache
	vectors     map[string][]float32 // 内存索引
	mutex       sync.RWMutex
}

// NewVectorDB 创建向量数据库
func NewVectorDB(config *types.VectorDBConfig) (interfaces.VectorDB, error) {
	// 连接PostgreSQL
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.PostgreSQL.Host,
		config.PostgreSQL.Port,
		config.PostgreSQL.Username,
		config.PostgreSQL.Password,
		config.PostgreSQL.Database,
		config.PostgreSQL.SSLMode,
	)

	pgConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	// 测试连接
	if err := pgConn.Ping(); err != nil {
		log.Printf("Warning: PostgreSQL connection failed, using memory-only mode: %v", err)
		pgConn = nil
	}

	// 创建缓存
	cache := utils.NewCache(config.CacheSize)

	vdb := &vectorDB{
		config:  config,
		pgConn:  pgConn,
		cache:   cache,
		vectors: make(map[string][]float32),
	}

	// 初始化数据库表
	if pgConn != nil {
		if err := vdb.initTables(); err != nil {
			log.Printf("Warning: Failed to init database tables: %v", err)
		}
	}

	return vdb, nil
}

// AddVector 添加向量
func (vdb *vectorDB) AddVector(id string, vector []float32) error {
	vdb.mutex.Lock()
	defer vdb.mutex.Unlock()

	// 添加到内存索引
	vectorCopy := make([]float32, len(vector))
	copy(vectorCopy, vector)
	vdb.vectors[id] = vectorCopy

	// 缓存向量
	vdb.cache.Set(fmt.Sprintf("vector:%s", id), vectorCopy, 3600) // TTL 1小时

	// 持久化到PostgreSQL
	if vdb.pgConn != nil {
		vectorJSON, _ := json.Marshal(vector)
		_, err := vdb.pgConn.Exec(`
			INSERT INTO vectors (id, vector_data, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
			ON CONFLICT (id) DO UPDATE SET
				vector_data = $2, updated_at = NOW()
		`, id, string(vectorJSON))

		if err != nil {
			log.Printf("Failed to persist vector to database: %v", err)
		}
	}

	log.Printf("Added vector: %s (dim: %d)", id, len(vector))
	return nil
}

// SearchSimilar 搜索相似向量
func (vdb *vectorDB) SearchSimilar(query []float32, topK int) ([]types.SearchResult, error) {
	vdb.mutex.RLock()
	defer vdb.mutex.RUnlock()

	// 计算所有向量的相似度
	similarities := make([]types.SearchResult, 0, len(vdb.vectors))

	for id, vector := range vdb.vectors {
		similarity := utils.CosineSimilarity(query, vector)
		similarities = append(similarities, types.SearchResult{
			ID:         id,
			Similarity: similarity,
			Vector:     vector,
		})
	}

	// 按相似度排序
	vdb.sortBySimilarity(similarities)

	// 返回前topK个结果
	if topK > len(similarities) {
		topK = len(similarities)
	}

	results := similarities[:topK]
	log.Printf("Found %d similar vectors for query (dim: %d)", len(results), len(query))

	return results, nil
}

// GetVector 获取向量
func (vdb *vectorDB) GetVector(id string) ([]float32, error) {
	// 先检查缓存
	cacheKey := fmt.Sprintf("vector:%s", id)
	if cached, found := vdb.cache.Get(cacheKey); found {
		if vector, ok := cached.([]float32); ok {
			return vector, nil
		}
	}

	// 检查内存索引
	vdb.mutex.RLock()
	if vector, exists := vdb.vectors[id]; exists {
		vdb.mutex.RUnlock()

		// 缓存结果
		vdb.cache.Set(cacheKey, vector, 3600)
		return vector, nil
	}
	vdb.mutex.RUnlock()

	// 从PostgreSQL获取
	if vdb.pgConn != nil {
		var vectorJSON string
		err := vdb.pgConn.QueryRow("SELECT vector_data FROM vectors WHERE id = $1", id).Scan(&vectorJSON)
		if err == nil {
			var vector []float32
			if err := json.Unmarshal([]byte(vectorJSON), &vector); err == nil {
				// 添加到内存索引
				vdb.mutex.Lock()
				vdb.vectors[id] = vector
				vdb.mutex.Unlock()

				// 缓存结果
				vdb.cache.Set(cacheKey, vector, 3600)
				return vector, nil
			}
		}
	}

	return nil, fmt.Errorf("vector not found: %s", id)
}

// DeleteVector 删除向量
func (vdb *vectorDB) DeleteVector(id string) error {
	vdb.mutex.Lock()
	defer vdb.mutex.Unlock()

	// 从内存索引删除
	delete(vdb.vectors, id)

	// 从缓存删除
	vdb.cache.Delete(fmt.Sprintf("vector:%s", id))

	// 从PostgreSQL删除
	if vdb.pgConn != nil {
		_, err := vdb.pgConn.Exec("DELETE FROM vectors WHERE id = $1", id)
		if err != nil {
			log.Printf("Failed to delete vector from database: %v", err)
		}
	}

	log.Printf("Deleted vector: %s", id)
	return nil
}

// GetVectorCount 获取向量数量
func (vdb *vectorDB) GetVectorCount() (int64, error) {
	vdb.mutex.RLock()
	count := int64(len(vdb.vectors))
	vdb.mutex.RUnlock()

	return count, nil
}

// initTables 初始化数据库表
func (vdb *vectorDB) initTables() error {
	createVectorsTable := `
		CREATE TABLE IF NOT EXISTS vectors (
			id VARCHAR(255) PRIMARY KEY,
			vector_data JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_vectors_created_at ON vectors (created_at);
		CREATE INDEX IF NOT EXISTS idx_vectors_updated_at ON vectors (updated_at);
	`

	_, err := vdb.pgConn.Exec(createVectorsTable)
	if err != nil {
		return fmt.Errorf("failed to create vectors table: %v", err)
	}

	log.Println("Database tables initialized")
	return nil
}

// sortBySimilarity 按相似度排序
func (vdb *vectorDB) sortBySimilarity(results []types.SearchResult) {
	// 简单的冒泡排序，按相似度降序排列
	n := len(results)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if results[j].Similarity < results[j+1].Similarity {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

// Close 关闭连接
func (vdb *vectorDB) Close() error {
	if vdb.pgConn != nil {
		return vdb.pgConn.Close()
	}
	return nil
}
