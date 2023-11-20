package netsigma

import (
	"context"
	"fmt"
	"io"
	"net"
	db "sigmaos/debug"
	"strings"

	sp "sigmaos/sigmap"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

// Rearrange addrs so that first addr is in the realm as clnt.
func Rearrange(clntnet string, addrs sp.Taddrs) sp.Taddrs {
	if len(addrs) == 1 {
		return addrs
	}
	raddrs := make(sp.Taddrs, len(addrs))
	for i := 0; i < len(addrs); i++ {
		raddrs[i] = addrs[i]
	}
	p := -1
	l := -1
	for i, a := range raddrs {
		if a.Net == clntnet {
			l = i
			break
		}
		if a.Net == sp.ROOTREALM.String() && p < 0 {
			p = i
		}
	}
	if l >= 0 {
		swap(raddrs, l)
	} else if p >= 0 {
		swap(raddrs, p)
	}
	return raddrs
}

func swap(addrs sp.Taddrs, i int) sp.Taddrs {
	v := addrs[0]
	addrs[0] = addrs[i]
	addrs[i] = v
	return addrs
}

func QualifyAddr(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	if host == "::" {
		ip, err := LocalIP()
		if err != nil {
			return "", err
		}
		addr = net.JoinHostPort(ip, port)
	}
	return addr, nil
}

// XXX deduplicate with localIP
func LocalInterface() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.IsLoopback() {
				continue
			}
			if ip.To4() == nil {
				continue
			}
			return i.Name, nil
		}
	}
	return "", fmt.Errorf("localInterface: not found")
}

// adapted from https://gist.github.com/nanmu42/9c8139e15542b3c4a1709cb9e9ac61eb
var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			db.DPrintf(db.ALWAYS, "parse error on %q: %v", cidr, err)
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func localIPs() ([]net.IP, error) {
	var ips []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.IsLoopback() {
				continue
			}
			if ip.To4() == nil {
				continue
			}
			ips = append(ips, ip)
		}
	}
	return ips, nil
}

func PublicIP() (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		db.DPrintf(db.ALWAYS, "Could not load default config: %v", err)
		return "", err
	}

	client := imds.NewFromConfig(cfg)
	// client := imds.New(imds.Options{})

	publicip, err := client.GetMetadata(context.TODO(), &imds.GetMetadataInput{
		Path: "public-ipv4",
	})

	if err == nil {
		defer publicip.Content.Close()
		bytes, _ := io.ReadAll(publicip.Content)
		// log.Printf("ee %v", publ)
		db.DPrintf(db.ALWAYS, "Retrieved public IP: %v\n", string(bytes))
		return string(bytes), nil
	}

	db.DPrintf(db.ALWAYS, "Unable to retrieve the public IP address from the EC2 instance: %s\n", err)

	ips, err := localIPs()
	if err != nil {
		db.DPrintf(db.ALWAYS, "Error retrieving local IPs: %v", err)
	}

	// if we have a local ip in 10.10.x.x (for Cloudlab), prioritize that first
	for _, i := range ips {
		if !(isPrivateIP(i)) {
			db.DPrintf(db.ALWAYS, "%v", i.String())
			return i.String(), nil
		}
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("LocalIP: no IP")
	}

	db.DPrintf(db.ALWAYS, "Available IP: %v", ips[len(ips)-1].String())
	return ips[len(ips)-1].String(), nil
}

func LocalIP() (string, error) {
	ips, err := localIPs()
	if err != nil {
		return "", err
	}

	// if we have a local ip in 10.10.x.x (for Cloudlab), prioritize that first
	for _, i := range ips {
		if strings.HasPrefix(i.String(), "10.10.") {
			return i.String(), nil
		}
		if !strings.HasPrefix(i.String(), "127.") {
			return i.String(), nil
		}
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("LocalIP: no IP")
	}

	return ips[len(ips)-1].String(), nil
}
