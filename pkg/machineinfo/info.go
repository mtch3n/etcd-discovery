package machineinfo

import (
	"net"
	"os"
)

type IpAutoDetectMode int

const (
	IpAutoDetectModeDocker IpAutoDetectMode = iota
)

// ServiceDiscoveryInfoFromEnv returns a Discovery struct with the service name, ip, hostname and port
// populated from the environment variables SERVICE_NAME, SERVICE_PORT and the hostname of the machine.
func ServiceDiscoveryInfoFromEnv() *Discovery {
	return &Discovery{
		Name:     nameFromEnv(),
		IP:       addr(IpAutoDetectModeDocker),
		Hostname: hostname(),
		Port:     portFromEnv(),
	}
}

func ServiceDiscoveryInfo(name, ip, port string) *Discovery {
	return &Discovery{
		Name:     name,
		IP:       ip,
		Hostname: hostname(),
		Port:     port,
	}
}

func addr(mode IpAutoDetectMode) string {
	switch mode {
	case IpAutoDetectModeDocker:
		return readFromNslookup()
	default:
		return ""
	}
}

func readFromNslookup() string {
	ips, err := net.LookupIP(hostname())
	if err != nil {
		return ""
	}

	return ips[0].String()
}

func nameFromEnv() string {
	name := os.Getenv("SERVICE_NAME")
	if name == "" {
		return "unknown"
	}
	return name
}

func portFromEnv() string {
	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		return "8080"
	}
	return port

}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "err_unknown"
	}
	return name
}
