package exdns

import (
	"github.com/miekg/dns"
	"net"
)

type DNSFakeSettings struct {
	Port int
	// "udp","tcp"
	protocol        string
	EdgeDNSZoneFQDN string
	DNSZoneFQDN     string
}


func (m *DNSMock) AddTXTRecord(fqdn string, strings ...string) *DNSMock {
	t := &dns.TXT{
		Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
		Txt: strings,
	}
	m.records[dns.TypeTXT] = append(m.records[dns.TypeTXT], t)
	return m
}

func (m *DNSMock) AddNSRecord(fqdn, nsName string) *DNSMock {
	ns := &dns.NS{
		Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 0},
		Ns:  nsName,
	}
	m.records[dns.TypeNS] = append(m.records[dns.TypeNS], ns)
	return m
}

func (m *DNSMock) AddARecord(fqdn string, ip net.IP) *DNSMock {
	rr := &dns.A{
		Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
		A:   ip.To4(),
	}
	m.records[dns.TypeA] = append(m.records[dns.TypeA], rr)
	return m
}

func (m *DNSMock) AddAAAARecord(ip net.IP) *DNSMock {
	rr := &dns.A{
		Hdr: dns.RR_Header{Name: m.settings.DNSZoneFQDN, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
		A:   ip.To16(),
	}
	m.records[dns.TypeAAAA] = append(m.records[dns.TypeAAAA], rr)
	return m
}
