package etcdv3

import (
	"context"
	"go.etcd.io/etcd/client/v3"
	"log"
	"time"
)

// ServiceRegister 注册租约服务
type ServiceRegister struct {
	cli           *clientv3.Client                        // etcd 客户端
	leaseID       clientv3.LeaseID                        // 租约ID
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse // 租约keepalive相应的chan
	key           string
	val           string
}

// NewServiceRegister 注册服务
func NewServiceRegister(endpoint []string, serName, addr string, lease int64) (*ServiceRegister, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoint,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	ser := &ServiceRegister{
		cli: cli,
		key: "/" + schema + "/" + serName + "/" + addr,
		val: addr,
	}

	if err = ser.putKeyWithLease(lease); err != nil {
		return nil, err
	}

	return ser, nil
}

// 续租
func (s *ServiceRegister) putKeyWithLease(lease int64) error {
	// 设置租约时间
	resp, err := s.cli.Grant(context.Background(), lease)
	if err != nil {
		return err
	}

	// 注册服务并绑定租约
	_, err = s.cli.Put(context.Background(), s.key, s.val, clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}

	// 设置续租， 定期发送需求请求
	leaseRespChan, err := s.cli.KeepAlive(context.Background(), resp.ID)
	if err != nil {
		return err
	}

	s.leaseID = resp.ID
	log.Println(s.leaseID)
	s.keepAliveChan = leaseRespChan
	log.Printf("Put key:%s  val:%s  success!", s.key, s.val)

	return nil
}

// ListenLeaseRespChan 监听续租
func (s *ServiceRegister) ListenLeaseRespChan() {
	for leaseKeepResp := range s.keepAliveChan {
		log.Println("续租成功", leaseKeepResp)
	}
	log.Println("关闭续租")
}

// Close 注销服务
func (s *ServiceRegister) Close() error {
	if _, err := s.cli.Revoke(context.Background(), s.leaseID); err != nil {
		return err
	}
	log.Println("撤销续租")
	return nil
}
