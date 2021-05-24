package exdns

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	server = "localhost"
	port   = 7753
)

var testSettings = FakeDNSSettings{
	Port:            port,
	EdgeDNSZoneFQDN: "example.com.",
	DNSZoneFQDN:     "cloud.example.com.",
}

func TestFakeDNS(t *testing.T) {
	err := NewFakeDNS(t, testSettings).
		AddNSRecord("blah.cloud.example.com.", "gslb-ns-us-cloud.example.com.").
		AddNSRecord("blah.cloud.example.com.", "gslb-ns-uk-cloud.example.com.").
		AddNSRecord("blah.cloud.example.com.", "gslb-ns-eu-cloud.example.com.").
		AddTXTRecord("First", "Second", "Banana").
		AddTXTRecord("White", "Red", "Purple").
		AddARecord("ip.blah.cloud.example.com.", net.IPv4(10, 0, 1, 5)).
		Start().
		RunTestFunc(func() {
			g := new(dns.Msg)
			g.SetQuestion("ip.blah.cloud.example.com.", dns.TypeA)
			//g.SetQuestion("blah.cloud.example.com.", dns.TypeNS)
			a, err := dns.Exchange(g, fmt.Sprintf("%s:%v", server, port))
			require.NoError(t, err)
			require.NotEmpty(t, a.Answer)
		})
	assert.NoError(t, err)
}

// TestFakeN runs DNSFake several 10 times
func TestFakeDNSRepeatable(t *testing.T) {
	for i := 1; i < 10; i++ {
		NewFakeDNS(t, testSettings).
			AddARecord("localtargets-roundrobin.cloud.example.com.", net.IPv4(10, 1, 0, 3)).
			AddARecord("localtargets-roundrobin.cloud.example.com.", net.IPv4(10, 1, 0, 2)).
			AddARecord("localtargets-roundrobin.cloud.example.com.", net.IPv4(10, 1, 0, 1)).
			AddTXTRecord("localtargets-heartbeat-us.cloud.example.com.", "5m").
			Start().
			RunTestFunc(func() {
				fmt.Println("FakeDNS test: ", i)
				g := new(dns.Msg)
				g.SetQuestion("localtargets-roundrobin.cloud.example.com.", dns.TypeA)
				// put server under load....
				for i := 0; i <= 20; i++ {
					a, err := dns.Exchange(g, fmt.Sprintf("%s:%v", server, port))
					require.NoError(t, err)
					require.NotEmpty(t, a.Answer)
					require.Equal(t, 3, len(a.Answer))
					time.Sleep(5 * time.Millisecond)
				}
			})
	}
}

func TestFakeDNSPortIsAlreadyInUse(t *testing.T) {
	s := &dns.Server{Addr: fmt.Sprintf("[::]:%v", port), Net: "udp", TsigSecret: nil, ReusePort: false}
	go func() { _ = s.ListenAndServe() }()
	time.Sleep(100 * time.Millisecond)
	err := NewFakeDNS(t, testSettings).
		Start().
		RunTestFunc(func() {
			fmt.Println("doing something...")
		})
	assert.Error(t, err)
}
