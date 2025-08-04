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
	"fmt"
	"time"

	"github.com/quickfixgo/quickfix/internal"
)

type stateMachine struct {
	State                 sessionState
	pendingStop, stopped  bool
	notifyOnInSessionTime chan interface{}
}

func (sm *stateMachine) Start(s *Session) {
	sm.pendingStop = false
	sm.stopped = false

	sm.State = latentState{}
	sm.CheckSessionTime(s, time.Now())
}

func (sm *stateMachine) Connect(session *Session) {
	// No special logon logic needed for FIX Acceptors.
	if !session.InitiateLogon {
		sm.setState(session, logonState{})
		return
	}

	if session.RefreshOnLogon {
		if err := session.store.Refresh(); err != nil {
			session.logError(err)
			return
		}
	}

	if session.ResetOnLogon {
		if err := session.store.Reset(); err != nil {
			session.logError(err)
			return
		}
	}

	session.log.OnEvent("Sending logon request")
	if err := session.sendLogon(); err != nil {
		session.logError(err)
		return
	}

	sm.setState(session, logonState{})
	// Fire logon timeout event after the pre-configured delay period.
	time.AfterFunc(session.LogonTimeout, func() { session.sessionEvent <- internal.LogonTimeout })
}

func (sm *stateMachine) Stop(session *Session) {
	sm.pendingStop = true
	sm.setState(session, sm.State.Stop(session))
}

func (sm *stateMachine) Stopped() bool {
	return sm.stopped
}

func (sm *stateMachine) Disconnected(session *Session) {
	if sm.IsConnected() {
		sm.setState(session, latentState{})
	}
}

func (sm *stateMachine) Incoming(session *Session, m fixIn) {
	sm.CheckSessionTime(session, time.Now())
	if !sm.IsConnected() {
		return
	}

	session.log.OnIncoming(m.bytes.Bytes())

	msg := NewMessage()
	if err := session.ParseMessage(msg, m.bytes); err != nil {
		session.log.OnEventf("Msg Parse Error: %v, %q", err.Error(), m.bytes)
	} else {
		msg.ReceiveTime = m.receiveTime
		sm.fixMsgIn(session, msg)
	}

	session.peerTimer.Reset(time.Duration(float64(1.2) * float64(session.HeartBtInt)))
}

func (sm *stateMachine) fixMsgIn(session *Session, m *Message) {
	sm.setState(session, sm.State.FixMsgIn(session, m))
}

func (sm *stateMachine) SendAppMessages(session *Session) {
	sm.CheckSessionTime(session, time.Now())

	session.sendMutex.Lock()
	defer session.sendMutex.Unlock()

	if session.IsLoggedOn() {
		session.sendQueued(false)
	} else {
		session.dropQueued()
	}
}

func (sm *stateMachine) Timeout(session *Session, e internal.Event) {
	sm.CheckSessionTime(session, time.Now())
	sm.setState(session, sm.State.Timeout(session, e))
}

func (sm *stateMachine) CheckSessionTime(session *Session, now time.Time) {
	if !session.SessionTime.IsInRange(now) {
		if sm.IsSessionTime() {
			session.log.OnEvent("Not in Session")
		}

		sm.State.ShutdownNow(session)
		sm.setState(session, notSessionTime{})

		if sm.notifyOnInSessionTime == nil {
			sm.notifyOnInSessionTime = make(chan interface{})
		}
		return
	}

	if !sm.IsSessionTime() {
		session.log.OnEvent("In Session")
		sm.notifyInSessionTime()
		sm.setState(session, latentState{})
	}

	if !session.SessionTime.IsInSameRange(session.store.CreationTime(), now) {
		session.log.OnEvent("Session reset")
		sm.State.ShutdownNow(session)
		if err := session.dropAndReset(); err != nil {
			session.logError(err)
		}
		sm.setState(session, latentState{})
	}
}

func (sm *stateMachine) CheckResetTime(session *Session, now time.Time) {
	if session.EnableResetSeqTime {
		if session.ResetSeqTime.Hour() == now.Hour() &&
			session.ResetSeqTime.Minute() == now.Minute() &&
			session.ResetSeqTime.Second() == now.Second() {
			session.sendLogonInReplyTo(true, nil)
		}
	}
}

func (sm *stateMachine) setState(session *Session, nextState sessionState) {
	if !nextState.IsConnected() {
		if sm.IsConnected() {
			sm.handleDisconnectState(session)
		}

		if sm.pendingStop {
			sm.stopped = true
			sm.notifyInSessionTime()
		}
	}

	sm.State = nextState
}

func (sm *stateMachine) notifyInSessionTime() {
	if sm.notifyOnInSessionTime != nil {
		close(sm.notifyOnInSessionTime)
	}
	sm.notifyOnInSessionTime = nil
}

func (sm *stateMachine) handleDisconnectState(s *Session) {
	doOnLogout := s.IsLoggedOn()

	switch s.State.(type) {
	case logoutState:
		doOnLogout = true
	case logonState:
		if s.InitiateLogon {
			doOnLogout = true
		}
	}

	if doOnLogout {
		s.application.OnLogout(s.sessionID)
	}

	s.onDisconnect()
}

func (sm *stateMachine) IsLoggedOn() bool {
	return sm.State.IsLoggedOn()
}

func (sm *stateMachine) IsConnected() bool {
	return sm.State.IsConnected()
}

func (sm *stateMachine) IsSessionTime() bool {
	return sm.State.IsSessionTime()
}

func handleStateError(s *Session, err error) sessionState {
	s.logError(err)
	return latentState{}
}

// sessionState is the current state of the Session state machine. The Session state determines how the Session responds to
// incoming messages, timeouts, and requests to send application messages.
type sessionState interface {
	// FixMsgIn is called by the Session on incoming messages from the counter party.
	// The return type is the next Session state following message processing.
	FixMsgIn(*Session, *Message) (nextState sessionState)

	// Timeout is called by the Session on a timeout event.
	Timeout(*Session, internal.Event) (nextState sessionState)

	// IsLoggedOn returns true if state is logged on an in Session, false otherwise.
	IsLoggedOn() bool

	// IsConnected returns true if the state is connected.
	IsConnected() bool

	// IsSessionTime returns true if the state is in Session time.
	IsSessionTime() bool

	// ShutdownNow terminates the Session state immediately.
	ShutdownNow(*Session)

	// Stop triggers a clean stop.
	Stop(*Session) (nextState sessionState)

	// Stringer debugging convenience.
	fmt.Stringer
}

type inSessionTime struct{}

func (inSessionTime) IsSessionTime() bool { return true }

type connected struct{}

func (connected) IsConnected() bool   { return true }
func (connected) IsSessionTime() bool { return true }

type connectedNotLoggedOn struct{ connected }

func (connectedNotLoggedOn) IsLoggedOn() bool     { return false }
func (connectedNotLoggedOn) ShutdownNow(*Session) {}

type loggedOn struct{ connected }

func (loggedOn) IsLoggedOn() bool { return true }
func (loggedOn) ShutdownNow(s *Session) {
	if err := s.sendLogout(""); err != nil {
		s.logError(err)
	}
}

func (loggedOn) Stop(s *Session) (nextState sessionState) {
	if err := s.initiateLogout(""); err != nil {
		return handleStateError(s, err)
	}

	return logoutState{}
}
