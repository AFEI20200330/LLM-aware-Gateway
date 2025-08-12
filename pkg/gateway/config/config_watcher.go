package config

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/llm-aware-gateway/pkg/interfaces"
	"github.com/llm-aware-gateway/pkg/types"
)

// configWatcher 配置监听器实现
type configWatcher struct {
	etcdClient *clientv3.Client
	policies   map[string]*types.Policy
	callbacks  []interfaces.PolicyUpdateCallback
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	stopCh     chan struct{}
}

// NewConfigWatcher 创建配置监听器
func NewConfigWatcher(config *types.ETCDConfig) (interfaces.ConfigWatcher, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: config.Timeout,
		Username:    config.Username,
		Password:    config.Password,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &configWatcher{
		etcdClient: client,
		policies:   make(map[string]*types.Policy),
		ctx:        ctx,
		cancel:     cancel,
		stopCh:     make(chan struct{}),
	}, nil
}

// WatchPolicyUpdates 监听策略更新
func (cw *configWatcher) WatchPolicyUpdates() error {
	// 首先加载现有的策略
	err := cw.loadExistingPolicies()
	if err != nil {
		log.Printf("Failed to load existing policies: %v", err)
	}

	// 开始监听策略变更
	watchChan := cw.etcdClient.Watch(cw.ctx, "/policies/", clientv3.WithPrefix())

	go func() {
		for {
			select {
			case watchResp := <-watchChan:
				for _, event := range watchResp.Events {
					cw.handleConfigEvent(event)
				}
			case <-cw.stopCh:
				return
			}
		}
	}()

	log.Println("Config watcher started")
	return nil
}

// GetPolicy 获取策略
func (cw *configWatcher) GetPolicy(clusterID string) (*types.Policy, error) {
	cw.mutex.RLock()
	defer cw.mutex.RUnlock()

	policy, exists := cw.policies[clusterID]
	if !exists {
		return nil, nil
	}

	// 检查策略是否过期
	if time.Now().After(policy.ExpireTime) {
		return nil, nil
	}

	return policy, nil
}

// RegisterCallback 注册回调
func (cw *configWatcher) RegisterCallback(callback interfaces.PolicyUpdateCallback) error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	cw.callbacks = append(cw.callbacks, callback)
	return nil
}

// Start 启动配置监听器
func (cw *configWatcher) Start() error {
	return cw.WatchPolicyUpdates()
}

// Stop 停止配置监听器
func (cw *configWatcher) Stop() error {
	close(cw.stopCh)
	cw.cancel()

	if cw.etcdClient != nil {
		cw.etcdClient.Close()
	}

	log.Println("Config watcher stopped")
	return nil
}

// loadExistingPolicies 加载现有策略
func (cw *configWatcher) loadExistingPolicies() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := cw.etcdClient.Get(ctx, "/policies/", clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kv := range resp.Kvs {
		clusterID := strings.TrimPrefix(string(kv.Key), "/policies/")

		var policy types.Policy
		if err := json.Unmarshal(kv.Value, &policy); err != nil {
			log.Printf("Failed to unmarshal policy for cluster %s: %v", clusterID, err)
			continue
		}

		cw.mutex.Lock()
		cw.policies[clusterID] = &policy
		cw.mutex.Unlock()

		// 通知回调
		cw.notifyPolicyUpdate(clusterID, &policy)
	}

	log.Printf("Loaded %d existing policies", len(resp.Kvs))
	return nil
}

// handleConfigEvent 处理配置事件
func (cw *configWatcher) handleConfigEvent(event *clientv3.Event) {
	clusterID := strings.TrimPrefix(string(event.Kv.Key), "/policies/")

	switch event.Type {
	case clientv3.EventTypePut:
		var policy types.Policy
		if err := json.Unmarshal(event.Kv.Value, &policy); err != nil {
			log.Printf("Failed to unmarshal policy for cluster %s: %v", clusterID, err)
			return
		}

		cw.mutex.Lock()
		cw.policies[clusterID] = &policy
		cw.mutex.Unlock()

		// 通知回调
		cw.notifyPolicyUpdate(clusterID, &policy)

		log.Printf("Policy updated for cluster: %s", clusterID)

	case clientv3.EventTypeDelete:
		cw.mutex.Lock()
		delete(cw.policies, clusterID)
		cw.mutex.Unlock()

		// 通知回调
		cw.notifyPolicyDelete(clusterID)

		log.Printf("Policy deleted for cluster: %s", clusterID)
	}
}

// notifyPolicyUpdate 通知策略更新
func (cw *configWatcher) notifyPolicyUpdate(clusterID string, policy *types.Policy) {
	cw.mutex.RLock()
	callbacks := make([]interfaces.PolicyUpdateCallback, len(cw.callbacks))
	copy(callbacks, cw.callbacks)
	cw.mutex.RUnlock()

	for _, callback := range callbacks {
		go func(cb interfaces.PolicyUpdateCallback) {
			if err := cb.OnPolicyUpdate(clusterID, policy); err != nil {
				log.Printf("Failed to notify policy update for cluster %s: %v", clusterID, err)
			}
		}(callback)
	}
}

// notifyPolicyDelete 通知策略删除
func (cw *configWatcher) notifyPolicyDelete(clusterID string) {
	cw.mutex.RLock()
	callbacks := make([]interfaces.PolicyUpdateCallback, len(cw.callbacks))
	copy(callbacks, cw.callbacks)
	cw.mutex.RUnlock()

	for _, callback := range callbacks {
		go func(cb interfaces.PolicyUpdateCallback) {
			if err := cb.OnPolicyDelete(clusterID); err != nil {
				log.Printf("Failed to notify policy delete for cluster %s: %v", clusterID, err)
			}
		}(callback)
	}
}
