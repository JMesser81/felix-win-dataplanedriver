/*
 * This code will simulate the ext_dataplane driver used by Felix
 * It will create a child process (winpddriver) and hook up stdio via pipes
*/

package main

import (
  "bytes"
  "io"
  "os"
  "time"
  "os/exec"
  "encoding/binary"

  log "github.com/Sirupsen/logrus"
  )

const ExtDriverFilename = "windpdriver"

var dpCnxn *extDataplaneConn
var dpCmd *exec.Cmd

func main() {
  //parent
  dpCnxn, dpCmd := setupIo (ExtDriverFilename)

  // MAIN server/parent loop
  // First step, simply read message from the windpdriver and echo

  go sendMessages()
  go readMessages()

  // Need to send the dataplane driver something...
  log.Info(dpCnxn.nextSeqNumber)
  log.Info(dpCmd.Dir)

  for {
    time.Sleep(time.Second * 1)
  }
}

func readMessages() {
  for {
    buf := make([]byte, 1)
    time.Sleep(time.Second * 30)
    _, err := io.ReadFull(dpCnxn.fromDataplane, buf)
    if err != nil {
      return
    }
    length := binary.LittleEndian.Uint64(buf)

    data := make([]byte, length)
    _, err = io.ReadFull(dpCnxn.fromDataplane, data)
    if err != nil {
      return
    }

    // Now I have data, what should I do with it?
    log.Infof("PARENT: Received message with length=%d: %s\n", length, data)

    // Permute and send back to client?

  }

}

func sendMessages() {

  for {

    lengthBytes := make([]byte, 1)
    lengthBytes[0] = 5

    data := make([]byte, 5)
    data[0] = 'H'
    data[1] = 'E'
    data[2] = 'L'
    data[3] = 'L'
    data[4] = 'O'

    var messageBuf bytes.Buffer
    messageBuf.Write(lengthBytes)
    messageBuf.Write(data)
    for {
      time.Sleep(time.Second * 5)
      _, err := messageBuf.WriteTo(dpCnxn.toDataplane)
      if err == io.ErrShortWrite {
        log.Warn("Short write to dataplane driver; buffer full?")
        continue
      }
      if err != nil {
        return
      }
      log.Debug("Wrote message to dataplane driver")
      break
    }
  }
}

type extDataplaneConn struct {
  fromDataplane io.Reader
  toDataplane   io.Writer
  nextSeqNumber uint64
}


/*
	envelope := proto.FromDataplane{}
	err = pb.Unmarshal(data, &envelope)
	if err != nil {
		return
	}
	log.WithField("envelope", envelope).Debug("Received message from dataplane.")

	switch payload := envelope.Payload.(type) {
	case *proto.FromDataplane_ProcessStatusUpdate:
		msg = payload.ProcessStatusUpdate
	case *proto.FromDataplane_WorkloadEndpointStatusUpdate:
		msg = payload.WorkloadEndpointStatusUpdate
	case *proto.FromDataplane_WorkloadEndpointStatusRemove:
		msg = payload.WorkloadEndpointStatusRemove
	case *proto.FromDataplane_HostEndpointStatusUpdate:
		msg = payload.HostEndpointStatusUpdate
	case *proto.FromDataplane_HostEndpointStatusRemove:
		msg = payload.HostEndpointStatusRemove
	default:
		log.WithField("payload", payload).Warn("Ignoring unknown message from dataplane")
	}
*/

func setupIo(dpBinaryFilename string) (*extDataplaneConn, *exec.Cmd) {
  // Create IPC Pipes
  toDriverR, toDriverW, err := os.Pipe()
  if err != nil {
    log.Info("PARENT: Failed to open pipe for dataplane driver")
  }
  fromDriverR, fromDriverW, err := os.Pipe()
  if err != nil {
    log.Info("PARENT: Failed to open pipe for dataplane driver")
  }

  cmd := exec.Command(dpBinaryFilename)
  driverOut, err := cmd.StdoutPipe()
  if err != nil {
    log.Info("PARENT: Failed to create pipe for dataplane driver")
  }

  driverErr, err := cmd.StderrPipe()
  if err != nil {
    log.Info("Failed to create pipe for dataplane driver")
  }

  // go - coroutine
  go io.Copy(os.Stdout, driverOut)
  go io.Copy(os.Stderr, driverErr)

  cmd.ExtraFiles = []*os.File{toDriverR, fromDriverW}
  if err := cmd.Start(); err != nil {
    log.Info("Failed to start dataplane driver", err)
  }

  // Now the sub-process is running, close our copy of the file handles
  // for the child's end of the pipes.
  if err := toDriverR.Close(); err != nil {
    cmd.Process.Kill()
    log.Info("Failed to close parent's copy of pipe")
  }
  if err := fromDriverW.Close(); err != nil {
    cmd.Process.Kill()
    log.Info("Failed to close parent's copy of pipe")
  }

  dataplaneConnection := &extDataplaneConn{
    toDataplane:   toDriverW,
    fromDataplane: fromDriverR,
  }

  return dataplaneConnection, cmd
}
