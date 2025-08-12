package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// GenerateID 生成唯一ID
func GenerateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GenerateClusterID 生成簇ID
func GenerateClusterID() string {
	return fmt.Sprintf("cluster_%d_%s", time.Now().Unix(), GenerateID()[:8])
}

// GeneratePolicyID 生成策略ID
func GeneratePolicyID() string {
	return fmt.Sprintf("policy_%d_%s", time.Now().Unix(), GenerateID()[:8])
}

// ExtractTraceID 从Gin上下文提取TraceID
func ExtractTraceID(ctx *gin.Context) string {
	span := trace.SpanFromContext(ctx.Request.Context())
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// ExtractSpanID 从Gin上下文提取SpanID
func ExtractSpanID(ctx *gin.Context) string {
	span := trace.SpanFromContext(ctx.Request.Context())
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// ExtractServiceName 从请求路径提取服务名
func ExtractServiceName(ctx *gin.Context) string {
	path := ctx.Request.URL.Path
	parts := strings.Split(path, "/")

	if len(parts) >= 2 && parts[1] != "" {
		return parts[1]
	}

	return "unknown"
}

// ExtractStackTrace 提取堆栈信息
func ExtractStackTrace(err error, maxFrames int) []string {
	if err == nil {
		return nil
	}

	// 获取堆栈信息
	pcs := make([]uintptr, maxFrames+1)
	n := runtime.Callers(2, pcs)

	var traces []string
	frames := runtime.CallersFrames(pcs[:n])

	for i := 0; i < maxFrames; i++ {
		frame, more := frames.Next()
		if !more {
			break
		}

		trace := fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function)
		traces = append(traces, trace)
	}

	return traces
}

// ExtractErrorSignature 提取错误签名
func ExtractErrorSignature(ctx *gin.Context) string {
	// 从上下文获取错误信息
	if err, exists := ctx.Get("error"); exists {
		if e, ok := err.(error); ok {
			return e.Error()
		}
	}
	return ""
}

// CosineSimilarity 计算余弦相似度
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// EuclideanDistance 计算欧几里得距离
func EuclideanDistance(a, b []float32) float64 {
	if len(a) != len(b) {
		return math.Inf(1)
	}

	var sum float64
	for i := range a {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}

	return math.Sqrt(sum)
}

// NormalizeVector 向量归一化
func NormalizeVector(vector []float32) []float32 {
	var norm float64
	for _, v := range vector {
		norm += float64(v * v)
	}

	norm = math.Sqrt(norm)
	if norm == 0 {
		return vector
	}

	normalized := make([]float32, len(vector))
	for i, v := range vector {
		normalized[i] = float32(float64(v) / norm)
	}

	return normalized
}

// CalculateVectorCentroid 计算向量中心点
func CalculateVectorCentroid(vectors [][]float32) []float32 {
	if len(vectors) == 0 {
		return nil
	}

	dimension := len(vectors[0])
	centroid := make([]float32, dimension)

	for _, vector := range vectors {
		for i, v := range vector {
			centroid[i] += v
		}
	}

	count := float32(len(vectors))
	for i := range centroid {
		centroid[i] /= count
	}

	return centroid
}

// Float64ToString 将float64转换为字符串
func Float64ToString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// StringToFloat64 将字符串转换为float64
func StringToFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// Int64ToString 将int64转换为字符串
func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

// StringToInt64 将字符串转换为int64
func StringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// Truncate 截断字符串到指定长度
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// FormatDuration 格式化时间间隔
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	} else if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.2fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.2fh", d.Hours())
	}
}

// Min 返回两个float64中的较小值
func MinFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Max 返回两个float64中的较大值
func MaxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// ClampFloat64 将float64值限制在指定范围内
func ClampFloat64(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
