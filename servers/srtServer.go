package servers

import (
	"github.com/haivision/srtgo"
	"log"
)

func initialiseSocket() *srtgo.SrtSocket {
	options := make(map[string]string)
	options["transtype"] = "live"
	options["passphrase"] = "test"
	options["pbkeylen"] = "32"
	options["latency"] = "420"

	return srtgo.NewSrtSocket("0.0.0.0", 8090, options)
}

func startListener(socket *srtgo.SrtSocket) {
	for {
		err := socket.Listen(3)
		if err != nil {
			continue
		}

		conn, _, err := socket.Accept()
		if err != nil {
			log.Printf("Error accepting SRT connection: %s", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn *srtgo.SrtSocket) {

}
