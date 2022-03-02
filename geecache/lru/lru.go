package lru

import "container/list"

type Cache struct {
	maxBytes  int64                         //允许使用的最大内存
	usedbytes int64                         //已经使用的内存
	ll        *list.List                    //双向链表
	cache     map[string]*list.Element      //键值对
	OnEvicted func(key string, value Value) //某条记录被移除的回调函数
}
type entry struct {
	key   string //淘汰队首结点时方便在map中删除映射
	value Value
} //结点为任何实现了Value这个接口的任意类型
type Value interface {
	Len() int //用于返回值所占内存的大小
}

func New(maxBytes int64, OnEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: OnEvicted,
	}
}

// Get()第一步是从字典中找到对应的双向链表的节点，第二步，将该节点移动到队尾
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele) //对应节点移动至队尾，这里约定front为队尾
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

//这里的删除，实际上是缓存淘汰。即移除最近最少访问的节点（队首）
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.usedbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

/*
  如果键存在，则更新对应节点的值，并将该节点移到队尾。
  不存在则是新增场景，首先队尾添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系。
  更新 c.usedbytes，如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点。
*/
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.usedbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.usedbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.usedbytes {
		c.RemoveOldest()
	}
}
func (c *Cache) Len() int {
	return c.ll.Len()
}
