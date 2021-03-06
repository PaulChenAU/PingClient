package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	ping "github.com/scientiacoder/PingClient"
)

var usage = `
PingClient Usage:

    go run cmd/ping.go [-n num] [-i interval] [-t timeout] [-c continuous] [--privileged] host

Examples:
    # ping with config yaml file
    go run cmd/ping.go config.yaml

    # ping github continuously
    go run cmd/ping.go -c www.github.com

    # ping github 5 times
    go run cmd/ping.go -n 5 www.github.com

    # ping github 5 times at 500ms intervals
    go run cmd/ping.go -n 5 -i 500ms www.github.com

    # ping github for 10 seconds
    go run cmd/ping.go -t 10s www.github.com

    # Send a privileged raw ICMP ping
    sudo go run cmd/ping.go -privileged www.github.com
`

// run with config yaml file
func runWithYaml() {
	yamlfile := flag.Arg(0)

	pingClients, err := ping.InitWithYAMLFile(yamlfile)
	if err != nil {
		log.Fatalf("%s", err)
		return
	}
	// Listen for Ctrl-C.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			for _, pingClient := range pingClients {
				pingClient.Stop()
			}
		}
	}()

	for _, pingClient := range pingClients {
		pingClient.OnRecv = func(pkt *ping.Packet) {
			fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v ttl=%v\n",
				pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.Ttl)
		}
		pingClient.OnFinish = func(stats []*ping.Statistics) {
			for _, stat := range stats {
				fmt.Printf("\n--- %s %s ping statistics ---\n", stat.URL, stat.IP)
				/*
					for _, pkt := range stat.PacketsInfo {
						fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v ttl=%v\n",
							pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.Ttl)
					}
				*/
				fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
					stat.PacketsSent, stat.PacketsRecv, stat.PacketLoss)
				fmt.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
					stat.MinRtt, stat.AvgRtt, stat.MaxRtt, stat.StdDevRtt)
			}
		}
	}
	for _, pingClient := range pingClients {
		for i := range pingClient.IPs {
			ipStr := pingClient.IPs[i].IP.String()
			if url, ok := pingClient.IPToURL[ipStr]; ok {
				fmt.Printf("PING %s %s:\n", url, pingClient.IPs[i].IP.String())
			} else {
				fmt.Printf("PING %s:\n", ipStr)
			}
		}
		err := pingClient.Run()
		if err != nil {
			log.Fatalf("%s", err)
			return
		}
	}
}

// run with cmd flags
func runWithCmd(timeout *time.Duration, interval *time.Duration, num *int,
	continuous *bool, privileged *bool) {
	var err error
	pingClient := ping.New()
	for i := 0; i < flag.NArg(); i++ {
		err = pingClient.Add(flag.Arg(i))
	}
	if err != nil {
		log.Fatalf("%s", err)
		return
	}

	// Listen for Ctrl-C.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			pingClient.Stop()
		}
	}()

	pingClient.OnRecv = func(pkt *ping.Packet) {
		fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v ttl=%v\n",
			pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.Ttl)

	}
	pingClient.OnFinish = func(stats []*ping.Statistics) {
		for _, stat := range stats {
			fmt.Printf("\n--- %s %s ping statistics ---\n", stat.URL, stat.IP)
			/*
				for _, pkt := range stat.PacketsInfo {
					fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v ttl=%v\n",
						pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.Ttl)
				}
			*/
			fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
				stat.PacketsSent, stat.PacketsRecv, stat.PacketLoss)
			fmt.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
				stat.MinRtt, stat.AvgRtt, stat.MaxRtt, stat.StdDevRtt)
		}
	}
	pingClient.Timeout = *timeout
	pingClient.Interval = *interval
	pingClient.Num = *num
	pingClient.Continuous = *continuous
	pingClient.SetPrivileged(*privileged)
	for i := range pingClient.IPs {
		ipStr := pingClient.IPs[i].IP.String()
		if url, ok := pingClient.IPToURL[ipStr]; ok {
			fmt.Printf("PING %s %s:\n", url, pingClient.IPs[i].IP.String())
		} else {
			fmt.Printf("PING %s:\n", ipStr)
		}
	}

	err = pingClient.Run()
	if err != nil {
		log.Fatalf("%s", err)
		return
	}
}

func main() {
	timeout := flag.Duration("t", 5*time.Second, "")
	interval := flag.Duration("i", 1*time.Second, "")
	num := flag.Int("n", 5, "")
	continuous := flag.Bool("c", false, "")
	privileged := flag.Bool("privileged", false, "")

	flag.Usage = func() {
		fmt.Print(usage)
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	if host := flag.Arg(0); strings.HasSuffix(host, ".yaml") || strings.HasSuffix(host, ".yml") {
		runWithYaml()
	} else {
		runWithCmd(timeout, interval, num, continuous, privileged)
	}
}
