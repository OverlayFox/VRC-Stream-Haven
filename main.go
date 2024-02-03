package main

import (
	"fmt"
	"net"
	"io"

	"github.com/oschwald/geoip2-golang"
	"github.com/sirupsen/logrus"
)

var logger logrus.Logger
var ipDb geoip2.Reader

func init() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	db, err := geoip2.Open("path/to/GeoLite2-Country.mmdb")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Could not load GeoLite Database")
		return
	}
	defer db.Close()
}

func handleConnection(client net.Conn, targetAddr string) {
	defer client.Close()

	target, err := net.Dial("tcp", targetAddr)
	if err != nil {
		fmt.Println("Error connecting to target:", err)
		return
	}
	defer target.Close()

	// Perform bidirectional copying
	go func() {
		_, err := io.Copy(target, client)
		if err != nil {
			fmt.Println("Error copying from client to target:", err)
		}
	}()

	_, err = io.Copy(client, target)
	if err != nil {
		fmt.Println("Error copying from target to client:", err)
	}
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Couldn't listen on port 8080/tcp")
		return
	}
	defer listener.Close()

	logger.Info("Started listening on Port 8080/tcp")

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err,
			}).Error("Couldn't accept incoming connection")
			continue
		}

		record, err := ipDb.Country(net.ParseIP(conn.RemoteAddr().String()))
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err,
			}).Error("Could not locate IP-Address of incoming connection")
			continue
		}

		fmt.Printf("Connection is coming from %s\n", record.Country.IsoCode)
		conn.Close()
	}
}
