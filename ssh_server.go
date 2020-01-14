// https://github.com/f-secure-foundry/tamago-example
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.
//
// +build tamago,arm

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"

	"github.com/f-secure-foundry/tamago/imx6"
)

const help = `
  exit, quit		# close session
  example		# launch example test code
  help			# this help
  rand			# gather 32 bytes from TRNG via crypto/rand
  stack			# stack trace of current goroutine
  stackall		# stack trace of all goroutines
  version		# Go runtime version
`

func handleCommand(term *terminal.Terminal, cmd string) (err error) {
	var res string

	switch cmd {
	case "exit", "quit":
		res = "logout"
		err = io.EOF
	case "example":
		log.SetOutput(term)
		defer log.SetOutput(os.Stdout)
		example()
	case "help":
		res = string(term.Escape.Cyan) + help + string(term.Escape.Reset)
	case "rand":
		buf := make([]byte, 32)
		rand.Read(buf)
		res = string(term.Escape.Cyan) + fmt.Sprintf("%x", buf) + string(term.Escape.Reset)
	case "stack":
		res = string(debug.Stack())
	case "stackall":
		pprof.Lookup("goroutine").WriteTo(term, 1)
	case "version":
		res = runtime.Version()
	default:
		res = "unknown command, type `help`"
	}

	fmt.Fprintln(term, res)

	return
}

func handleChannel(newChannel ssh.NewChannel) {
	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}

	conn, requests, err := newChannel.Accept()

	if err != nil {
		log.Printf("error accepting channel, %v", err)
		return
	}

	term := terminal.NewTerminal(conn, "")
	term.SetPrompt(string(term.Escape.Red) + "> " + string(term.Escape.Reset))

	go func() {
		defer conn.Close()

		fmt.Fprintf(term, "%s (%s %d MHz)\n", banner, imx6.Model(), imx6.ARMFreq()/1000000)
		fmt.Fprintf(term, "%s\n", string(term.Escape.Cyan)+help+string(term.Escape.Reset))

		for {
			cmd, err := term.ReadLine()

			if err == io.EOF {
				break
			}

			if err != nil {
				log.Println("readline error: %v\n", err)
				continue
			}

			err = handleCommand(term, cmd)

			if err == io.EOF {
				break
			}
		}

		log.Printf("closing ssh connection")
	}()

	go func() {
		for req := range requests {
			reqSize := len(req.Payload)

			switch req.Type {
			case "shell":
				// do not accept payload commands
				if len(req.Payload) == 0 {
					req.Reply(true, nil)
				}
			case "pty-req":
				// p10, 6.2.  Requesting a Pseudo-Terminal, RFC4254
				if reqSize < 4 {
					log.Printf("malformed pty-req request")
					continue
				}

				termVariableSize := int(req.Payload[3])

				if reqSize < 4+termVariableSize+8 {
					log.Printf("malformed pty-req request")
					continue
				}

				w := binary.BigEndian.Uint32(req.Payload[4+termVariableSize:])
				h := binary.BigEndian.Uint32(req.Payload[4+termVariableSize+4:])

				log.Printf("resizing terminal (%s:%dx%d)", req.Type, w, h)
				term.SetSize(int(w), int(h))

				req.Reply(true, nil)
			case "window-change":
				// p10, 6.7.  Window Dimension Change Message, RFC4254
				if reqSize < 8 {
					log.Printf("malformed window-change request")
					continue
				}

				w := binary.BigEndian.Uint32(req.Payload)
				h := binary.BigEndian.Uint32(req.Payload[4:])

				log.Printf("resizing terminal (%s:%dx%d)", req.Type, w, h)
				term.SetSize(int(w), int(h))
			}
		}
	}()
}

func handleChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		go handleChannel(newChannel)
	}
}

func startSSHServer(s *stack.Stack, addr tcpip.Address, port uint16, nic tcpip.NICID) {
	var err error

	fullAddr := tcpip.FullAddress{Addr: addr, Port: port, NIC: nic}
	listener, err := gonet.NewListener(s, fullAddr, ipv4.ProtocolNumber)

	if err != nil {
		log.Fatal("listener error: ", err)
	}

	srv := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {
		log.Fatal("ECDSA key error: ", err)
	}

	signer, err := ssh.NewSignerFromKey(key)

	if err != nil {
		log.Fatal("key conversion error: ", err)
	}

	log.Printf("starting ssh server (%s) at %s:%d", ssh.FingerprintSHA256(signer.PublicKey()), addr.String(), port)

	srv.AddHostKey(signer)

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Printf("error accepting connection, %v", err)
			continue
		}

		sshConn, chans, reqs, err := ssh.NewServerConn(conn, srv)

		if err != nil {
			log.Printf("error accepting handshake, %v", err)
			continue
		}

		log.Printf("new ssh connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

		go ssh.DiscardRequests(reqs)
		go handleChannels(chans)
	}
}
