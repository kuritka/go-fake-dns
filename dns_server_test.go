package exdns

import (
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func TestMock(t *testing.T) {
	NewDNSMock(t, DNSMockSettings{
		Port:            8853,
		EdgeDNSZoneFQDN: "example.com.",
		DNSZoneFQDN:     "cloud.example.com.",
	}).
		AddNSRecord("blah.cloud.example.com.","gslb-ns-us-cloud.example.com.").
		AddNSRecord("blah.cloud.example.com.","gslb-ns-uk-cloud.example.com.").
		AddNSRecord("blah.cloud.example.com.","gslb-ns-eu-cloud.example.com.").
		AddTXTRecord("First", "Second", "Banana").
		AddTXTRecord("White", "Red", "Purple").
		AddARecord("blah.cloud.example.com.",net.IPv4(192,168,0,5)).
		Start().
		RunTestFunc(func() {
			g := new(dns.Msg)
			//g.SetQuestion("blah.cloud.example.com.", dns.TypeA)
			g.SetQuestion("blah.cloud.example.com.", dns.TypeNS)
			a, err := dns.Exchange(g, "localhost:8853")
			require.NoError(t, err)
			require.NotEmpty(t, a.Answer)
			for _, A := range a.Answer {
				t.Log(A.String())
			}
		})
}
