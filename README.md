# grpc
本项目扩充了一些 grpc 开发中常用的组件，使其具有基本完备的功能，可以拿来即用。

包含的功能有：
- server 和 client 使用范例
- TSL 证书
- 中间件
- 请求超时，重试，取消
- 支持 http 请求
- grpc-gateway
- grpc 调试


## 安装
```go
//安装protoc,protoc 是 Protobuf 的编译器
wget http://github.com/protocolbuffers/protobuf/releases/download/v3.17.3/protobuf-all-3.17.3.zip
unzip protobuf-all-3.17.3.zip
cd protobuf-3.17.3/
./configure
make
make install

//protoc 基于插件来生成对应的语言, 生成Go语言的插件是 protoc-gen-go
go get -u github.com/golang/protobuf/protoc-gen-go

//安装Grpc
go get -u google.golang.org/grpc
```



## 定义 Protobuf 文件
Grpc 最基本的开发步骤是定义 proto 文件， 定义请求 Request 和 响应 Response 的格式，然后定义一个服务 Service， Service可以包含多个方法。
```go
//声明使用 proto3 语法,如果不声明,将默认使用 proto2 语法
syntax = "proto3";

//option go_package = "path;name"; path 定义了生成go文件的存放地址,name定义了go文件所属的包名
option go_package="./;proto";

package proto;

//定义服务
service SearchService {
    //服务提供的方法,入参为 SearchRequest 对象,出参为 SearchResponse 对象
    rpc Search(SearchRequest) returns (SearchResponse)
}

//定义请求参数对象
message SearchRequest {
    //每一个字段包含三个属性：类型、字段名称、字段编号(编号可以不唯一,但不能重复)
    string request = 1;
}

//定义响应参数对象
message SearchResponse {
    string response = 1;
}
```
注意定义方法的语法格式：
```go
rpc 函数名 (参数) returns (返回值) {}
```


## 编译 Protobuf 文件
```go
//语法格式
protoc --go_out=plugins=xxx:代码输出目录 proto文件

//将当前目录下的 *.proto 文件使用 grpc 编译后存放于当前目录下
protoc --go_out=plugins=grpc:. *.proto

//将当前目录下的 ProductInfo.proto 文件使用 grpc 编译后存放于上级目录下的product中
protoc --go_out=plugins=grpc:../product ProductInfo.proto
```

选项说明：
- --go_out=:  设置 Go 代码输出的目录。
- plugins=xxx 指定要编译proto文件所使用的插件。
- :. *.proto 定义了生成文件的存放目录和要编译的 proto 文件。


## Sever 和 Client
我们要填充业务代码必须实现 xxx.pb.go 中的两个接口：XXXServiceClient 和  XXXServiceServer，这两个接口中内嵌了我们定义的 service rpc 方法，这些方法需要实现它，这样就能填充我们的业务逻辑。
```go
//grpc server
package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "go-grpc/proto"
)

//SearchService 定义服务，需实现了SearchServiceServer，这样该服务才能注册
type SearchService struct{}
//Search 实现了SearchServiceServer接口定义的方法,实现该方法就能填充我们的业务逻辑
func (s *SearchService) Search(ctx context.Context, r *pb.SearchRequest) (*pb.SearchResponse, error) {
	return &pb.SearchResponse{Response: r.GetRequest() + " Server"}, nil
}

const PORT = "9001"

func main() {
	//创建 gRPC Server,用来注册服务
	server := grpc.NewServer()
	//注册服务
	pb.RegisterSearchServiceServer(server, &SearchService{})
	//监听tcp请求
	lis, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}
	//处理
	server.Serve(lis)
}
```

```go
//grpc 客户端
package main

import (
	"context"
	pb "go-grpc/proto"
	"google.golang.org/grpc"
	"log"
)

const PORT = "9001"

func main() {
	conn, err := grpc.Dial(":"+PORT, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("grpc.Dial err: %v", err)
	}
	defer conn.Close()

	client := pb.NewSearchServiceClient(conn)
	//调用服务端 Search 方法
	resp, err := client.Search(context.Background(), &pb.SearchRequest{
		Request: "gRPC",
	})
	if err != nil {
		log.Fatalf("client.Search err: %v", err)
	}

	log.Printf("resp: %s", resp.GetResponse())
}
```


## TLS 证书
go 1.15 版本开始废弃 CommonName 方式生成的证书，因此推荐使用 SAN 证书。 如果想兼容之前的方式，需要设置环境变量 GODEBUG 为 x509ignoreCN=0。
```go
$ go run client.go

rpc error: code = Unavailable desc = connection error: desc = "transport
: authentication handshake failed: x509: certificate relies on legacy Common Name field, use SANs or temporaril
y enable Common Name matching with GODEBUG=x509ignoreCN=0"
```

兼容方式：
```go
GODEBUG="x509ignoreCN=0" go run client.go

//下面这种方式不行
os.Setenv("GODEBUG", "x509ignoreCN=0")
```

关于生成SAN证书见：
- https://docs.azure.cn/zh-cn/articles/azure-operations-guide/application-gateway/aog-application-gateway-howto-create-self-signed-cert-via-openssl     自签名和SAN 证书设置https://eddycjy.com/posts/go/grpc/2018-10-08-ca-tls 自建CA签发证书
- https://mp.weixin.qq.com/s/Qa3YZcfl-JaJ6iQd3mXptQ GRPC+SAN证书

客户端集成证书：
```go
func main() {
	cert, err := tls.LoadX509KeyPair("../../cert/client.pem", "../../cert/client.key")
	if err != nil {
		log.Fatalf("tls.LoadX509KeyPair err: %v", err)
	}

	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile("../../cert/ca.pem")
	if err != nil {
		log.Fatalf("ioutil.ReadFile err: %v", err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatalf("certPool.AppendCertsFromPEM err")
	}

	c := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   "go-grpc-example",
		RootCAs:      certPool,
	})

	conn, err := grpc.Dial(":"+PORT, grpc.WithTransportCredentials(c))
	...
}
```
服务端集成证书：
```go
func main() {
	cert, err := tls.LoadX509KeyPair("../../cert/server.pem", "../../cert/server.key")
	if err != nil {
		log.Fatalf("tls.LoadX509KeyPair err: %v", err)
	}

	certPool := x509.NewCertPool()
	ca, err  := ioutil.ReadFile("../../cert/ca.pem")
	if err != nil {
		log.Fatalf("ioutil.ReadFile err: %v", err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatalf("certPool.AppendCertsFromPEM err")
	}

	c := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	})

	server := grpc.NewServer(grpc.Creds(c))
	pb.RegisterSearchServiceServer(server, &SearchService{})

	lis, err := net.Listen("tcp", ":"+PORT)
    ...    
}
```

## 中间件
在 gRPC 中的中间件大致可以分为两类：
- 一元拦截器（grpc.UnaryInterceptor）
- 流拦截器（grpc.StreamInterceptor）

注册进grpc的中间件必须要实现如下函数:
```go
//一元拦截器函数:
  // ctx context.Context：请求上下文
  // req interface{}：RPC 方法的请求参数
  // info *UnaryServerInfo：RPC 方法的所有信息
  // handler UnaryHandler：RPC 方法本身
type UnaryServerInterceptor func(ctx context.Context, req interface{}, info *UnaryServerInfo, handler UnaryHandler) (resp interface{}, err error)

//流拦截器函数
type StreamServerInterceptor func(srv interface{}, ss ServerStream, info *StreamServerInfo, handler StreamHandler) error
```
实现：
```go
// 一元拦截器实现
func orderUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (res interface{}, err error) {
   //错误拦截
   if req!= "gRPC" {
		return nil, status.Errorf(codes.DeadlineExceeded, "cannot access!!!")
   }
    
   //前置处理
   log.Println("==========[Server Unary Interceptor]===========", info.FullMethod)

   //完成方法的正常执行
   res, err = handler(ctx, req)

   //后置处理
   log.Printf("After method call, res = %+v\n", res)
   return
}


// 流拦截器实现
type WrappedServerStream struct {
   grpc.ServerStream
}

func (w *WrappedServerStream) SendMsg(m interface{}) error {
   //后置处理 
   log.Printf("[order stream server interceptor] send a msg : %+v", m)
   return w.ServerStream.SendMsg(m)
}

func (w *WrappedServerStream) RecvMsg(m interface{}) error {
   //前置处理
   log.Printf("[order stream server interceptor] recv a msg : %+v", m)
   return w.ServerStream.RecvMsg(m)
}

func NewWrappedServerStream(s grpc.ServerStream) *WrappedServerStream {
   return &WrappedServerStream{s}
}

func orderStreamServerInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
   log.Printf("=========[order stream]start %s\n", info.FullMethod)
   err := handler(srv, NewWrappedServerStream(ss))
   if err != nil {
      log.Println("handle method err.", err)
   }
   log.Printf("=========[order stream]end")
   return nil
}


// 注册拦截器
s := grpc.NewServer(
   grpc.UnaryInterceptor(orderUnaryServerInterceptor),
   grpc.StreamInterceptor(orderStreamServerInterceptor),
)
...
```

## 请求超时、重试、取消
这块主要针对的是客户端。

**超时**
> 微服务中，服务之间的调用往往存在不确定性，比如网络环境差，数据聚合量太大等等，都有可能会导致服务调用者长时间等待响应结果。go 语言中对 gRPC 请求超时的控制，主要是使用 context 来实现。
```go
func main () {
    ...
    // 使用带有截止时间的context
    ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5 * time.Second))
    defer cancel()

    client := pb.NewSearchServiceClient(conn)
    //如果 Search 方法的调用超过了截止时间，那么调用就会被取消
    resp, err := client.Search(ctx, &pb.SearchRequest{
        Request: "gRPC",
    })
    if err != nil {
        log.Fatalf("client.Search err: %v", err)
    }

    log.Printf("resp: %s", resp.GetResponse())
    ...
}
```

**取消**
> 在某些情况下，我们可能需要取消 RPC 请求，这和设置截止时间有些类似，都是为了避免请求挂起让客户端一直等待，在 Go 语言中，取消的操作仍然借助于 context 来实现。
```go
// 取消RPC请求
func cancelRpcRequest(client order.OrderManagementClient) {
   ctx, cancelFunc := context.WithCancel(context.Background())
   done := make(chan string)
   go func() {
      var id string
      defer func() {
         fmt.Println("结束执行, id = ", id)
         done <- id
      }()
		
      //执行不到这里就取消了 
      time.Sleep(2 * time.Second)
      id = AddOrder(ctx, client)
      log.Println("添加订单成功, id = ", id)
   }()

   //等待一秒后取消
   time.Sleep(time.Second)
   cancelFunc()

   <-done
}
```

**重试**
> 借助grpc的客户端拦截器来实现重试，重试需要处理好幂等的问题。
```go
opts := []grpc.DialOption {
		grpc.WithInsecure(),
}
opts = append(opts, grpc.WithUnaryInterceptor(
    grpc_middleware.ChainUnaryClient(
        grpc_retry.UnaryClientInterceptor(
            grpc_retry.WithMax(2),
            grpc_retry.WithCodes(
                codes.Unknown,
                codes.Internal,
                codes.DeadlineExceeded,
            ),
        ),
    ),
))
conn, err := grpc.Dial(":"+PORT, opts...)
```

## 支持Http请求

> 如果既想提供 rpc 服务，又想提供 http 服务，可以通过检测请求协议是否为 HTTP/2，以及 Content-Type 是否为 application/grpc，这样就能根据协议的不同转发到不同的服务处理。
```go
if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
    server.ServeHTTP(w, r)
} else {
    mux.ServeHTTP(w, r)
}
```

示例：
```go
func main() {
	...
    
	//实例化rpc服务器
	server := grpc.NewServer(opts...)
	pb.RegisterSearchServiceServer(server, &SearchService{})

	//实例化http服务器
	mux := GetHTTPServeMux()
	certFile := "../../cert/server.pem"
	keyFile := "../../cert/server.key"

	http.ListenAndServeTLS(":"+PORT,
		certFile,
		keyFile,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            //基于协议判断分发给哪个服务
			if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
				server.ServeHTTP(w, r)
			} else {
				mux.ServeHTTP(w, r)
			}
			return
		}),
	)
}

//GetHTTPServeMux 实例化http服务器
func GetHTTPServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("http Server"))
	})

	return mux
}
...
```

## grpc-gateway
grpc-gateway 是 protoc 的一个插件，它能够读取 protobuf 的服务定义，并生成一个反向代理服务器，将 RESTful JSON API 转换为 gRPC，实现同一个 RPC 方法提供 rpc 和 api 调用。

![](https://s3.bmp.ovh/imgs/2022/04/25/1cbdf915303c1694.jpg)


proto 文件修改
> 要支持grpc-gateway，需要在proto文件中引入google/api/annotations.proto(该文件针对HTTP 转换提供了支持)，并在对应的 rcp 方法新增针对 HTTP 路由的注解。
```go
syntax = "proto3";

option go_package=".;proto";
package proto;

import "google/api/annotations.proto";

service SearchService {
    rpc Search(SearchRequest) returns (SearchResponse) {
        //新增 option 以支持http请求
        option (google.api.http) = {
            get: "/search"
        };
    }
}

message SearchRequest {
    string request = 1;
}

message SearchResponse {
    string response = 1;
}
```

还可以通过 `additional_bindings` 来支持`多个http方法访问`，例如既可以使用GET请求，也可以使用POST来请求，甚至还可以为接口`取一个别名`。
```go
syntax = "proto3";

option go_package=".;proto";
package proto;

import "google/api/annotations.proto";

service SearchService {
    rpc Search(SearchRequest) returns (SearchResponse) {
        //新增 option 以支持http请求
        option (google.api.http) = {
            get: "/search"
            //支持post请求
            additional_bindings {
                post: "/search"
            }
            //支持PUT请求
            additional_bindings {
                put: "/search"
            }
            //接口别名
            additional_bindings {
                get: "/v2/search"
            }
        };
    }
}

message SearchRequest {
    string request = 1;
}

message SearchResponse {
    string response = 1;
}
```

重新编译proto文件，proto 目录下将生成 .pb.go和.pb.gw.go 两种文件，分别对应 rpc 和 http 的支持。
> 注意：grpc-gateway 路径指向你自己的路径，选项-I 的作用是指定查找 annotations.proto 的位置。
```go
protoc -I/usr/local/include -I. \
       -I/mnt/d/docker/go/pkg/mod \
       -I/mnt/d/docker/go/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0/third_party/googleapis \
       --grpc-gateway_out=logtostderr=true:. search.proto
```

gataway 逻辑实现:
```go
package main

import (
	"context"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pb "go-grpc/proto"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net/http"
	"strings"
)

//SearchService 定义服务，需实现了SearchServiceServer，这样该服务才能注册
type SearchService struct{}

//Search 实现了SearchServiceServer接口定义的方法,实现该方法就能填充我们的业务逻辑
func (s *SearchService) Search(ctx context.Context, r *pb.SearchRequest) (*pb.SearchResponse, error) {
	return &pb.SearchResponse{Response: r.GetRequest() + " Server"}, nil
}

const PORT = "9001"
const ENDPOINT = ":" + PORT

func main() {
	httpServer := httpServer()
	grpcServer := grpcServer()
	gateway    := grpcGateway()

	httpServer.Handle("/", gateway)

	http.ListenAndServe(ENDPOINT, grpcHandlerFunc(grpcServer, httpServer))
	fmt.Println("hello")
}

func grpcHandlerFunc(grpcServer *grpc.Server, httpServer http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			httpServer.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}

func httpServer() *http.ServeMux {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`pong`))
	})

	return serveMux
}

func grpcServer() *grpc.Server {
	server := grpc.NewServer()
	pb.RegisterSearchServiceServer(server, &SearchService{})
	reflection.Register(server)
	return server
}

func grpcGateway() *runtime.ServeMux {
	//endpoint := "0.0.0.0:" + port
	gwmux := runtime.NewServeMux()
	dopts := []grpc.DialOption{grpc.WithInsecure()}
	_ = pb.RegisterSearchServiceHandlerFromEndpoint(context.Background(), gwmux, ENDPOINT, dopts)
	return gwmux
}
```

服务调用:
```go
//rcp 方式
$ grpcurl -plaintext -d '{"request":"Gprc"}' localhost:9001 proto.SearchService.Search  
{
  "response": "Grpc Server"
}

//http 方式
$ curl localhost:9001/search?request=Http
{"response":"Http Server"}
```


## grpc 调试
安装调试工具:
> gRPC 是基于 HTTP/2 协议的，因此不像普通的 HTTP/1.1 接口可以直接通过 postman 或普通的 curl 进行调用，因此需要专用的调试工具  grpcurl 来调试。
```go
$ go get github.com/fullstorydev/grpcurl
$ go install github.com/fullstorydev/grpcurl/cmd/grpcurl
```

注册反射服务:
> grpcurl 工具的使用前提是 gRPC Server 已经注册了反射服务。
```go
import (
    "google.golang.org/grpc/reflection"
    ...
)

func main() {
	s := grpc.NewServer()
	pb.RegisterTagServiceServer(s, server.NewTagServer())
    //注册反射服务
	reflection.Register(s)
	...
}
```

调试:
```go
//查看服务列表, plaintext：该选项用来忽略 TLS 认证
$ grpcurl -plaintext localhost:9001 list   
grpc.reflection.v1alpha.ServerReflection
proto.SearchService

//查看服务下面的方法
$ grpcurl -plaintext localhost:9001 list proto.SearchService
proto.SearchService.Search


//调试方法
$ grpcurl -plaintext -d '{"request":"Gprc"}' localhost:9001 proto.SearchService.Search  
{
  "response": "Grpc Server"
}
```


## 参考
- https://eddycjy.com/go-categories/  
- https://zhuanlan.zhihu.com/p/359968500 
- https://www.lixueduan.com/tags/
- https://golang2.eddycjy.com/posts/ch3/02-simple-protobuf 

