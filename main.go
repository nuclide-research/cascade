package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/nuclide-research/cascade/engine"
	"github.com/nuclide-research/cascade/output"
	"github.com/nuclide-research/cascade/tools"
	"github.com/spf13/cobra"
)

var jsonOut string

func main() {
	root := &cobra.Command{
		Use:   "cascade <target>",
		Short: "DNS/IP reconnaissance cascade — all 31 ViewDNS tools, no API keys",
		Args:  cobra.ExactArgs(1),
		RunE:  run,
	}
	root.Flags().StringVarP(&jsonOut, "json", "j", "", "write JSON results to file (- for stdout)")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	target := strings.TrimSpace(args[0])
	output.Banner(target)

	e := engine.New()
	e.OnResult = output.Result

	// Seed state based on what the target looks like
	seed(e, target)

	// Register all tools
	register(e)

	results := e.Run()

	output.Summary(e.State().Snapshot())

	if jsonOut != "" {
		if err := output.JSON(results, jsonOut); err != nil {
			return fmt.Errorf("json output: %w", err)
		}
	}
	return nil
}

// seed detects the target type and populates initial state.
func seed(e *engine.Engine, target string) {
	// Check if it looks like an IP
	if ip := net.ParseIP(target); ip != nil {
		if ip.To4() != nil {
			e.Seed(engine.KeyIPv4, target)
		} else {
			e.Seed(engine.KeyIPv6, target)
		}
		return
	}

	// Check if it looks like a MAC address
	if _, err := net.ParseMAC(target); err == nil {
		e.Seed(engine.KeyMAC, target)
		return
	}

	// Check if it looks like an email
	if strings.Contains(target, "@") {
		e.Seed(engine.KeyEmail, target)
		// Also seed the domain part
		parts := strings.Split(target, "@")
		if len(parts) == 2 {
			e.Seed(engine.KeyDomain, parts[1])
		}
		return
	}

	// Default: domain
	// Strip scheme if present
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "http://")
	target = strings.Split(target, "/")[0]
	e.Seed(engine.KeyDomain, target)
}

// register wires all tools into the engine.
func register(e *engine.Engine) {
	e.Register(
		// DNS tools
		&tools.DNSRecordLookup{},
		&tools.MXLookup{},
		&tools.NSLookup{},
		&tools.SOALookup{},
		&tools.SPFLookup{},
		&tools.DMARCLookup{},
		&tools.DNSSECLookup{},
		&tools.DNSPropagation{},
		&tools.ReverseDNSLookup{},
		&tools.HostnameToIP{},
		&tools.IPToHostname{},

		// Geo / ASN
		&tools.IPLocationFinder{},
		&tools.ASNLookup{},

		// Whois
		&tools.WhoisLookup{},
		&tools.AbuseContactLookup{},
		&tools.ReverseWhoisLookup{},

		// Network
		&tools.PortScanner{},
		&tools.HTTPHeaders{},
		&tools.Ping{},
		&tools.Traceroute{},
		&tools.ReverseIPLookup{},
		&tools.FindSharedDNSServers{},
		&tools.IPHistory{},
		&tools.ProxyChecker{},

		// Email
		&tools.EmailBlacklistCheck{},
		&tools.SpamDatabaseLookup{},
		&tools.FreeEmailLookup{},

		// Firewall
		&tools.ChineseFirewallTest{},
		&tools.IranianFirewallTest{},

		// MAC
		&tools.MACAddressLookup{},
	)
}
