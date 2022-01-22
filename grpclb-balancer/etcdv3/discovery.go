package etcdv3

import (
	"context"
	weight "etcd_test/grpclb-balancer/balancer"
	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

const schema = "grpclb"

// ServiceDiscovery 服务发现
type ServiceDiscovery struct {
	cli        *clientv3.Client
	cc         resolver.ClientConn
	serverList sync.Map //服务列表
	prefix     string   //监视的前缀
}

// NewServiceDiscovery 发现服务
func NewServiceDiscovery(endpoint []string) resolver.Builder {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoint,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	return &ServiceDiscovery{
		cli: cli,
	}
}

func (s *ServiceDiscovery) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	log.Println("Build")
	s.cc = cc
	s.prefix = "/" + target.Scheme + "/" + target.Endpoint + "/"
	// 根据前缀获取现有的key
	resp, err := s.cli.Get(context.Background(), s.prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	for _, ev := range resp.Kvs {
		s.SetServerList(string(ev.Key), string(ev.Value))
	}
	s.cc.UpdateState(resolver.State{Addresses: s.getServices()})

	// 监听后续操作
	go s.watcher(s.prefix)

	return s, nil
}

func (s *ServiceDiscovery) ResolveNow(rn resolver.ResolveNowOptions) {
	log.Println("ResolveNow")
}

func (s *ServiceDiscovery) Scheme() string {
	return schema
}

func (s *ServiceDiscovery) Close() {
	log.Println("Close")
	s.cli.Close()
}

// 监听前缀
func (s *ServiceDiscovery) watcher(prefix string) {
	rch := s.cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
	log.Printf("watching prefix: %s now...", prefix)
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case clientv3.EventTypePut:
				// 修改或新增
				s.SetServerList(string(ev.Kv.Key), string(ev.Kv.Value))
			case clientv3.EventTypeDelete:
				// 删除
				s.DelServerList(string(ev.Kv.Key))
			}
		}
	}
}

// SetServerList 新增服务地址
func (s *ServiceDiscovery) SetServerList(key, val string) {
	//获取服务地址
	addr := resolver.Address{Addr: strings.TrimPrefix(key, s.prefix)}
	//获取服务地址权重
	nodeWeight, err := strconv.Atoi(val)
	if err != nil {
		//非数字字符默认权重为1
		nodeWeight = 1
	}
	//把服务地址权重存储到resolver.Address的元数据中
	addr = weight.SetAddrInfo(addr, weight.AddrInfo{Weight: nodeWeight})
	s.serverList.Store(key, addr)
	s.cc.UpdateState(resolver.State{Addresses: s.getServices()})
	log.Println("put key :", key, "weight:", val)
}

// DelServerList 删除服务地址
func (s *ServiceDiscovery) DelServerList(key string) {
	s.serverList.Delete(key)
	s.cc.UpdateState(resolver.State{Addresses: s.getServices()})
	log.Println("del key:", key)
}

// GetServices 获取服务地址
func (s *ServiceDiscovery) getServices() []resolver.Address {
	addrs := make([]resolver.Address, 0, 10)
	s.serverList.Range(func(k, v interface{}) bool {
		addrs = append(addrs, v.(resolver.Address))
		return true
	})
	return addrs
}
