package mock_senso

import (
	"context"
	"io"
	"log"
	"net"
)

type MockSenso struct {
	ControlConnectionHandler func(net.Conn)
	DataConnectionHandler    func(net.Conn)
}

// ControlConnectionHandler should implement the expected Senso behaviour on the Control port
func ControlConnectionHandler(conn net.Conn) {
	buffer := make([]byte, 1024)
	for {

		readN, readErr := conn.Read(buffer)

		if readErr != nil {
			if readErr == io.EOF {
				return
			} else {
				log.Println(readErr)
			}
		} else {
			log.Println("control data received:", buffer[:readN])
		}

	}
}

func Default() MockSenso {
	var mSenso MockSenso

	mSenso.ControlConnectionHandler = ControlConnectionHandler

	return mSenso
}

func (mSenso *MockSenso) Start(ctx context.Context) {
	log.Println("mock Senso starting up ... ")
	controlServer(mSenso.ControlConnectionHandler)
}

func controlServer(handler func(net.Conn)) {

	for {
		log.Println("opening control port")
		server, err := net.Listen("tcp", ":55567")
		if err != nil {
			log.Println(err)
			return
		}

		// accept a connection
		conn, err := server.Accept()
		// Close the server (disallow multiple connections)
		server.Close()
		if err != nil {
			log.Println(err)
		} else {
			log.Println("new control connection from: ", conn.RemoteAddr())
			handler(conn)
			log.Println("control connection handler returned. Closing connection.")
			conn.Close()
		}

	}

}
