# go-fake-dns
unit-testing DNS mock tool

## Usage
The test below set FakeDNS, run listener on port `8853` and run test against it.

```go
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


//=== RUN   TestMock
//fake DNS listening on port 8853
//dns_server_test.go:32: blah.cloud.example.com.	0	IN	NS	gslb-ns-us-cloud.example.com.
//dns_server_test.go:32: blah.cloud.example.com.	0	IN	NS	gslb-ns-uk-cloud.example.com.
//dns_server_test.go:32: blah.cloud.example.com.	0	IN	NS	gslb-ns-eu-cloud.example.com.
//--- PASS: TestMock (0.00s)
```
