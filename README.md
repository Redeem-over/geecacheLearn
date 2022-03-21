
## 分布式缓存系统GeeCache
### 项目介绍:GeeCache是一种模仿groupcache实现的分布式缓存系统，利用golang语言开发，支持的特性有:
1. 单机缓存和基于 HTTP 的分布式缓存 
2. 最近最少访问(Least Recently Used, LRU) 
3. 缓存策略 使用一致性哈希选择节点，实现负载均衡
4. 使用 Go 锁机制防止缓存击穿

## 测试
```shell
./run.sh
```
