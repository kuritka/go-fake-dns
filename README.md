# go-fake-dns
unit-testing DNS mock tool

## Usage
The test below set FakeDNS, run listener on port `8853` and run test against it.

```go
import (
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func TestMock(t *testing.T){
	NewDNSMock(t,DNSMockSettings{
		Port:            8853,
		EdgeDNSZoneFQDN: "example.com.",
		DNSZoneFQDN:     "cloud.example.com.",
	}).
		AddNSRecord("gslb-ns-eu-cloud.example.com").
		AddNSRecord("gslb-ns-uk-cloud.example.com").
		AddTXTRecord("First","Second","Banana").
		AddTXTRecord("White","Red","Purple").
		Start().
		RunTestFunc(func() {
			g := new(dns.Msg)
			g.SetQuestion("cloud.example.com.", dns.TypeTXT)
			a, err := dns.Exchange(g, "localhost:8853")
			require.NoError(t, err)
			for _, A := range a.Answer {
				t.Log(A.String())
			}
	})
}
```
