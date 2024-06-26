package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
)

var (
  socketPath = "/tmp/upgrade.sock"
  filePath = "myfile"
  reader = bufio.NewReader(os.Stdin)
  writeNo = 0

  err error
  openFds []int
  sockFile *os.File
)

func main() {
  fmt.Printf(">>> Started process with PID: [%d]\n", os.Getpid())
  printHelp()

  for {
    fmt.Printf("\ninput> ")
    input, _ := reader.ReadSlice('\n')
    str_input := strings.TrimSuffix(string(input), "\n")

    if str_input == "pid" {
      fmt.Printf("PID is : %d\n", os.Getpid())

    } else if str_input == "openfile" {
      f, err := os.OpenFile(filePath, os.O_RDWR | os.O_APPEND, 0)
      handleError(err)
      fd := int(f.Fd())
      openFds = append(openFds, fd)
      fmt.Printf("Opened file, FD = %v\n", fd)

    } else if str_input == "closefile" {
      if len(openFds) == 0 {
        fmt.Printf("No open files to close\n")
        continue
      }
      // Pop the last file descriptor and close the file
      fd := openFds[len(openFds)-1]
      openFds = openFds[:len(openFds)-1]
      err := syscall.Close(fd) 
      handleError(err)
      fmt.Printf("Closed file, FD = %v\n", fd)

    } else if str_input == "start-sock" {
      err = startSock(socketPath)
      handleError(err)

    } else if str_input == "connect-sock" {
      err = connectSock(socketPath)
      handleError(err)

    } else if str_input == "read-first-fd" {
      err = readFirstFd()
      handleError(err)

    } else if str_input == "write-first-fd" {
      err = writeFirstFd()
      handleError(err)


    } else if str_input == "stop-sock"{
      sockFile.Close()
      fmt.Printf("Closed unix socket connexion\n")

    } else if str_input == "lsof" {
      fmt.Printf("Open file descriptors: %v\n", openFds)

    } else if str_input == "help" {
      printHelp()

    } else if str_input == "" {
      continue

    } else {
      fmt.Printf("Unknown option: %s\n", str_input)
    }
  }
}

// Creates a socket that will be used to transfer open file descriptors of the opening process
func startSock(sockPath string) (error) {
  sockFd, err := syscall.Socket(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
  if err != nil {
    return err
  }
  fmt.Printf("Created socket with fd %v\n", sockFd)

  syscall.Unlink(sockPath)
  sockFile := os.NewFile(uintptr(sockFd), sockPath)

  addr := &syscall.SockaddrUnix{Name: sockPath}
  err = syscall.Bind(sockFd, addr)
  if err != nil {
    return err
  }
  err = syscall.Listen(sockFd, 1)
  if err != nil {
    return err
  }
  fmt.Printf("Started listening on socket file %v\n", sockFile)

  peerFd, _, err := syscall.Accept(sockFd)
  if err != nil {
    return err 
  }
  fmt.Printf("Accepted from client. Client socket peer fd [%v]\n", peerFd)

  rights := syscall.UnixRights(openFds...) // Sending the file descriptors
  err = syscall.Sendmsg(peerFd, nil, rights, nil, 0) 
  if err != nil {
    return err 
  }
  fmt.Printf("Sent message on socket to client peer socket fd [%v]\n", peerFd)

  return nil
}

// Connect to the socket and receive the file descriptors
func connectSock(sockPath string) (error) {
  sockFd, err := syscall.Socket(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
  if err != nil {
    return err
  }
  fmt.Printf("Created socket with fd [%v]\n", sockFd)

  addr := &syscall.SockaddrUnix{Name: sockPath}
  err = syscall.Connect(sockFd, addr)
  if err != nil {
    return err
  }
  fmt.Printf("Connected to socket on path [%v]\n", sockPath)

  oob := make([]byte, syscall.CmsgSpace(4))
  _, oobn, _, _, err := syscall.Recvmsg(sockFd, nil, oob, 0)
  fmt.Printf("Received msg from server on fd %v, length of oob data [%v]\n", sockFd, len(oob))

  messages, err := syscall.ParseSocketControlMessage(oob[:oobn])
  if err != nil {
    return err
  }

  if len(messages) != 1 { 
    return fmt.Errorf("received %d messages\n", len(messages))
  }
  message := messages[0]

  fds, err := syscall.ParseUnixRights(&message)
  if len (fds) == 0 {
    return fmt.Errorf("received 0 fds\n")
  }
  fmt.Printf("Parsed message from server, received fds [%v]\n", fds)
  for _, fd := range fds {
    openFds = append(openFds, fd)
  }
  if err != nil {
    return err
  }

  return err
}

func readFirstFd() error {
  if len(openFds) == 0 {
    return fmt.Errorf("No open file descriptors to read from\n")
  }
  firstFd := openFds[0]
  buf := make([]byte, 1024)
  fmt.Printf("Attempting to read data from file descriptor [%v]\n", firstFd)

  if _, err := syscall.Read(firstFd, buf); err != nil {
    return err 
  }

  fmt.Printf("Read data from first file descriptor [%v]: %s\n", firstFd, string(buf))
  return nil
}

func writeFirstFd() error {
  if len(openFds) == 0 {
    return fmt.Errorf("No open file descriptors to write to\n")
  }
  firstFd := openFds[0]
  payload := fmt.Sprintf("Payload #%d. Process #%d\n", writeNo, os.Getpid())
  fmt.Printf("Attempting to write data to file descriptor [%v]\n", firstFd)

  if _, err := syscall.Write(firstFd, []byte(payload)); err != nil {
    return err
  }

  fmt.Printf("Written data to file descriptor [%v] from process [%d]: %s\n", firstFd, os.Getpid(), payload)
  return nil
}

func handleError(e error) {
  if e != nil {
    panic(e) 
  }
}

func printHelp() {
  fmt.Printf(">>> Available options: pid, openfile, closefile, lsof, start-sock, connect-sock, read-first-fd, stop-sock, help\n")
  fmt.Printf(">>> \tpid:              Prints the process ID\n")
  fmt.Printf(">>> \topenfile:         Opens the file 'myfile' available on this directory. The FD generated by the kernel is stored in memory\n")
  fmt.Printf(">>> \tclosefile:        Closes the last stored FD\n")
  fmt.Printf(">>> \tlsof:             Lists FDs stored in memory\n")
  fmt.Printf(">>> \tstart-sock:       On a server, starts a socket and listens for incoming client requests\n")
  fmt.Printf(">>> \tconnect-sock:     On a client, connects to a socket and receives FDs from a server\n")
  fmt.Printf(">>> \tread-first-fd:    Reads the file corresponding to the first open file descriptor\n")
  fmt.Printf(">>> \twrite-first-fd:   Writes data to the file corresponding to the first open file descriptor\n")
  fmt.Printf(">>> \tstop-sock:        Stops the socket\n\n")
}
