package qparts

import (
	"fmt"
	// "log"
	"math/rand"
	"net"
	"strings"
)

// GetIPv4ForNet0 returns the first IPv4 address of an interface that starts with "net0".
func GetIPv4ForNet0() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if strings.HasPrefix(iface.Name, "lo") {
			addrs, err := iface.Addrs()
			if err != nil {
				return "", err
			}
			for _, addr := range addrs {
				ip, _, err := net.ParseCIDR(addr.String())
				if err != nil {
					continue
				}
				if ipv4 := ip.To4(); ipv4 != nil {
					return ipv4.String(), nil
				}
			}
		}
	}

	return "", fmt.Errorf("no IPv4 address found for interface starting with 'net0'")
}

func NewLocalAddr(localAddr string) string {
	hc := host()
	iaStr := hc.ia.String()

	// get random port number between 50000 and 52000 and check if its is free
	port := 50000 + rand.Intn(2000)
	la := strings.ReplaceAll(localAddr, iaStr+",", "")
	hostPortParts := strings.Split(la, ":")

	/*ip, err := GetIPv4ForNet0()
	if err != nil {
		log.Fatal(err)
	}*/
	Log.Info("Local IP: ", hostPortParts[0])
	return fmt.Sprintf("%s,%s:%d", iaStr, hostPortParts[0], port) // TODO: Local IP

}
