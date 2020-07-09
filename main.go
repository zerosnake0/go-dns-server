package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
)

type config struct {
	Bind     string            `yaml:"bind"`
	Addr     string            `yaml:"addr"`
	Records  map[string]string `yaml:"records"`
	WildCard map[string]string `yaml:"wildcard"`
}

var (
	cfgFile   string
	mu        sync.RWMutex
	cfg       config
	dnsClient dns.Client
)

type handler struct{}

func (this *handler) findDomainAddr(domain string) (string, bool) {
	address, ok := cfg.Records[domain]
	if ok {
		return address, ok
	}
	for domain != "" {
		address, ok := cfg.WildCard[domain]
		if ok {
			return address, ok
		}
		i := strings.IndexByte(domain, '.')
		if i < 0 {
			break
		}
		domain = domain[i+1:]
	}
	return "", false
}

func (this *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	mu.RLock()
	defer mu.RUnlock()
	log.Println("serving request...")
	msg := dns.Msg{}

	q := &r.Question[0]
	switch q.Qtype {
	case dns.TypeA:
		domain := q.Name
		log.Println("domain", domain)
		address, ok := this.findDomainAddr(domain)
		if ok {
			msg.SetReply(r)
			msg.Authoritative = true
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 3600},
				A:   net.ParseIP(address),
			})
		}
	}
	if !msg.Authoritative {
		log.Println("alternative", cfg.Addr)
		if cfg.Addr == "" {
			msg.SetRcode(&msg, dns.RcodeServerFailure)
		} else {
			r2 := r.Copy()
			m2, _, err := dnsClient.Exchange(r2, cfg.Addr)
			if err != nil {
				msg.SetRcode(&msg, dns.RcodeServerFailure)
			} else {
				m2.CopyTo(&msg)
			}
		}
	}
	w.WriteMsg(&msg)
}

func loadConfig() (cfg config, err error) {
	fp, err := os.Open(cfgFile)
	if err != nil {
		return
	}
	defer fp.Close()
	dec := yaml.NewDecoder(fp)
	dec.SetStrict(true)
	err = dec.Decode(&cfg)
	return
}

func init() {
	flag.StringVar(&cfgFile, "config", "config.yaml", "config file")
	flag.Parse()
	var err error
	cfg, err = loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("watcher event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("config modified:", event.Name)
					newCfg, err := loadConfig()
					if err != nil {
						log.Println("unabled to reload config", err)
						continue
					}
					mu.Lock()
					cfg = newCfg
					mu.Unlock()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("watcher error:", err)
			}
		}
	}()
	err = watcher.Add(cfgFile)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	srv := &dns.Server{Addr: cfg.Bind, Net: "udp"}
	srv.Handler = &handler{}
	log.Println("listening...")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}
