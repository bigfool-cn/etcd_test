package main

import (
	"context"
	"etcd_test/grpclb-balancer/etcdv3"
	pb "etcd_test/grpclb-balancer/proto"
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

// SimpleService 定义我们的服务
type SimpleService struct{}

const (
	Ip string = "127.0.0.1"
	// Network 网络通信协议
	Network string = "tcp"
	// SerName 服务名称
	SerName string = "simple_grpc"
)

// EtcdEndpoints etcd地址
var EtcdEndpoints = []string{"127.0.0.1:32379"}

func main() {
	var port = flag.Int("port", 8000, "listening port")
	var weight = flag.String("weight", "1", "balancer weight")

	flag.Parse()
	var address = fmt.Sprintf("%s:%d", Ip, *port)
	// 监听本地端口
	listener, err := net.Listen(Network, address)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}
	log.Println(address + " net.Listing...")
	// 新建gRPC服务器实例
	grpcServer := grpc.NewServer()
	// 在gRPC服务器注册我们的服务
	pb.RegisterSimpleServer(grpcServer, &SimpleService{})
	//把服务注册到etcd
	ser, err := etcdv3.NewServiceRegister(EtcdEndpoints, SerName+"/"+address, *weight, 5)
	if err != nil {
		log.Fatalf("register service err: %v", err)
	}
	defer ser.Close()
	//用服务器 Serve() 方法以及我们的端口信息区实现阻塞等待，直到进程被杀死或者 Stop() 被调用
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}

// Route 实现Route方法
func (s *SimpleService) Route(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	log.Println("receive: " + req.Data)
	res := pb.SimpleResponse{
		Code:  200,
		Value: "hello " + req.Data,
	}
	return &res, nil
}
