package main

import (
	"io"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/pkg/errors"
)

func newMemlogStream(serverSocket, name string) (stdout, stderr io.WriteCloser, err error) {
	localStdoutLog, remoteStdoutFd := getLogFileSocketPair()
	localStderrLog, remoteStderrFd := getLogFileSocketPair()

	var outSocket int
	if outSocket, err = syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0); err != nil {
		log.Fatal("Unable to create socket: ", err)
	}

	var outFile net.Conn
	if outFile, err = net.FileConn(os.NewFile(uintptr(outSocket), "")); err != nil {
		return nil, nil, err
	}

	var conn *net.UnixConn
	var ok bool
	if conn, ok = outFile.(*net.UnixConn); !ok {
		return nil, nil, errors.New("error making unix conn")
	}

	raddr := net.UnixAddr{Name: serverSocket, Net: "unixgram"}

	if err = sendFD(conn, &raddr, name+".stdout", remoteStdoutFd); err != nil {
		return nil, nil, errors.Wrap(err, "fd stdout send failed")
	}

	if err = sendFD(conn, &raddr, name+".stderr", remoteStderrFd); err != nil {
		return nil, nil, errors.Wrap(err, "fd stderr send failed")
	}

	return localStdoutLog, localStderrLog, nil
}

func getLogFileSocketPair() (*os.File, int) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}

	localFd := fds[0]
	remoteFd := fds[1]

	localLogFile := os.NewFile(uintptr(localFd), "")
	return localLogFile, remoteFd
}

func sendFD(conn *net.UnixConn, remoteAddr *net.UnixAddr, source string, fd int) error {
	oobs := syscall.UnixRights(fd)
	_, _, err := conn.WriteMsgUnix([]byte(source), oobs, remoteAddr)
	return err
}
