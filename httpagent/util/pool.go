package util

import (
	"config"
	"errors"
	wsnmp "github.com/cdevr/wapsnmp"
	"sync"
	"time"
)

var once sync.Once

var testoid = wsnmp.MustParseOid(".1.3.6.1.2.1.1.2.0")

var errReachMaxconn = errors.New("snmpagent: reach max snmp connections")

type SessionPool struct {
	mLock    sync.RWMutex
	Sessions map[string]*Session //默认一个ip只保存一个session，snmp版本不一致会用最后一个版本session覆盖
}

type Session struct {
	version        wsnmp.SNMPVersion
	lastaccesstime time.Time
	Snmpsess       *wsnmp.WapSNMP
}

var SnmpSession = &SessionPool{Sessions: make(map[string]*Session)}

func init() {
	// 启动定时器清理session池
	once.Do(func() { go SnmpSession.sessCleaner(config.Maxlifetime) })
}

// 限定最大连接数
func (sesspool *SessionPool) GetSession(ip, community string, version wsnmp.SNMPVersion, tt time.Duration, rt int) (*wsnmp.WapSNMP, error) {
	var snmpsess *wsnmp.WapSNMP
	var err error
	snmpSess, ok := sesspool.Sessions[ip]
	if !ok || snmpSess.version != version {
		if len(sesspool.Sessions) >= config.Maxsesspool {
			return nil, errReachMaxconn
		}
		snmpsess, err = newsess(ip, community, version, tt, rt)
	}
	if ok {
		snmpSess.lastaccesstime = time.Now()
		snmpsess = snmpSess.Snmpsess
	}

	if err == nil && !ok {
		// 小于最大连接维持数，则缓存连接池
		if len(sesspool.Sessions) < config.Maxsesspool {
			if config.Debug {
				Debug("save snmp session:", ip)
			}
			sesspool.putSess(ip, &Session{version, time.Now(), snmpsess})
		}
	}

	return snmpsess, err
}

func (sesspool *SessionPool) putSess(ip string, s *Session) {
	sesspool.mLock.Lock()
	defer sesspool.mLock.Unlock()
	// 默认可用，超时清理时判断是否可用
	s.lastaccesstime = time.Now()
	sesspool.Sessions[ip] = s
}

func snmptest(s *wsnmp.WapSNMP) bool {
	r, err := s.Get(testoid)
	Debug("snmptest", r)
	if err != nil {
		return false
	}
	return true
}

func newsess(ip, community string, version wsnmp.SNMPVersion, tt time.Duration, rt int) (*wsnmp.WapSNMP, error) {
	snmpsess, err := wsnmp.NewWapSNMP(ip, community, version, tt, rt)
	Info("create snmp session", ip)
	return snmpsess, err
}

func (sesspool *SessionPool) poolRemove(ip string, s *Session) {
	if config.Debug {
		Debug(s.lastaccesstime, time.Now())
		Debug("snmp session expired:", ip)
	}
	s.Snmpsess.Close()
	delete(sesspool.Sessions, ip)
}

func (sesspool *SessionPool) sessCleaner(maxlifetime time.Duration) {
	const d = 10 * time.Second

	t := time.NewTimer(d)
	for {
		select {
		case <-t.C:
		}

		sesspool.mLock.Lock()
		for ip, sess := range sesspool.Sessions {
			if sess.lastaccesstime.Before(time.Now().Add(-maxlifetime)) {
				sesspool.poolRemove(ip, sess)
			} else {
				// 测试sess是否可用，不可用则删除
				if !snmptest(sess.Snmpsess) {
					sesspool.poolRemove(ip, sess)
				}
			}
		}
		sesspool.mLock.Unlock()

		t.Reset(d)
	}
}
