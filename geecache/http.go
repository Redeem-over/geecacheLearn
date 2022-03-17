package geecache

import (
	"fmt"
	"geecache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

var _ PeerGetter = (*httpGetter)(nil)

//HTTPTool包含三个方法:log(),ServerHTTP(),Set(),PickPeer()
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	self        string //用来记录自己的地址，包括主机名/IP 和端口。
	basePath    string //作为节点间通讯地址的前缀，默认是 /_geecache/
	mu          sync.Mutex
	peers       *consistenthash.Map //利用一致性哈希选择其他结点
	httpGetters map[string]*httpGetter
} //http服务端

type httpGetter struct {
	baseURL string
} //http客户端

func NewHttpTool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

//用于打印日志
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

/*
	实现ServeHTTP方法的接口即实现了Handler接口
	我们约定访问路径格式为 /<basepath>/<groupname>/<key>
	再使用 group.Get(key) 获取缓存数据
	最终使用 w.Write() 将缓存值作为 httpResponse 的 body 返回。
*/
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)                          //打印日志
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2) //把key分割出来
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
	}
	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// 用于注册其他分布式结点及其httpGetter接口实例化
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

//  用于返回用于远程请求的节点的PeerGetter
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf("%v%v/%v", h.baseURL, url.QueryEscape(group), url.QueryEscape(key)) //对url进行编码
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server return %v", res.StatusCode)
	}
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body:%v", err)
	}
	return bytes, nil
}
