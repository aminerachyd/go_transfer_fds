package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func main() {
  fmt.Println("Available options: pid, openfile, closefile, preparesock")
  reader := bufio.NewReader(os.Stdin)
  filePath := "myfile"
  socketPath := "upgrade.sock"
  var err error
  var fd int
  var socketConn *net.UnixConn
  var socketFd int

  for {
    input, _ := reader.ReadSlice('\n')
    str_input := strings.TrimSuffix(string(input), "\n")

    if str_input == "pid" {
      fmt.Printf("PID is : %d\n\n", os.Getpid())

    } else if str_input == "openfile" {
      fd, err = openFile(filePath)
      handleError(err)

    } else if str_input == "closefile" {
      err = closeFile(fd, filePath)
      handleError(err)

    } else if str_input == "preparesock" {
      socketFd, err = prepareSock(socketPath)
      handleError(err)
      fmt.Printf("Bound socket connection %v\n", socketConn)

    } else if strings.Split(str_input, " ")[0] == "sendmsg" {
      fd, err := strconv.ParseInt(strings.Split(str_input, " ")[1], 10, 8)
      handleError(err)
      sendmsg(socketFd, int(fd))

    } else if str_input == "recvmsg" {
      err := recvmsg(socketPath, socketConn)
      handleError(err)
    } else {
      fmt.Printf("Unknown option: %s\n", input)
    }
  }
}

func openFile(filePath string) (int, error) {
  fd, err := syscall.Open(filePath, syscall.O_RDWR, 0)
  if err != nil {
    return -1, err
  }

  fmt.Printf("File %s opened, fd %d\n", filePath, fd)

  return fd, nil
}

func closeFile(fd int, filePath string) error {
  err := syscall.Close(fd)
  if err != nil {
     return err
  }

  fmt.Printf("File %s with fd %d closed\n", filePath, fd)
  return nil
}

func prepareSock(path string) (int, error) {
  syscall.Unlink(path)
  listener, err := net.Listen("unix", path)
  if err != nil {
    return -1, err
  }

  a, err := listener.Accept()
  if err != nil {
    return -1, err
  }
  listenConn := a.(*net.UnixConn)
  
  f, err := listenConn.File()
  if err != nil {
    return -1, err
  }

  socketFd := int(f.Fd())

  return socketFd, nil
}

func sendmsg(socketFd int, fd int) error {
  fds := make([]int, 1)
  fds = append(fds, fd)

  rights := syscall.UnixRights(fds...)

  return syscall.Sendmsg(socketFd, nil, rights, nil, 0)
}

func recvmsg(socketPath string, conn *net.UnixConn) error {
  if conn == nil {
    c, err := net.Dial("unix", socketPath)
    if err != nil {
      return err
    }

    conn = c.(*net.UnixConn)
  }

  f, err := conn.File()
  if err != nil {
    return err
  }

  socketFd := int(f.Fd())

  buf := make([]byte, syscall.CmsgSpace(8))

  fmt.Printf("Syscall recvmsg\n")
  _,_,_,_, err = syscall.Recvmsg(socketFd, nil, buf, 0)
  fmt.Printf("DONE Syscall recvmsg\n")
  if err != nil {
    return err
  }

  // Parse control msgs
  var msgs []syscall.SocketControlMessage
  msgs, err = syscall.ParseSocketControlMessage(buf)
  for i := 0; i < len(msgs); i++ {
    var fds []int
    fds, err = syscall.ParseUnixRights(&msgs[i])

    for fi, fd := range fds {
      fmt.Printf("Received fd %d [%d]\n", fd, fi)
      syscall.Close(fd)
    }
  }

  return nil
}

func handleError(e error) {
  if e != nil {
    panic(e) 
  }
}

