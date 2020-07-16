``` go
type Client interface {
	Connect() error
	Watch()
	Load() error
}

type Storage interface {
	Set(key string, value []byte)
	GetString(key string) (string,error)
	GetBytes(key string) ( []byte,error)
	GetObject(key string, target interface{}) error
	Init() error
}
```

几个主要的interface：
1. Storage 主要用来本地存储项目在配置中心的相关设置
  - demo中使用读写锁的map作为本地存储
  - 后期可以实现本接口，来更换其他存储形式
2.  Client 是用来连接远端配置中心的
 - demo中采用grpc协议，V3 api 直连etcd集群
 - 后期如果有其他协议如http需求，权限校验等更多功能可以实现本接口

待实现interface:
1. Discover 主要用于服务的注册发现
2. Proxy 主要用于流量的管控等


issue:
1. watch 的error返回？ 实测etcd down的时候，watch会处于自动重连,etcd恢复后会接到后续的event  
2. etcd 的connect是惰性连接且无限自动重连，哪怕etcd down了也不会报错,etcd恢复后会继续工作
 - 如果 get put 使用timeout context还是会返回超时err 如果是context.backgroud函数会阻塞一直到etcd恢复正常
3. 解决connect时候etcd不可用的方法：
```go
   // 1.
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 2 * time.Second)
	defer cancel()
	_, err = etcd.Status(timeoutCtx, clientConfig.Endpoints[0])
	if err != nil {
		return nil, errors.Wrapf(err, "error checking etcd status: %v", err)
	}
   //2.
    DialOptions: []grpc.DialOption{grpc.WithBlock()}    // 返回err：context deadline exceeded
// WithBlock returns a DialOption which makes caller of Dial blocks until the
// underlying connection is up. Without this, Dial returns immediately and
// connecting the server happens in background.
```
 
