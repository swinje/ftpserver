This is my implementation of Exercise 8.2 in Donovan and Kernighan’s “The Go Programming Language”.

With a LOT of help from Angus Morrison´s article https://betterprogramming.pub/how-to-write-a-concurrent-ftp-server-in-go-part-1-3904f2e3a9e5

DO NOT RUN THIS SERVER IN PRODUCTION as it allows unlimited anonymous access

On MacOS they have wisely removed the FTP and Telnet clients, install from HomeBrew with "brew install inettools"

Assuming you have Golang installed, build with "go build" and then "go install"

Default is port 8080 but you can set port for instance by "ftpserver -port 10000"


