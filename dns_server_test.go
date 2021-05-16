package exdns

import (
	"fmt"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func TestMock(t *testing.T){
	m := NewDNSMock(DNSMockSettings{
		Port:            8853,
		EdgeDNSZoneFQDN: "example.com.",
		DNSZoneFQDN:     "cloud.example.com.",
	})
	defer m.Stop()
	m.AddNSRecord("gslb-ns-eu-cloud.example.com")
	m.AddNSRecord("gslb-ns-uk-cloud.example.com")
	m.AddTXTRecord("First","Second","Banana")
	m.AddTXTRecord("White","Red","Purple")
	go func() {
		fmt.Println("listening on port 8853")
		err := m.Listen()
		require.NoError(t, err)
	}()
	<- m.ReadinessProbe
	g := new(dns.Msg)
	g.SetQuestion("cloud.example.com.", dns.TypeA)
	a, err := dns.Exchange(g, "localhost:8853")
	require.NoError(t, err)
	for _, A := range a.Answer {
		t.Log(A.String())
	}
}