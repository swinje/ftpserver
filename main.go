package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const (
	status150 = "150 OK"
	status200 = "200 OK"
	status220 = "220 OK"
	status221 = "221 Service closing control connection."
	status226 = "226 OK"
	status230 = "230 User %s logged in."
	status425 = "425 Can't open data connection."
	status426 = "426 Connection closed; transfer aborted."
	status501 = "501 Syntax error in parameters or arguments."
	status502 = "502 Command not implemented."
	status504 = "504 Cammand not implemented for that parameter."
	status550 = "550 File unavailable."
)

var port int

func init() {
	flag.IntVar(&port, "port", 8080, "port number")
	flag.Parse()
}

func main() {

	workDir := "."
	absPath, err := filepath.Abs(".")
	if err != nil {
		log.Fatal(err)
	}

	server := fmt.Sprintf(":%d", port)
	log.Printf("Running FTP server on port %d", port)
	listener, err := net.Listen("tcp", server)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handleConn(conn, &absPath, &workDir)
	}
}

func handleConn(conn net.Conn, absPath *string, workDir *string) {
	defer conn.Close()
	ch := make(chan string)                 // return channel to client
	go clientWriter(conn, ch)               // connect channel to client
	ch <- status220 + " FTP server" // OK

	ip := conn.RemoteAddr().String()
	log.Printf("Client connected: %s", ip)

	var address string
	dataType := 0

	input := bufio.NewScanner(conn)
	for input.Scan() {
		fields := strings.Fields(input.Text())
		if len(fields) == 0 {
			continue
		}
		command, args := fields[0], fields[1:]
		//log.Printf("Command: %s %v", command, args)

		switch command {
		case "USER":
			ch <- fmt.Sprintf(status230, strings.Join(args, " "))
		case "CWD":
			ch <- changeDir(args, workDir, absPath)
		case "LIST": // ls
			list(conn, args, workDir, address, dataType)
		case "PORT":
			address = portAdress(args[0])
			ch <- status200
		case "LPRT": // unclear why client sends
			ch <- status200
		case "QUIT": // close
			ch <- status221
			log.Printf("Client disonnected: %s", ip)
			return
		case "RETR": // get
			getFile(conn, args, workDir, address, dataType)
		case "TYPE":
			setDataType(conn, args, &dataType)
		case "PWD":
			ch <- *absPath
		case "STOR":
			storeFile(conn, args, workDir, address, dataType)
		default:
			ch <- status502 // Not implemented
		}

	}
}

func EOL(dataType int) string {
	if dataType == 0 {
		return "\r\n"
	}
	return "\n"
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg) // NOTE: ignoring network errors
	}
}

func portAdress(val string) string {

	type dataPort struct {
		h1, h2, h3, h4 int // host
		p1, p2         int // port
	}

	var dp dataPort

	fmt.Sscanf(val, "%d,%d,%d,%d,%d,%d",
		&dp.h1, &dp.h2, &dp.h3, &dp.h4, &dp.p1, &dp.p2)
	port := dp.p1<<8 + dp.p2
	address := fmt.Sprintf("%d.%d.%d.%d:%d", dp.h1, dp.h2, dp.h3, dp.h4, port)
	return address

}

func list(conn net.Conn, args []string, workDir *string, address string, dataType int) {
	ch := make(chan string) // return channel to client
	go clientWriter(conn, ch)
	defer close(ch)

	var target string
	if len(args) > 0 {
		target = filepath.Join(".", *workDir, args[0])
	} else {
		target = filepath.Join(".", *workDir)
	}

	files, err := ioutil.ReadDir(target)
	if err != nil {
		log.Print(err)
		ch <- status550
		return
	}

	ch <- status150

	dConn, err := net.Dial("tcp", address)
	if err != nil {
		log.Print(err)
		ch <- status425
		return
	}
	defer dConn.Close()

	for _, file := range files {
		_, err := fmt.Fprint(dConn, file.Name(), EOL(dataType))
		if err != nil {
			log.Print(err)
			ch <- status426
			return
		}
	}
	_, err = fmt.Fprintf(dConn, EOL(dataType))
	if err != nil {
		log.Print(err)
		ch <- status426
		return
	}

	ch <- status226

}

func changeDir(args []string, workdir *string, absPath *string) string {
	if len(args) != 1 {
		return status501
	}

	*workdir = filepath.Join(*workdir, args[0])
	*absPath, _ = filepath.Abs(*workdir)

	_, err := os.Stat(*absPath)
	if err != nil {
		log.Print(err)
		return status550
	}
	return status200
}

func getFile(conn net.Conn, args []string, workDir *string, address string, dataType int) {
	ch := make(chan string) // return channel to client
	go clientWriter(conn, ch)
	defer close(ch)

	if len(args) != 1 {
		ch <- status501
		return
	}

	path := filepath.Join(*workDir, args[0])
	file, err := os.Open(path)
	if err != nil {
		log.Print(err)
		ch <- status550
		return
	}
	ch <- status150

	dConn, err := net.Dial("tcp", address)
	if err != nil {
		log.Print(err)
		ch <- status425
		return
	}
	defer dConn.Close()

	_, err = io.Copy(dConn, file)
	if err != nil {
		log.Print(err)
		ch <- status426
		return
	}
	io.WriteString(dConn, EOL(dataType))
	ch <- status226

}

func storeFile(conn net.Conn, args []string, workDir *string, address string, dataType int) {
	ch := make(chan string) // return channel to client
	go clientWriter(conn, ch)
	defer close(ch)

	if len(args) != 1 {
		ch <- status501
		return
	}

	log.Printf("storing %s", args[0])

	f, err := os.Create(args[0]) // creates a file at current directory
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	ch <- status150

	dConn, err := net.Dial("tcp", address)
	if err != nil {
		log.Print(err)
		ch <- status425
		return
	}
	defer dConn.Close()

	_, err = io.Copy(f, dConn)
	if err != nil {
		log.Print(err)
		ch <- status426
		return
	}
	ch <- status226

}

func setDataType(conn net.Conn, args []string, dataType *int) {
	ch := make(chan string) // return channel to client
	go clientWriter(conn, ch)
	defer close(ch)

	if len(args) == 0 {
		ch <- status501
		return
	}

	switch args[0] {
	case "A":
		*dataType = 0
	case "I": // binary
		*dataType = 1
	default:
		ch <- status504
		return
	}
	ch <- status200
}
