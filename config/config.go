package config

import (
	"fmt"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"
)

const cfgfile string = "config/snmpagent.yml"

// system level config interface
var Maxsesspool = 1000
var Maxlifetime time.Duration = 30 * time.Second
var Timeout time.Duration = 2 * time.Second
var Retry int = 1
var Debug bool = false
var Port string = "1216"
var Logdir string = "./"
var Asyncnum int = 100

func init() {
	configmap, err := GetConfig()
	if err != nil {
		log.Println(err)
		return
	}
	// port
	port := GetKey("http.port", configmap)
	if port != "" {
		Port = port
	}
	// timeout
	timeout := GetKey("snmp.timeout", configmap)
	if timeout != "" {
		t, err := strconv.Atoi(timeout)
		if err == nil {
			Timeout = time.Duration(t) * time.Second
		}
	}
	// retry
	retry := GetKey("snmp.retry", configmap)
	if retry != "" {
		r, err := strconv.Atoi(retry)
		if err == nil {
			Retry = r
		}
	}
	// debug
	debug := GetKey("log.debug", configmap)
	if debug == "true" || debug == "yes" {
		Debug = true
	}
	// Maxsesspool
	maxsesspool := GetKey("snmp.maxsesspool", configmap)
	if maxsesspool != "" {
		m, err := strconv.Atoi(maxsesspool)
		if err == nil {
			Maxsesspool = m
		}
	}
	// Maxlifetime
	maxlifetime := GetKey("snmp.maxlifetime", configmap)
	if maxlifetime != "" {
		t, err := strconv.Atoi(maxlifetime)
		if err == nil {
			Maxlifetime = time.Duration(t) * time.Second
		}
	}

	// Asyncnum
	asyncnum := GetKey("snmp.asyncnum", configmap)
	if asyncnum != "" {
		m, err := strconv.Atoi(asyncnum)
		if err == nil {
			Asyncnum = m
		}
	}
	// logdir
	logdir := GetKey("log.logdir", configmap)
	if logdir != "" {
		Logdir = logdir
	}
}

// 需要做动态加载配置文件，放在主调程序做
func GetConfig() (m map[interface{}]interface{}, err error) {
	data, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		return
	}
	//m = make(map[interface{}]interface{})
	err = yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		return
	}
	return
}

func GetKey(key string, cfgmap map[interface{}]interface{}) string {
	keys := strings.Split(key, ".")
	if v, ok := cfgmap[keys[0]]; ok {
		switch t := v.(type) {
		case string:
			return fmt.Sprint(v)
		case interface{}:
			return GetValue(keys[1:], t)
		default:
			return ""
		}
	} else {
		return ""
	}
}

func GetValue(keys []string, intf interface{}) string {
	switch t := intf.(type) {
	case string:
		return fmt.Sprint(t)
	case interface{}:
		// fmt.Println(t)
		if v, ok := t.(map[interface{}]interface{}); ok {
			if len(keys) > 1 {
				return GetValue(keys[1:], v[keys[0]])
			} else {
				if r, ok := v[keys[0]]; ok {
					return fmt.Sprint(r)
				}
				return ""
			}
		} else if v, ok := t.([]interface{}); ok {
			r := ""
			for _, i := range v {
				r = GetValue(keys, i)
				if r != "" {
					break
				}
			}
			return r
		} else if v, ok := t.(string); ok {
			return fmt.Sprint(v)
		} else {
			return ""
		}
	default:
		return ""
	}
	return ""
}
