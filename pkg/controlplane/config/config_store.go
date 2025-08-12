package config

import (
	"context"
	"log"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/llm-aware-gateway/pkg/interfaces"
	"github.com/llm-aware-gateway/pkg/types"
)

// etcdConfigStore ETCD配置存储实现
type etcdConfigStore struct {
	client *clientv3.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// NewETCDConfigStore 创建ETCD配置存储
func NewETCDConfigStore(config *types.ETCDConfig) (interfaces.ConfigStore, error) {
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

	return &etcdConfigStore{
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Put 存储键值对
func (ecs *etcdConfigStore) Put(key string, value string) error {
	ctx, cancel := context.WithTimeout(ecs.ctx, 5*time.Second)
	defer cancel()

	_, err := ecs.client.Put(ctx, key, value)
	if err != nil {
		return err
	}

	log.Printf("Stored config: %s", key)
	return nil
}

// Get 获取值
func (ecs *etcdConfigStore) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(ecs.ctx, 5*time.Second)
	defer cancel()

	resp, err := ecs.client.Get(ctx, key)
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", nil
	}

	return string(resp.Kvs[0].Value), nil
}

// Delete 删除键
func (ecs *etcdConfigStore) Delete(key string) error {
	ctx, cancel := context.WithTimeout(ecs.ctx, 5*time.Second)
	defer cancel()

	_, err := ecs.client.Delete(ctx, key)
	if err != nil {
		return err
	}

	log.Printf("Deleted config: %s", key)
	return nil
}

// Watch 监听键变化
func (ecs *etcdConfigStore) Watch(prefix string) (<-chan *interfaces.ConfigChangeEvent, error) {
	watchChan := ecs.client.Watch(ecs.ctx, prefix, clientv3.WithPrefix())
	eventChan := make(chan *interfaces.ConfigChangeEvent, 100)

	go func() {
		defer close(eventChan)

		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				changeEvent := &interfaces.ConfigChangeEvent{
					Key:   string(event.Kv.Key),
					Value: string(event.Kv.Value),
				}

				switch event.Type {
				case clientv3.EventTypePut:
					changeEvent.Type = interfaces.ConfigChangeTypePut
				case clientv3.EventTypeDelete:
					changeEvent.Type = interfaces.ConfigChangeTypeDelete
				}

				select {
				case eventChan <- changeEvent:
				case <-ecs.ctx.Done():
					return
				}
			}
		}
	}()

	return eventChan, nil
}

// Close 关闭连接
func (ecs *etcdConfigStore) Close() error {
	ecs.cancel()
	return ecs.client.Close()
}

// ListKeys 列出所有匹配前缀的键
func (ecs *etcdConfigStore) ListKeys(prefix string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ecs.ctx, 5*time.Second)
	defer cancel()

	resp, err := ecs.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		return nil, err
	}

	keys := make([]string, len(resp.Kvs))
	for i, kv := range resp.Kvs {
		keys[i] = string(kv.Key)
	}

	return keys, nil
}

// GetWithPrefix 获取所有匹配前缀的键值对
func (ecs *etcdConfigStore) GetWithPrefix(prefix string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ecs.ctx, 5*time.Second)
	defer cancel()

	resp, err := ecs.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		value := string(kv.Value)
		result[key] = value
	}

	return result, nil
}
