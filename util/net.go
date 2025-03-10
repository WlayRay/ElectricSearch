package util

import (
	"errors"
	"net"
	"os"
)

// 获取本机网卡IP(内网ip)
func GetLocalIP() (ipv4 string, err error) {

	if os.Getenv("DOCKER_ENVIRONMENT") != "" {
		return getDockerIP()
	}

	var (
		addrs   []net.Addr
		addr    net.Addr
		ipNet   *net.IPNet // IP地址
		isIpNet bool
	)
	// 获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}

	for _, addr = range addrs {
		// 这个网络地址是IP地址: ipv4, ipv6
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet {
			// 取第一个非lo的网卡IP
			if !ipNet.IP.IsLoopback() {
				if ipNet.IP.IsPrivate() { // 取内网地址
					// 跳过IPV6
					if ipNet.IP.To4() != nil {
						ipv4 = ipNet.IP.String()
						return
					}
				}
			}
		}
	}

	err = errors.New("ERR_NO_LOCAL_IP_FOUND")
	return
}

// 获取Docker网络接口的IP
func getDockerIP() (ipv4 string, err error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// 跳过回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, addErr := iface.Addrs()
		if addErr != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, isIpNet := addr.(*net.IPNet); isIpNet {
				// 确保是全局单播地址且为IPv4
				if !ipNet.IP.IsLoopback() && ipNet.IP.IsGlobalUnicast() && ipNet.IP.To4() != nil {
					ipv4 = ipNet.IP.String()
					Log.Printf("Found Docker IP: %s", ipv4)
					return ipv4, nil
				}
			}
		}
	}

	err = errors.New("ERR_NO_LOCAL_IP_FOUND")
	return
}
