# POC: Transfer file descriptors between processes in Go using UDS sockets

## How to:

Spawn two terminal instances and start the program in both instances. Let's call them the "old" process and the "new" process

1. In the "old" process:
    - Use the "openfile" command to open the file "myfile" available in the directory. (Note: the program doesn't support more than 2 file descriptors, opening more than 2 times is useless for the purpose of this demo)  
   You can check the open file descriptors with the **lsof -w -p <PID>** command on a regular shell, notice the created file descriptors for the file "myfile"

    - Use the "start-sock" command to start a server socket. The program will listen and await for client connection, to which it will transfer its open file descriptors

2. In the "new" process:
    - Check first the open file descriptors using the lsof command on a regular shell. You should find that there are no file descriptors for "myfile"
    - Use the "connect-sock" command to connect to the server socket. The client will then receive the open file descriptors from the "old" process
    - You can check the open file descriptors with the lsof command. Notice that there are now file descriptors for myfile. Note that the file descriptors are not necessarily the same as in the old process as the file descript**ors** are just per-process references to the actual file descript**ions**, these latter are held in the kernel open file table

The file descriptors created on the "new" process are unrelated to the ones on the "old" process; closing them on the "old" process won't have an impact on the "new" process.  
