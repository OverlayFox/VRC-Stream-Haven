package servers

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
