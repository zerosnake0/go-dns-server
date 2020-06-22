package main

import (
	"flag"
	"log"
	"net"

	"github.com/miekg/dns"
)

var domainsToAddresses = map[string]string{
	"tr240.": "172.18.188.240",
}

var (
	bind      string
	dnsClient dns.Client
	dnsAddr   string
)

type handler struct{}

func (this *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	log.Println("serving request...")
	msg := dns.Msg{}

	q := &r.Question[0]
	switch q.Qtype {
	case dns.TypeA:
		domain := q.Name
		log.Println("domain", domain)
		address, ok := domainsToAddresses[domain]
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
		log.Println("alternative", dnsAddr)
		if dnsAddr == "" {
			msg.SetRcode(&msg, dns.RcodeServerFailure)
		} else {
			r2 := r.Copy()
			m2, _, err := dnsClient.Exchange(r2, dnsAddr)
			if err != nil {
				msg.SetRcode(&msg, dns.RcodeServerFailure)
			} else {
				m2.CopyTo(&msg)
			}
		}
	}
	w.WriteMsg(&msg)
}

func main() {
	flag.StringVar(&bind, "bind", ":53", "bind addr")
	flag.StringVar(&dnsAddr, "addr", "", "backup dns addr")
	flag.Parse()

	srv := &dns.Server{Addr: bind, Net: "udp"}
	srv.Handler = &handler{}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}
