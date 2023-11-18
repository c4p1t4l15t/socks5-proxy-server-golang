package main
import (
	"log"
	"net"
	"time"
	"bytes"
	"errors"
	"runtime"
	"strconv"
	"encoding/binary"
)
func tunnel (from net.Conn, to net.Conn, finished chan bool) {
	buffer := make ([] byte, 1024)

	for {
		bytesRead, err := from.Read (buffer)
		if err != nil || bytesRead < 1 {
			break
		}

		bytesWrite, err := to.Write (buffer [:bytesRead])
		if err != nil || bytesWrite < 1 {
			break
		}
	}

	finished <- true
}
func handShake (conn net.Conn) error {
	buffer := make ([] byte, 3)

	bytesRead, err := conn.Read (buffer)
	if err != nil || bytesRead < 3 {
		return errors.New ("Handshake read error")
	}

	if bytes.Equal (buffer, [] byte {05, 01, 00}) {
		bytesWrite, err := conn.Write ([] byte {0x05, 0x00})
		if err != nil || bytesWrite < 1 {
			return errors.New ("Handshake write error")
		}
	} else {
		bytesWrite, err := conn.Write ([] byte {0x05, 0xff})
		if err != nil || bytesWrite < 1 {
			return errors.New ("Handshake write error")
		}
	}

	return nil
}
func request (conn net.Conn) (net.Conn, error) {
	buffer := make ([] byte, 10)

	bytesRead, err := conn.Read (buffer)
	if err != nil || bytesRead < 10 {
		return nil, errors.New ("Request read error")
	}

	if bytes.Equal (buffer [:4], [] byte {05, 01, 00, 01}) {

		ipAddress := net.IP (buffer [4:8]).String ()
		port := binary.BigEndian.Uint16 (buffer [8:10])
		hostPort := net.JoinHostPort (ipAddress, strconv.Itoa (int (port)))

		remote, err := net.DialTimeout ("tcp", hostPort, time.Duration (time.Second * 15))
		if err != nil {
			return nil, errors.New ("Request connect error")
		}

		bytesWrite, err := conn.Write (append ([] byte {0x05, 0x00, 0x00, 0x01}, buffer [4:10] ...))
		if err != nil || bytesWrite < 1 {
			return nil, errors.New ("Request write error")
		}

		return remote, nil
	} else {
		return nil, errors.New ("Request format error")
	}
}
func accept (conn net.Conn) {
	defer conn.Close ()

	for {
		ret := handShake (conn);
		if ret != nil {
			return
		}

		remote, ret := request (conn);
		if ret != nil {
			return
		}
		defer remote.Close ()

		finished := make (chan bool)

		go tunnel (conn, remote, finished)
		go tunnel (remote, conn, finished)

		<- finished
		<- finished

		return
	}
}
func server (address string) {
	listen, err := net.Listen ("tcp", address)
	if err != nil {
		log.Fatal (err)
	}
	defer listen.Close ()
	for {
		conn, err := listen.Accept ()
		if err != nil {
			log.Fatal (err)
		}
		go accept (conn)
	}
}
func main () {
	runtime.GOMAXPROCS (runtime.NumCPU ())
	go server (":1080")
	for {
		time.Sleep (time.Second)
	}
}
