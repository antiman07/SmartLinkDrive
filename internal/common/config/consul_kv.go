package config

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/consul/api"
)

// LoadConfigFromConsulKV 从 Consul KV 读取 JSON 配置并解析为 Config。
//
// 约定：
// - key 对应的 value 必须是 JSON（结构与 Config 一致）
// - 该函数只负责“读取 + 解析”，是否做动态 watch 由上层决定
func LoadConfigFromConsulKV(consulHost string, consulPort int, key string) (*Config, error) {
	if key == "" {
		return nil, fmt.Errorf("consul kv key is empty")
	}

	c, err := api.NewClient(&api.Config{
		Address: fmt.Sprintf("%s:%d", consulHost, consulPort),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	kv := c.KV()
	pair, _, err := kv.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get consul kv key=%s: %w", key, err)
	}
	if pair == nil || len(pair.Value) == 0 {
		return nil, fmt.Errorf("consul kv key=%s is empty or not found", key)
	}

	cfg := &Config{}
	if err := json.Unmarshal(pair.Value, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal consul kv json key=%s: %w", key, err)
	}
	return cfg, nil
}
