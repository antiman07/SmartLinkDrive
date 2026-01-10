package discovery

import (
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/resolver"
)

const (
	consulScheme = "consul"
)

// ConsulResolver Consul服务解析器
type ConsulResolver struct {
	client     *api.Client
	cc         resolver.ClientConn
	service    string
	watchers   map[string]*consulWatcher
	watchersMu sync.RWMutex
}

type consulWatcher struct {
	client    *api.Client
	service   string
	addrs     []resolver.Address
	lastIndex uint64
}

// NewConsulResolver 创建Consul解析器
func NewConsulResolver(client *api.Client, service string, cc resolver.ClientConn) *ConsulResolver {
	r := &ConsulResolver{
		client:   client,
		cc:       cc,
		service:  service,
		watchers: make(map[string]*consulWatcher),
	}
	resolver.Register(r)
	return r
}

// Build 构建解析器
func (r *ConsulResolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	watcher := &consulWatcher{
		client:  r.client,
		service: r.service,
	}

	go watcher.watch(cc)
	return r, nil
}

// Scheme 返回scheme
func (r *ConsulResolver) Scheme() string {
	return consulScheme
}

// ResolveNow 立即解析
func (r *ConsulResolver) ResolveNow(resolver.ResolveNowOptions) {}

// Close 关闭解析器
func (r *ConsulResolver) Close() {}

func (w *consulWatcher) watch(cc resolver.ClientConn) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.update(cc)
		}
	}
}

func (w *consulWatcher) update(cc resolver.ClientConn) {
	services, meta, err := w.client.Health().Service(w.service, "", true, &api.QueryOptions{
		WaitIndex: w.lastIndex,
	})
	if err != nil {
		return
	}

	w.lastIndex = meta.LastIndex

	addrs := make([]resolver.Address, 0, len(services))
	for _, service := range services {
		addr := fmt.Sprintf("%s:%d", service.Service.Address, service.Service.Port)
		addrs = append(addrs, resolver.Address{
			Addr: addr,
		})
	}

	if len(addrs) > 0 {
		cc.UpdateState(resolver.State{
			Addresses: addrs,
		})
		w.addrs = addrs
	}
}

// ServiceRegistry Consul服务注册
type ServiceRegistry struct {
	client    *api.Client
	serviceID string
	service   string
	address   string
	port      int
	tags      []string
	check     *api.AgentServiceCheck
}

// NewServiceRegistry 创建服务注册器
func NewServiceRegistry(client *api.Client, serviceID, service, address string, port int, tags []string) *ServiceRegistry {
	return &ServiceRegistry{
		client:    client,
		serviceID: serviceID,
		service:   service,
		address:   address,
		port:      port,
		tags:      tags,
		check: &api.AgentServiceCheck{
			GRPC:                           fmt.Sprintf("%s:%d", address, port),
			Interval:                       "10s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}
}

// Register 注册服务
func (r *ServiceRegistry) Register() error {
	registration := &api.AgentServiceRegistration{
		ID:      r.serviceID,
		Name:    r.service,
		Tags:    r.tags,
		Address: r.address,
		Port:    r.port,
		Check:   r.check,
	}

	return r.client.Agent().ServiceRegister(registration)
}

// Deregister 注销服务
func (r *ServiceRegistry) Deregister() error {
	return r.client.Agent().ServiceDeregister(r.serviceID)
}

// NewConsulClient 创建Consul客户端
func NewConsulClient(host string, port int) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = fmt.Sprintf("%s:%d", host, port)
	return api.NewClient(config)
}
