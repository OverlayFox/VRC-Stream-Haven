package main

import "net"

func IsPortFree(port string) bool {
	listener, err := net.Listen("udp", ":"+port)
	if err != nil {
		// Port is not free
		return false
	}

	listener.Close()
	return true
}

func GetLocalIP() (net.IP, error) {
	conn, err := net.Dial("udp", "1.1.1.1:80")
	if err != nil {
		return net.IP{}, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}
