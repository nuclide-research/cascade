# cascade

DNS/IP recon cascade. Seed one value — domain, IP, email, or MAC — and every tool whose inputs are satisfied fires automatically, feeding its outputs into the next layer.

```
go install github.com/nuclide-research/cascade@latest
```

## Usage

```
cascade <target> [-j output.json]
```

```
cascade cloudflare.com
cascade 104.16.133.229
cascade user@example.com
cascade 00:50:56:ab:cd:ef
cascade cloudflare.com -j results.json
```

## Tools

| Tool | Input |
|------|-------|
| DNS Record Lookup | domain |
| MX Lookup | domain |
| NS Lookup | domain |
| SOA Lookup | domain |
| SPF Lookup | domain |
| DMARC Lookup | domain |
| DNSSEC Lookup | domain |
| DNS Propagation Checker | domain |
| Whois Lookup | domain |
| HTTP Headers | domain |
| IP History | domain |
| Chinese Firewall Test | domain |
| Iranian Firewall Test | domain |
| Traceroute | domain |
| Find Shared DNS Servers | ns_host (from NS Lookup) |
| Reverse Whois Lookup | registrant_email (from Whois) |
| Spam Database Lookup | mx_host (from MX Lookup) |
| Reverse DNS Lookup | ipv4 |
| IP Location Finder | ipv4 |
| Abuse Contact Lookup | ipv4 |
| Port Scanner | ipv4 |
| Ping | ipv4 |
| Reverse IP Lookup | ipv4 |
| Email Blacklist Check | ipv4 |
| Proxy Checker | ipv4 |
| IP to Hostname | ipv6 |
| Hostname to IP | hostname (from Reverse DNS) |
| ASN Lookup | asn (from IP Location) |
| Reverse Whois Lookup | org (from Whois) |
| Free Email Lookup | email |
| MAC Address Lookup | mac |

No API keys required.
