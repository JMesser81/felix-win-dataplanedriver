package main

import (
  "fmt"
  "os"
  "bufio"
  "time"
)

func main() {
  // child

  // Write message to server and exit
  fmt.Println("Opening pipes")
  //reader := bufio.NewReader(os.Stdin)


  writer := bufio.NewWriter(os.Stdout)

  var length byte = 5
  //var test string = "Hello!"
  if err := writer.WriteByte(length); err != nil {
    fmt.Println("Error")
  }

  data := make([]byte, 5)
  if _, err := writer.Write(data); err != nil {
    fmt.Println("Error")
  }
  writer.Flush()


  //go child
  time.Sleep(time.Second * 10)
  return
}
