package proxy

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/e1732a364fed/v2ray_simple/netLayer"
	"github.com/e1732a364fed/v2ray_simple/utils"
	"go.uber.org/zap"
)

var (
	serverCreatorMap = make(map[string]ServerCreator)
	clientCreatorMap = map[string]ClientCreator{
		DirectName: DirectCreator{},
		RejectName: RejectCreator{},
	}
)

func PrintAllServerNames() {
	fmt.Printf("===============================\nSupported Proxy Listen protocols:\n")
	for _, v := range utils.GetMapSortedKeySlice(serverCreatorMap) {
		fmt.Print(v)
		fmt.Print("\n")
	}
}

func PrintAllClientNames() {
	fmt.Printf("===============================\nSupported Proxy Dial protocols:\n")

	for _, v := range utils.GetMapSortedKeySlice(clientCreatorMap) {
		fmt.Print(v)
		fmt.Print("\n")
	}
}

//可通过两种配置方式来初始化。
type ClientCreator interface {
	NewClient(*DialConf) (Client, error)
	NewClientFromURL(url *url.URL) (Client, error)
}

//可通过两种配置方式来初始化。
type ServerCreator interface {
	NewServer(*ListenConf) (Server, error)
	NewServerFromURL(url *url.URL) (Server, error)
}

// 规定，每个 实现Client的包必须使用本函数进行注册。
// direct 和 reject 统一使用本包提供的方法, 自定义协议不得覆盖 direct 和 reject。
func RegisterClient(name string, c ClientCreator) {
	switch name {
	case DirectName, RejectName:
		return
	}
	clientCreatorMap[name] = c

}

// 规定，每个 实现 Server 的包必须使用本函数进行注册
func RegisterServer(name string, c ServerCreator) {
	serverCreatorMap[name] = c
}

func NewClient(dc *DialConf) (Client, error) {
	protocol := dc.Protocol
	creator, ok := clientCreatorMap[protocol]
	if ok {
		c, e := creator.NewClient(dc)
		if e != nil {
			return nil, e
		}
		e = configCommonForClient(c, dc)
		if e != nil {
			return nil, e
		}
		if dc.TLS {
			c.SetUseTLS()
			e = prepareTLS_forClient(c, dc)
			return c, e
		}

		return c, nil
	} else {
		realScheme := strings.TrimSuffix(protocol, "s")
		creator, ok = clientCreatorMap[realScheme]
		if ok {
			c, err := creator.NewClient(dc)
			if err != nil {
				return c, err
			}
			err = configCommonForClient(c, dc)
			if err != nil {
				return nil, err
			}
			c.SetUseTLS()
			err = prepareTLS_forClient(c, dc)
			return c, err

		}
	}
	return nil, utils.ErrInErr{ErrDesc: "unknown client protocol ", Data: protocol}

}

// ClientFromURL calls the registered creator to create client. The returned bool is true if has err.
func ClientFromURL(s string) (Client, bool, utils.ErrInErr) {
	u, err := url.Parse(s)
	if err != nil {

		return nil, true, utils.ErrInErr{ErrDesc: "can not parse client url", ErrDetail: err, Data: s}
	}

	schemeName := strings.ToLower(u.Scheme)

	creator, ok := clientCreatorMap[schemeName]
	if ok {
		c, e := creator.NewClientFromURL(u)
		if e != nil {
			return nil, true, utils.ErrInErr{ErrDesc: "creator.NewClientFromURL err", ErrDetail: e}
		}
		configCommonByURL(c, u)
		return c, false, utils.ErrInErr{}
	} else {

		//尝试判断是否套tls, 比如vlesss实际上是vless+tls，https实际上是http+tls

		realScheme := strings.TrimSuffix(schemeName, "s")
		creator, ok = clientCreatorMap[realScheme]
		if ok {
			c, err := creator.NewClientFromURL(u)
			if err != nil {
				return nil, true, utils.ErrInErr{ErrDesc: "creator.NewClientFromURL err", ErrDetail: err}
			}
			configCommonByURL(c, u)

			c.SetUseTLS()
			prepareTLS_forProxyCommon_withURL(u, true, c)

			return c, false, utils.ErrInErr{}

		}

	}

	return nil, false, utils.ErrInErr{ErrDesc: "unknown client protocol ", Data: u.Scheme}
}

func NewServer(lc *ListenConf) (Server, error) {
	protocol := lc.Protocol
	creator, ok := serverCreatorMap[protocol]
	if ok {
		ser, err := creator.NewServer(lc)
		if err != nil {
			return nil, err
		}
		err = configCommonForServer(ser, lc)
		if err != nil {
			return nil, err
		}

		if lc.TLS {
			ser.SetUseTLS()
			err = prepareTLS_forServer(ser, lc)
			if err != nil {
				return nil, utils.ErrInErr{ErrDesc: "prepareTLS failed", ErrDetail: err}

			}
			return ser, nil
		}

		return ser, nil
	} else {
		realScheme := strings.TrimSuffix(protocol, "s")
		creator, ok = serverCreatorMap[realScheme]
		if ok {
			ser, err := creator.NewServer(lc)
			if err != nil {
				return nil, err
			}
			err = configCommonForServer(ser, lc)
			if err != nil {
				return nil, err
			}

			ser.SetUseTLS()
			err = prepareTLS_forServer(ser, lc)
			if err != nil {
				return nil, utils.ErrInErr{ErrDesc: "prepareTLS failed", ErrDetail: err}

			}
			return ser, nil

		}
	}

	return nil, utils.ErrInErr{ErrDesc: "unknown server protocol ", Data: protocol}
}

// ServerFromURL calls the registered creator to create proxy servers.
func ServerFromURL(s string) (Server, bool, utils.ErrInErr) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, true, utils.ErrInErr{
			ErrDesc:   "can not parse server url ",
			ErrDetail: err,
			Data:      s,
		}
	}

	schemeName := strings.ToLower(u.Scheme)
	creator, ok := serverCreatorMap[schemeName]
	if ok {
		ser, err := creator.NewServerFromURL(u)
		if err != nil {
			return nil, true, utils.ErrInErr{
				ErrDesc:   "creator.NewServerFromURL err ",
				ErrDetail: err,
			}
		}
		configCommonURLQueryForServer(ser, u)

		return ser, false, utils.ErrInErr{}
	} else {
		realScheme := strings.TrimSuffix(schemeName, "s")
		creator, ok = serverCreatorMap[realScheme]
		if ok {
			server, err := creator.NewServerFromURL(u)
			if err != nil {
				return nil, true, utils.ErrInErr{
					ErrDesc:   "creator.NewServerFromURL err ",
					ErrDetail: err,
				}
			}
			configCommonURLQueryForServer(server, u)

			server.SetUseTLS()
			prepareTLS_forProxyCommon_withURL(u, false, server)
			return server, false, utils.ErrInErr{}

		}
	}

	return nil, true, utils.ErrInErr{ErrDesc: "unknown server protocol ", Data: u.Scheme}
}

//setTag, setCantRoute, call configCommonByURL
func configCommonURLQueryForServer(ser ProxyCommon, u *url.URL) {
	nr := false
	q := u.Query()
	if q.Get("noroute") != "" {
		nr = true
	}
	configCommonByURL(ser, u)

	serc := ser.getCommon()
	if serc == nil {
		return
	}
	serc.cantRoute = nr
	serc.Tag = u.Fragment

	fallbackStr := q.Get("fallback")

	if fallbackStr != "" {
		fa, err := netLayer.NewAddr(fallbackStr)

		if err != nil {
			if utils.ZapLogger != nil {
				utils.ZapLogger.Fatal("configCommonURLQueryForServer failed", zap.String("invalid fallback", fallbackStr))
			} else {
				log.Fatalf("invalid fallback %s\n", fallbackStr)

			}
		}

		serc.FallbackAddr = &fa
	}
}

//SetAddrStr
func configCommonByURL(ser ProxyCommon, u *url.URL) {
	if u.Scheme != DirectName {
		ser.SetAddrStr(u.Host) //若不给出port，那就只有host名，这样不好，我们 默认 配置里肯定给了port

	}
}

//SetAddrStr,  ConfigCommon
func configCommonForClient(cli ProxyCommon, dc *DialConf) error {
	if cli.Name() != DirectName {
		cli.SetAddrStr(dc.GetAddrStrForListenOrDial())
	}

	clic := cli.getCommon()
	if clic == nil {
		return nil
	}

	clic.dialConf = dc

	clic.ConfigCommon(&dc.CommonConf)

	return nil
}

//SetAddrStr, setCantRoute,setFallback, ConfigCommon
func configCommonForServer(ser ProxyCommon, lc *ListenConf) error {
	ser.SetAddrStr(lc.GetAddrStrForListenOrDial())
	serc := ser.getCommon()
	if serc == nil {
		return nil
	}
	serc.listenConf = lc
	serc.cantRoute = lc.NoRoute

	serc.ConfigCommon(&lc.CommonConf)

	if fallbackThing := lc.Fallback; fallbackThing != nil {
		fa, err := netLayer.NewAddrFromAny(fallbackThing)

		if err != nil {
			return utils.ErrInErr{ErrDesc: "configCommonURLQueryForServer failed", Data: fallbackThing}

		}

		serc.FallbackAddr = &fa
	}

	return nil
}
