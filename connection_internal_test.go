// Copyright (c) quickfixengine.org  All rights reserved.
//
// This file may be distributed under the terms of the quickfixengine.org
// license as defined by quickfixengine.org and appearing in the file
// LICENSE included in the packaging of this file.
//
// This file is provided AS IS with NO WARRANTY OF ANY KIND, INCLUDING
// THE WARRANTY OF DESIGN, MERCHANTABILITY AND FITNESS FOR A
// PARTICULAR PURPOSE.
//
// See http://www.quickfixengine.org/LICENSE for licensing information.
//
// Contact ask@quickfixengine.org if any conditions of this licensing
// are not clear to you.

package quickfix

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWriteObserverWithTCPAndTLS(t *testing.T) {
	certificateServer := httptest.NewTLSServer(nil)
	certificate := certificateServer.TLS.Certificates[0]
	certificateServer.Close()
	for _, encrypted := range []bool{false, true} {
		t.Run(map[bool]string{false: "TCP", true: "TLS"}[encrypted], func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			if encrypted {
				listener = tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
			}
			defer listener.Close()
			payload := []byte("8=FIX.4.4\x019=5\x0135=D\x0110=000\x01")
			read := make(chan error, 1)
			go func() {
				peer, acceptErr := listener.Accept()
				if acceptErr != nil {
					read <- acceptErr
					return
				}
				defer peer.Close()
				_ = peer.SetDeadline(time.Now().Add(5 * time.Second))
				buf := make([]byte, len(payload))
				_, readErr := io.ReadFull(peer, buf)
				if readErr == nil && !bytes.Equal(buf, payload) {
					readErr = errors.New("write changed bytes")
				}
				read <- readErr
			}()
			var connection net.Conn
			dialer := &net.Dialer{Timeout: 5 * time.Second}
			if encrypted {
				connection, err = tls.DialWithDialer(dialer, "tcp", listener.Addr().String(), &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12})
			} else {
				connection, err = dialer.Dial("tcp", listener.Addr().String())
			}
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
			observed := false
			log := writeObserverLog{observe: func(raw []byte) func(int, error) {
				return func(n int, err error) { observed = n == len(payload) && err == nil }
			}}
			messages := make(chan []byte, 1)
			messages <- payload
			close(messages)
			writeLoop(connection, messages, log)
			if !observed {
				t.Fatal("socket write not observed")
			}
			if err := <-read; err != nil {
				t.Fatal(err)
			}
		})
	}
}

type writeObserverLog struct {
	nullLog
	observe func([]byte) func(int, error)
}

func (l writeObserverLog) OnOutgoingWrite(raw []byte) func(int, error) { return l.observe(raw) }

type observedWriter func([]byte) (int, error)

func (w observedWriter) Write(raw []byte) (int, error) { return w(raw) }

func TestWriteObserverRunsAfterWriteAndPreservesResult(t *testing.T) {
	for _, test := range []struct {
		name string
		n    int
		err  error
	}{
		{"success", 3, nil},
		{"short write", 1, nil},
		{"partial error", 1, io.ErrClosedPipe},
	} {
		t.Run(test.name, func(t *testing.T) {
			started, returned, observed := false, false, false
			log := writeObserverLog{observe: func(raw []byte) func(int, error) {
				if string(raw) != "FIX" || returned {
					t.Fatal("observer started at wrong boundary")
				}
				started = true
				return func(n int, err error) {
					if !returned || n != test.n || !errors.Is(err, test.err) {
						t.Fatal("incorrect write completion")
					}
					observed = true
				}
			}}
			writer := observedWriter(func(raw []byte) (int, error) {
				if !started || observed || string(raw) != "FIX" {
					t.Fatal("write was changed or observed early")
				}
				returned = true
				return test.n, test.err
			})
			messages := make(chan []byte, 1)
			messages <- []byte("FIX")
			close(messages)
			writeLoop(writer, messages, log)
			if !observed {
				t.Fatal("write completion missing")
			}
		})
	}
}

func TestDisabledWriteObserverDoesNotChangeBytes(t *testing.T) {
	var writer bytes.Buffer
	messages := make(chan []byte, 1)
	messages <- []byte("unchanged")
	close(messages)
	writeLoop(&writer, messages, writeObserverLog{observe: func([]byte) func(int, error) { return nil }})
	if writer.String() != "unchanged" {
		t.Fatal("disabled observer changed output")
	}
}

func TestWriteLoop(t *testing.T) {
	writer := bytes.NewBufferString("")
	msgOut := make(chan []byte)

	go func() {
		msgOut <- []byte("test msg 1 ")
		msgOut <- []byte("test msg 2 ")
		msgOut <- []byte("test msg 3")
		close(msgOut)
	}()
	writeLoop(writer, msgOut, nullLog{})

	expected := "test msg 1 test msg 2 test msg 3"

	if writer.String() != expected {
		t.Errorf("expected %v got %v", expected, writer.String())
	}
}

func TestReadLoop(t *testing.T) {
	msgIn := make(chan fixIn)
	stream := "hello8=FIX.4.09=5blah10=103garbage8=FIX.4.09=4foo10=103"

	parser := newParser(strings.NewReader(stream))
	go readLoop(parser, msgIn, nullLog{})

	var tests = []struct {
		expectedMsg   string
		channelClosed bool
	}{
		{expectedMsg: "8=FIX.4.09=5blah10=103"},
		{expectedMsg: "8=FIX.4.09=4foo10=103"},
		{channelClosed: true},
	}

	for _, test := range tests {
		msg, ok := <-msgIn
		switch {
		case !ok && !test.channelClosed:
			t.Error("Channel unexpectedly closed")
			fallthrough
		case !ok && test.channelClosed:
			continue
		}

		if msg.bytes.String() != test.expectedMsg {
			t.Errorf("Expected %v got %v", test.expectedMsg, msg.bytes.String())
		}
	}
}
