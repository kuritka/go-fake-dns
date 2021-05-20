package exdns

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
)

type DNSFakeSettings struct {
	Port int
	EdgeDNSZoneFQDN string
	DNSZoneFQDN     string
}

// DNSMock acts as DNS server but returns mock values
type DNSMock struct {
	// readinessProbe is the channel that is released when the dns server starts listening
	readinessProbe chan interface{}
	done chan interface{}
	settings       DNSFakeSettings
	records        map[uint16][]dns.RR
	server         *dns.Server
	t              *testing.T
	err 			error
}

func NewDNSFake(t *testing.T, settings DNSFakeSettings) *DNSMock {
	return &DNSMock{
		settings:       settings,
		readinessProbe: make(chan interface{}),
		done: 			make(chan interface{}),
		records:        make(map[uint16][]dns.RR),
		server:         &dns.Server{Addr: fmt.Sprintf("[::]:%v", settings.Port), Net: "udp", TsigSecret: nil, ReusePort: false},
		t:              t,
	}
}

func (m *DNSMock) Start() *DNSMock {
	go func() {
		m.err = m.listen()
	}()
	<-m.readinessProbe
	fmt.Printf("FakeDNS listening on port %v \n", m.settings.Port)
	return m
}

func (m *DNSMock) RunTestFunc(f func()) error {
	if m.err != nil {
		f()
		m.err = m.server.Shutdown()
	}
	return m.err
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

func (m *DNSMock) listen() (err error) {
	dns.HandleFunc(m.settings.EdgeDNSZoneFQDN, m.handleReflect)
	for e := range m.serve() {
		if e != nil {
			err = fmt.Errorf("%s%s", err, e)
		}
	}
	return
}

func (m *DNSMock) startReadinessProbe() {
	defer close(m.readinessProbe)
	for i := 0; i < 5; i++ {
		g := new(dns.Msg)
		host := fmt.Sprintf("localhost:%v", m.settings.Port)
		g.SetQuestion(m.settings.DNSZoneFQDN, dns.TypeA)
		_, err := dns.Exchange(g, host)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		return
	}
	// waiting too long, close listening
	close(m.done)
}

func (m *DNSMock) serve() <-chan error {
	errors := make(chan error)
	go func() {
		defer close(errors)
		var err error
		go m.startReadinessProbe()
		if err = m.server.ListenAndServe(); err != nil {
			errors <- fmt.Errorf("failed to setup the server: %s\n", err.Error())
			close(m.done)
		}
		select {
		case <-m.done:
			return
		}
	}()
	return errors
}

func (m *DNSMock) handleReflect(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Compress = false
	if m.records[r.Question[0].Qtype] != nil {
		for _, rr := range m.records[r.Question[0].Qtype] {
			switch r.Question[0].Qtype {
			case dns.TypeA, dns.TypeAAAA:
				fqdn := strings.Split(rr.String(), "\t")[0]
				if fqdn != r.Question[0].Name {
					continue
				}
			}
			msg.Answer = append(msg.Answer, rr)
			//msg.Extra = append(msg.Extra, rr)
		}
	}
	_ = w.WriteMsg(msg)
}
