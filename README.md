# istio上尝试jaeger的简单例子

## 1、本地尝试

```shell
# 启动jaeger all-in-one容器
$ ./start.sh

# 尝试直接向jaeger发送数据，默认是向jaeger:6831 UDP端口发送
$ go run main.go

# 发送完成后访问jaeger UI查看是否发送成功 jaeger:16686
```



## 2、多个服务中进行链路追踪

```shell
# 启动服务端
$ go run sample/server/main.go

# 使用客户端发送请求
$ go run smaple/client/main.go

# 查看UI
```



### 3、在istio中尝试

```shell
# 在istio注入的namespaces中启动两个golang容器
# 在第一个go容器中启动server
$ go run chain/server/main.go

# 在第二个go容器中启动queryBalance
$ go run chain/queryBalance/main.go

# 接下来在其他被istio注入的容器，或者直接在kubernetes宿主机中curl server:9090
# 查看istio-jeager:16686
```