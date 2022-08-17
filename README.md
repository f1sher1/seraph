# quick start

## configuration

### ~~ 1. [db](app/config/database.go)  ~~
### ~~ 2. [rabbitMQ](app/config/mq.go) ~~
### ~~ 3. [engine](app/config/engine.go)&emsp;&emsp;&emsp;&emsp;&emsp;&emsp;*配置本地（宿主机）的IP* ~~
### ~~ 4. [engine-executor](app/config/executor.go)&emsp;&emsp;*配置本地（宿主机）的IP* ~~
### ~~ 5. [nova-url](app/config/plugin.go) ~~
### ~~ 6. [unimq-url](app/config/plugin.go) ~~
### change [config file](config/config.ini)
``` ini
[db]
host='127.0.0.1'
port='3306'
user='root'
passwd='123456'
debug=false
pool_size=10
idle_timeout=3600

[myip]
myip='127.0.0.1'  # 宿主机ip

[rabbitMQ]
host='127.0.0.1:5672'
user='guest'
passwd='guest'
exchange='seraph'
amqp_queues_ttl=1800

[nova]
url='http://10.111.7.41:8774/v2'  # nova url前缀

[uniMQ]
url='http://10.111.17.15:8080/ksyun/mq'
topic_name='openstack_notify'
routing_key=''
app_key='LMvZeq11'
secret_key='uKbN6syEvyfwRsvz'

[log]
dir_path='log'   #容器内的目录，需要-v到宿主机 windows默认目录 c:\\log  linux目录: /var/log
```

## create databases
``` mysql
create database seraph;
```
## create tables
``` shell
go run ./app/db/create_table/main.go
```
## register action
``` shell
go run plugins/cmd/register.go
```
## run services use go run cmd
``` shell
  1. go run ./web/cmd/main.go
  2. go run ./plugins/nova/cmd/main.go
  3. go run ./plugins/standard/cmd/main.go
  4. go run ./app/engine/cmd/main.go
```
# docker

### build image

``` shell
docker build -t seraph:v1 .
```


### 数据库表初始化
```shell
docker run --name create-tables -v [存放config.ini的宿主机目录]:/opt seraph:v1 ./create-tables
```
### 初始化action definition，新增action definition后也要执行。
```shell
docker run --name register -v [存放config.ini的宿主机目录]:/opt seraph:v1 ./register
```

### **config.ini日志目录windows默认C:\\log linux默认/var/log**
### run api
``` shell
docker run -d --name seraph-api -p [宿主机端口]:8080 -v [存放config.ini的宿主机目录]:/opt -v [日志目录]:[config.ini设置的目录] seraph:v1 ./api
```

### run engine
``` shell
docker run -d --name seraph-engine -v [存放config.ini的宿主机目录]:/opt -v [日志目录]:[config.ini设置的目录] --volumes-from seraph-api seraph:v1 ./engine
```

### run plugin-nova
``` shell
docker run -d --name seraph-nova -v [存放config.ini的宿主机目录]:/opt -v [日志目录]:[config.ini设置的目录] --volumes-from seraph-api seraph:v1 ./plugin-nova
```

### run plugin-std
``` shell
docker run -d --name seraph-std -v [存放config.ini的宿主机目录]:/opt -v [日志目录]:[config.ini设置的目录] --volumes-from seraph-api seraph:v1 ./plugin-std
```


### 指令集操作
```shell
进入任意一个容器中执行
./tools -w 工作流ID
然后根据提示操作
如：
WORKFLOW:
  startTime:           2022-05-11 15:39:52.51 +0800 CST
  finishedTime:        2022-05-11 15:40:01.177 +0800 CST
  status:              SUCCESS
  statusInfo:
1. 绘制工作流图
2. 打印运行的task
输入一下操作编码(其他则退出):
```