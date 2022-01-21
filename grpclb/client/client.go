package main

import (
	"context"
	"etcd_test/grpclb/etcdv3"
	pb "etcd_test/grpclb/proto"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
	"log"
	"strconv"
	"time"
)

var (
	EtcdEndpoints = []string{"192.168.63.55:40032"}
	SerName       = "simple_grpc"
	grpcClient    pb.SimpleClient
)

func main() {
	r := etcdv3.NewServiceDiscovery(EtcdEndpoints)
	resolver.Register(r)
	// 连接服务器
	conn, err := grpc.Dial(fmt.Sprintf("%s///%s", r.Scheme(), SerName), grpc.WithBalancerName("round_robin"), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("net.Connect err: %v", err)
	}
	defer conn.Close()

	// 建立grpc连接
	grpcClient = pb.NewSimpleClient(conn)
	for idx := 0; idx < 100; idx++ {
		// 创建发送结构体
		req := pb.SimpleRequest{
			Data: "grpc " + strconv.Itoa(idx),
		}
		// 调用我们的服务(Route方法)
		// 同时传入了一个 context.Context ，在有需要时可以让我们改变RPC的行为，比如超时/取消一个正在运行的RPC
		res, err := grpcClient.Route(context.Background(), &req)
		if err != nil {
			log.Fatalf("Call Route err: %v", err)
		}
		// 打印返回值
		log.Println(res)

		time.Sleep(1 * time.Second)
	}
}
