package exdns

import (
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
)

type DNSMockSettings struct {
	Port int
	// "udp","tcp"
	Protocol        string
	EdgeDNSZoneFQDN string
	DNSZoneFQDN     string
}

// DNSMock acts as DNS server but returns mock values
type DNSMock struct {
	// ReadinessProbe is the channel that is released when the dns server starts listening
	ReadinessProbe chan interface{}
	settings       DNSMockSettings
	done           chan interface{}
	records 	   map[uint16][]dns.RR
}

func NewDNSMock(settings DNSMockSettings) *DNSMock {
	return &DNSMock{
		settings:       settings,
		done:           make(chan interface{}),
		ReadinessProbe: make(chan interface{}),
		records: 		make(map[uint16][]dns.RR),
	}
}

func (m *DNSMock) Listen() (err error) {
	dns.HandleFunc(m.settings.EdgeDNSZoneFQDN, m.handleReflect)
	for e := range m.serve(m.done,"udp","tcp") {
		if e != nil {
			close(m.done)
			err = fmt.Errorf("%s%s",err,e)
		}
	}
	return
}

func (m *DNSMock) Stop() {
	defer close(m.done)
}

func (m *DNSMock) AddTXTRecord(msgs ...string){
	t := &dns.TXT{
		Hdr: dns.RR_Header{Name: m.settings.DNSZoneFQDN, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
		Txt: msgs,
	}
	m.records[dns.TypeTXT] = append(m.records[dns.TypeTXT], t)
}

func (m *DNSMock)  AddNSRecord(nsName string){
	ns := &dns.NS{
		Hdr: dns.RR_Header{Name: m.settings.DNSZoneFQDN, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 0},
		Ns:  nsName,
	}
	m.records[dns.TypeNS] = append(m.records[dns.TypeNS], ns)
}

func (m *DNSMock)  AddARecord(ip net.IP){
	rr := &dns.A{
		Hdr: dns.RR_Header{Name: m.settings.DNSZoneFQDN, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
		A:   ip.To4(),
	}
	m.records[dns.TypeA] = append(m.records[dns.TypeA], rr)
}

func (m *DNSMock)  AddAAAARecord(ip net.IP){
	rr := &dns.A{
		Hdr: dns.RR_Header{Name: m.settings.DNSZoneFQDN, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
		A:   ip.To16(),
	}
	m.records[dns.TypeAAAA] = append(m.records[dns.TypeAAAA], rr)
}

func (m *DNSMock) startReadinessProbe() {
	defer close(m.ReadinessProbe)
	for {
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
}

func (m *DNSMock) serve(done <- chan interface{},  protocols ...string) <- chan error {
		errors := make(chan error)
		go func(){
			defer close(errors)
			for _, net := range protocols {
				var err error
				server := &dns.Server{Addr: fmt.Sprintf("[::]:%v", m.settings.Port), Net: net, TsigSecret: nil, ReusePort: false}
				go m.startReadinessProbe()
				if err = server.ListenAndServe(); err != nil {
					err = fmt.Errorf("Failed to setup the %s server: %s\n",net, err.Error())
				}
				select {
				case <- done:
					return
				case errors <- err:
				}
			}
		}()
		return errors
}

func (m *DNSMock) handleReflect(w dns.ResponseWriter, r *dns.Msg) {
	var (
		v4  bool
		a   net.IP
	)
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Compress = false
	if ip, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		a = ip.IP
		v4 = a.To4() != nil
	}
	if ip, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		a = ip.IP
		v4 = a.To4() != nil
	}

	if v4 {
		m.AddARecord(a)
	} else {
		m.AddAAAARecord(a)
	}

	if m.records[r.Question[0].Qtype] != nil {
		for _, rr := range m.records[r.Question[0].Qtype] {
			msg.Answer = append(msg.Answer, rr)
			//msg.Extra = append(msg.Extra, rr)
		}
	}

	if r.IsTsig() != nil {
		if w.TsigStatus() == nil {
			msg.SetTsig(r.Extra[len(r.Extra)-1].(*dns.TSIG).Hdr.Name, dns.HmacMD5, 300, time.Now().Unix())
		} else {
			println("Status", w.TsigStatus().Error())
		}
	}
	// set TC when question is tc.$EdgeDNSZoneFQDN
	if msg.Question[0].Name == fmt.Sprintf("tc.%s",m.settings.EdgeDNSZoneFQDN) {
		msg.Truncated = true
		// send half a message
		buf, _ := msg.Pack()
		w.Write(buf[:len(buf)/2])
		return
	}
	w.WriteMsg(msg)
}
