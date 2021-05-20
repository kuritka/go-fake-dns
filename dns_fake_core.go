package exdns

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/miekg/dns"
)


// DNSMock acts as DNS server but returns mock values
type DNSMock struct {
	// ReadinessProbe is the channel that is released when the dns server starts listening
	ReadinessProbe chan interface{}
	settings       DNSFakeSettings
	done           chan interface{}
	records        map[uint16][]dns.RR
	server 			*dns.Server
	t              *testing.T
}

func NewDNSFake(t *testing.T, settings DNSFakeSettings) *DNSMock {
	return &DNSMock{
		settings:       settings,
		done:           make(chan interface{}),
		ReadinessProbe: make(chan interface{}),
		records:        make(map[uint16][]dns.RR),
		server: 		&dns.Server{Addr: fmt.Sprintf("[::]:%v", settings.Port), Net: "udp", TsigSecret: nil, ReusePort: false},
		t:              t,
	}
}

func (m *DNSMock) Start() *DNSMock {
	go func() {
		err := m.listen()
		require.NoError(m.t, err)
	}()
	<-m.ReadinessProbe
	fmt.Printf("fake DNS listening on port %v \n", m.settings.Port)
	return m
}

func (m *DNSMock) RunTestFunc(f func()) {
	defer m.stop()
	f()
}


func (m *DNSMock) listen() (err error) {
	dns.HandleFunc(m.settings.EdgeDNSZoneFQDN, m.handleReflect)
	for e :=  range m.serve(m.done) {
		if e != nil {
			close(m.done)
			err = fmt.Errorf("%s%s", err, e)
		}
	}
	return
}

func (m *DNSMock) stop() {
	err := m.server.Shutdown()
	if err != nil {
		close(m.done)
	}
}

func (m *DNSMock) startReadinessProbe() {
	defer close(m.ReadinessProbe)
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
	fmt.Println("->closing done readiness")
	close(m.done)
}

func (m *DNSMock) serve(done <-chan interface{}) <-chan error {
	errors := make(chan error)
	go func() {
		defer close(errors)
		var err error
		go m.startReadinessProbe()
		if err = m.server.ListenAndServe(); err != nil {
			fmt.Println("listen serve error", err)
			err = fmt.Errorf("Failed to setup the server: %s\n", err.Error())
		}
		select {
		case errors <- err:
		case <-done:
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
