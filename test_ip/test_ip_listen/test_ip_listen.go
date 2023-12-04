package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

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
			log.Printf("parse error on %q: %v", cidr, err)
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
		log.Printf("Could not load default config: %v", err)
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
		log.Printf("Retrieved public IP: %v\n", string(bytes))
		return string(bytes), nil
	}

	log.Printf("Unable to retrieve the public IP address from the EC2 instance: %s\n", err)

	ips, err := localIPs()
	if err != nil {
		log.Printf("Error retrieving local IPs: %v", err)
	}

	// if we have a local ip in 10.10.x.x (for Cloudlab), prioritize that first
	for _, i := range ips {
		if !(isPrivateIP(i)) {
			log.Printf("%v", i.String())
			return i.String(), nil
		}
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("LocalIP: no IP")
	}

	log.Printf("Available IP: %v", ips[len(ips)-1].String())
	return ips[len(ips)-1].String(), nil
}

func LocalIP() (string, error) {
	return PublicIP()
}

func LocalIP1() (string, error) {
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

func QualifyAddr(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	log.Printf("addr: %v, host: %v, port: %v", addr, host, port)
	if err != nil {
		return "", err
	}
	if host == "::" {
		ip, err := LocalIP1()
		if err != nil {
			return "", err
		}
		addr = net.JoinHostPort(ip, port)
	}
	return addr, nil
}

func main() {
	ips, _ := localIPs()
	log.Printf("ips: %v", ips)
	pub, _ := PublicIP()
	log.Printf("PublicIP: %v", pub)
	loc, _ := LocalIP1()
	log.Printf("LocalIP: %v", loc)

	l, err := net.Listen("tcp", loc+":0")
	if err != nil {
		log.Printf("err %v", err)
	}

	log.Printf("l.MyAddr(): %v", l.Addr().String())
	qual, err := QualifyAddr(l.Addr().String())
	if err != nil {
		log.Printf("Error qualifying addr: err %v", err)
	}

	log.Printf("Qualified addr: %v", qual)
	for {
		// Listen for an incoming connection
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		// Handle connections in a new goroutine
		go func(conn net.Conn) {
			buf := make([]byte, 1024)
			len, err := conn.Read(buf)
			if err != nil {
				fmt.Printf("Error reading: %#v\n", err)
				return
			}
			fmt.Printf("Message received: %s\n", string(buf[:len]))

			conn.Write([]byte("Message received.\n"))
			conn.Close()
		}(conn)
	}

}
