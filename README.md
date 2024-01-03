## 共享剪贴板
一个简单的临时剪贴板工具

### 主要作用： 
+ 提供一个临时的剪贴板，任何人不需要任何验证，可以直接上传文本内容，并在另一个客户端上通过上传拿到的ID获取文本内容
+ 同时支持网页与命令行访问

### Docker Image:
+ https://hub.docker.com/r/saryta/shared-clipboard/tags

### 接口：
+ `GET /api/:id`
  + 根据 ID 获取文本内容
+ `POST /api/new`
  + 创建新临时剪贴板，用于存储文本内容，将会返回 ID
+ `POST /api/:id`
  + 上传时指定 ID，如果此 ID 存在，将会把此 ID 对应的文本内容更新

### 环境变量
+ `CLIPFILE_NUMBER_LIMIT`
  + 非必填项
  + 用于限制临时剪贴板的数量
  + 默认值 1000
+ `CLIPFILE_TIME_LIMIT`
  + 非必填项
  + 用于限制每个临时剪贴板存在的时间
  + 单位为分钟
  + 默认值 15

### 命令行用法示例
```shell
# 上传文本内容
$ curl -X POST -d "demo" localhost:8080/api/new
jqbiy

# 文本内容较长，存在文件中，不需要换行区分
$ curl -X POST -d @test.txt localhost:8080/api/new
pzqsb

# 文本内容较长，存在文件中，需要区分换行
$ curl -X POST --data-binary @test.txt localhost:8080/api/new
jecfv

# 获取文本内容
$ curl localhost:8080/api/jqbiy
demo
```

### 证书
MIT License