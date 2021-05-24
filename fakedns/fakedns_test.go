package fakedns

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

const (
	server = "localhost"
	port   = 7053
)

var testSettings = FakeDNSSettings{
	FakeDNSPort:     port,
	EdgeDNSZoneFQDN: "example.com.",
	DNSZoneFQDN:     "cloud.example.com.",
}

func TestFakeDNSMultipleTXTRecords(t *testing.T) {
	NewFakeDNS(testSettings).
		AddTXTRecord("heartbeat-us.cloud.example.com.", "1").
		AddTXTRecord("heartbeat-uk.cloud.example.com.", "2").
		AddTXTRecord("heartbeat-eu.cloud.example.com.", "0", "6", "8").
		Start().
		RunTestFunc(func() {
			g := new(dns.Msg)
			g.SetQuestion("ip.blah.cloud.example.com.", dns.TypeTXT)
			a, err := dns.Exchange(g, fmt.Sprintf("%s:%v", server, port))
			require.NoError(t, err)
			require.Empty(t, a.Answer)

			g = new(dns.Msg)
			g.SetQuestion("heartbeat-uk.cloud.example.com.", dns.TypeTXT)
			a, err = dns.Exchange(g, fmt.Sprintf("%s:%v", server, port))
			require.NoError(t, err)
			require.Len(t, a.Answer, 1)
			require.Len(t, a.Answer[0].(*dns.TXT).Txt, 1)
			require.Equal(t, "2", a.Answer[0].(*dns.TXT).Txt[0])

			g = new(dns.Msg)
			g.SetQuestion("heartbeat-eu.cloud.example.com.", dns.TypeTXT)
			a, err = dns.Exchange(g, fmt.Sprintf("%s:%v", server, port))
			require.NoError(t, err)
			require.Len(t, a.Answer, 1)
			require.Len(t, a.Answer[0].(*dns.TXT).Txt, 3)
			require.Equal(t, "0", a.Answer[0].(*dns.TXT).Txt[0])
			require.Equal(t, "6", a.Answer[0].(*dns.TXT).Txt[1])
			require.Equal(t, "8", a.Answer[0].(*dns.TXT).Txt[2])
		}).RequireNoError(t)
}

func TestFakeDNS(t *testing.T) {
	NewFakeDNS(testSettings).
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
			a, err := dns.Exchange(g, fmt.Sprintf("%s:%v", server, port))
			require.NoError(t, err)
			require.NotEmpty(t, a.Answer)
		}).RequireNoError(t)
}

// TestFakeN runs DNSFake several 10 times
func TestFakeDNSRepeatable(t *testing.T) {
	for i := 1; i < 10; i++ {
		NewFakeDNS(testSettings).
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
			}).RequireNoError(t)
	}
}

func TestFakeDNSPortIsAlreadyInUse(t *testing.T) {
	s := &dns.Server{Addr: fmt.Sprintf("[::]:%v", port), Net: "udp", TsigSecret: nil, ReusePort: false}
	defer func() { _ = s.Shutdown() }()
	go func() { _ = s.ListenAndServe() }()
	time.Sleep(100 * time.Millisecond)
	err := NewFakeDNS(testSettings).
		Start().
		RunTestFunc(func() {
			require.NoError(t, errors.New("this code will not be executed"))
		}).Error
	require.Error(t, err)
}
