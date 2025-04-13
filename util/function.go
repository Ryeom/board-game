package util

import (
	"math/rand"
	"net"
	"time"
)

func IntContains(l []int, p int) bool {
	for _, v := range l {
		if v == p {
			return true
		}
	}
	return false
}

func RandomNumber(min, max int) int { // cno
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Intn(max-min) + min
}

func GetLocalIP() string {
	addr, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addr {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
