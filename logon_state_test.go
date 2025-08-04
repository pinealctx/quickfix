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
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/quickfixgo/quickfix/internal"
)

type LogonStateTestSuite struct {
	SessionSuiteRig
}

func TestLogonStateTestSuite(t *testing.T) {
	suite.Run(t, new(LogonStateTestSuite))
}

func (s *LogonStateTestSuite) SetupTest() {
	s.Init()
	s.Session.stateMachine.State = logonState{}
}

func (s *LogonStateTestSuite) TestPreliminary() {
	s.False(s.Session.IsLoggedOn())
	s.True(s.Session.IsConnected())
	s.True(s.Session.IsSessionTime())
}

func (s *LogonStateTestSuite) TestTimeoutLogonTimeout() {
	s.Timeout(s.Session, internal.LogonTimeout)
	s.State(latentState{})
}

func (s *LogonStateTestSuite) TestTimeoutLogonTimeoutInitiatedLogon() {
	s.Session.InitiateLogon = true

	s.MockApp.On("OnLogout")
	s.Timeout(s.Session, internal.LogonTimeout)

	s.MockApp.AssertExpectations(s.T())
	s.State(latentState{})
}

func (s *LogonStateTestSuite) TestTimeoutNotLogonTimeout() {
	tests := []internal.Event{internal.PeerTimeout, internal.NeedHeartbeat, internal.LogoutTimeout}

	for _, test := range tests {
		s.Timeout(s.Session, test)
		s.State(logonState{})
	}
}

func (s *LogonStateTestSuite) TestDisconnected() {
	s.Session.Disconnected(s.Session)
	s.State(latentState{})
}

func (s *LogonStateTestSuite) TestFixMsgInNotLogon() {
	s.fixMsgIn(s.Session, s.NewOrderSingle())

	s.MockApp.AssertExpectations(s.T())
	s.State(latentState{})
	s.NextTargetMsgSeqNum(1)
}

func (s *LogonStateTestSuite) TestFixMsgInLogon() {
	s.IncrNextSenderMsgSeqNum()
	s.MessageFactory.seqNum = 1
	s.IncrNextTargetMsgSeqNum()

	logon := s.Logon()
	logon.Body.SetField(tagHeartBtInt, FIXInt(32))

	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("OnLogon")
	s.MockApp.On("ToAdmin")
	s.Zero(s.Session.HeartBtInt)
	s.fixMsgIn(s.Session, logon)

	s.MockApp.AssertExpectations(s.T())

	s.State(inSession{})
	s.Equal(32*time.Second, s.Session.HeartBtInt) // Should be written from logon message.
	s.False(s.Session.HeartBtIntOverride)

	s.LastToAdminMessageSent()
	s.MessageType(string(msgTypeLogon), s.MockApp.lastToAdmin)
	s.FieldEquals(tagHeartBtInt, 32, s.MockApp.lastToAdmin.Body)

	s.NextTargetMsgSeqNum(3)
	s.NextSenderMsgSeqNum(3)
}

func (s *LogonStateTestSuite) TestFixMsgInLogonHeartBtIntOverride() {
	s.IncrNextSenderMsgSeqNum()
	s.MessageFactory.seqNum = 1
	s.IncrNextTargetMsgSeqNum()

	logon := s.Logon()
	logon.Body.SetField(tagHeartBtInt, FIXInt(32))

	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("OnLogon")
	s.MockApp.On("ToAdmin")
	s.Session.HeartBtIntOverride = true
	s.Session.HeartBtInt = time.Second
	s.fixMsgIn(s.Session, logon)

	s.MockApp.AssertExpectations(s.T())

	s.State(inSession{})
	s.Equal(time.Second, s.Session.HeartBtInt) // Should not have changed.
	s.True(s.Session.HeartBtIntOverride)

	s.LastToAdminMessageSent()
	s.MessageType(string(msgTypeLogon), s.MockApp.lastToAdmin)
	s.FieldEquals(tagHeartBtInt, 1, s.MockApp.lastToAdmin.Body)

	s.NextTargetMsgSeqNum(3)
	s.NextSenderMsgSeqNum(3)
}

func (s *LogonStateTestSuite) TestFixMsgInLogonEnableLastMsgSeqNumProcessed() {
	s.Session.EnableLastMsgSeqNumProcessed = true

	s.MessageFactory.SetNextSeqNum(2)
	s.IncrNextSenderMsgSeqNum()
	s.IncrNextTargetMsgSeqNum()

	logon := s.Logon()
	logon.Body.SetField(tagHeartBtInt, FIXInt(32))

	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("OnLogon")
	s.MockApp.On("ToAdmin")
	s.fixMsgIn(s.Session, logon)

	s.MockApp.AssertExpectations(s.T())

	s.LastToAdminMessageSent()
	s.MessageType(string(msgTypeLogon), s.MockApp.lastToAdmin)
	s.FieldEquals(tagLastMsgSeqNumProcessed, 2, s.MockApp.lastToAdmin.Header)
}

func (s *LogonStateTestSuite) TestFixMsgInLogonResetSeqNum() {
	s.IncrNextTargetMsgSeqNum()

	logon := s.Logon()
	logon.Body.SetField(tagHeartBtInt, FIXInt(32))
	logon.Body.SetField(tagResetSeqNumFlag, FIXBoolean(true))

	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("OnLogon")
	s.MockApp.On("ToAdmin")
	s.fixMsgIn(s.Session, logon)

	s.MockApp.AssertExpectations(s.T())

	s.State(inSession{})
	s.Equal(32*time.Second, s.Session.HeartBtInt)

	s.LastToAdminMessageSent()
	s.MessageType(string(msgTypeLogon), s.MockApp.lastToAdmin)
	s.FieldEquals(tagHeartBtInt, 32, s.MockApp.lastToAdmin.Body)
	s.FieldEquals(tagResetSeqNumFlag, true, s.MockApp.lastToAdmin.Body)

	s.NextTargetMsgSeqNum(2)
	s.NextSenderMsgSeqNum(2)
}

func (s *LogonStateTestSuite) TestFixMsgInLogonInitiateLogon() {
	s.Session.InitiateLogon = true
	s.IncrNextSenderMsgSeqNum()
	s.MessageFactory.seqNum = 1
	s.IncrNextTargetMsgSeqNum()

	logon := s.Logon()
	logon.Body.SetField(tagHeartBtInt, FIXInt(32))

	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("OnLogon")
	s.fixMsgIn(s.Session, logon)

	s.MockApp.AssertExpectations(s.T())
	s.State(inSession{})

	s.NextTargetMsgSeqNum(3)
	s.NextSenderMsgSeqNum(2)
}

func (s *LogonStateTestSuite) TestFixMsgInLogonInitiateLogonExpectResetSeqNum() {
	s.Session.InitiateLogon = true
	s.Session.sentReset = true
	s.Require().Nil(s.store.IncrNextSenderMsgSeqNum())

	logon := s.Logon()
	logon.Body.SetField(tagHeartBtInt, FIXInt(32))
	logon.Body.SetField(tagResetSeqNumFlag, FIXBoolean(true))

	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("OnLogon")
	s.fixMsgIn(s.Session, logon)

	s.MockApp.AssertExpectations(s.T())
	s.State(inSession{})

	s.NextTargetMsgSeqNum(2)
	s.NextSenderMsgSeqNum(2)
}

func (s *LogonStateTestSuite) TestFixMsgInLogonInitiateLogonRejectedSeqNumNotReset() {
	s.Session.InitiateLogon = true
	s.Session.sentReset = true
	s.Require().Nil(s.store.IncrNextSenderMsgSeqNum())

	logon := s.Logon()
	logon.Body.SetField(tagHeartBtInt, FIXInt(32))
	logon.Body.SetField(tagResetSeqNumFlag, FIXBoolean(true))

	s.MockApp.On("FromAdmin").Return(RejectLogon{"reject message"})
	s.MockApp.On("OnLogout")
	s.MockApp.On("ToAdmin")
	s.fixMsgIn(s.Session, logon)

	s.MockApp.AssertExpectations(s.T())
	s.State(latentState{})

	s.NextTargetMsgSeqNum(2)
	s.NextSenderMsgSeqNum(3)
}

func (s *LogonStateTestSuite) TestFixMsgInLogonInitiateLogonUnExpectedResetSeqNum() {
	s.Session.InitiateLogon = true
	s.Session.sentReset = false
	s.IncrNextTargetMsgSeqNum()
	s.IncrNextSenderMsgSeqNum()

	logon := s.Logon()
	logon.Body.SetField(tagHeartBtInt, FIXInt(32))
	logon.Body.SetField(tagResetSeqNumFlag, FIXBoolean(true))

	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("OnLogon")
	s.fixMsgIn(s.Session, logon)

	s.MockApp.AssertExpectations(s.T())
	s.State(inSession{})

	s.NextTargetMsgSeqNum(2)
	s.NextSenderMsgSeqNum(1)
}

func (s *LogonStateTestSuite) TestFixMsgInLogonRefreshOnLogon() {
	var tests = []bool{true, false}

	for _, doRefresh := range tests {
		s.SetupTest()
		s.Session.RefreshOnLogon = doRefresh

		logon := s.Logon()
		logon.Body.SetField(tagHeartBtInt, FIXInt(32))

		if doRefresh {
			s.MockStore.On("Refresh").Return(nil)
		}
		s.MockApp.On("FromAdmin").Return(nil)
		s.MockApp.On("OnLogon")
		s.MockApp.On("ToAdmin")
		s.fixMsgIn(s.Session, logon)

		s.MockStore.AssertExpectations(s.T())
	}
}

func (s *LogonStateTestSuite) TestStop() {
	var tests = []bool{true, false}

	for _, doInitiateLogon := range tests {
		s.SetupTest()
		s.Session.InitiateLogon = doInitiateLogon

		if doInitiateLogon {
			s.MockApp.On("OnLogout")
		}

		s.Session.Stop(s.Session)
		s.MockApp.AssertExpectations(s.T())
		s.Disconnected()
		s.Stopped()
	}
}

func (s *LogonStateTestSuite) TestFixMsgInLogonRejectLogon() {
	s.IncrNextSenderMsgSeqNum()
	s.MessageFactory.seqNum = 1
	s.IncrNextTargetMsgSeqNum()

	logon := s.Logon()
	logon.Body.SetField(tagHeartBtInt, FIXInt(32))

	s.MockApp.On("FromAdmin").Return(RejectLogon{"reject message"})
	s.MockApp.On("ToAdmin")
	s.fixMsgIn(s.Session, logon)

	s.MockApp.AssertExpectations(s.T())

	s.State(latentState{})

	s.LastToAdminMessageSent()
	s.MessageType(string(msgTypeLogout), s.MockApp.lastToAdmin)
	s.FieldEquals(tagText, "reject message", s.MockApp.lastToAdmin.Body)

	s.NextTargetMsgSeqNum(3)
	s.NextSenderMsgSeqNum(3)
}

func (s *LogonStateTestSuite) TestFixMsgInLogonSeqNumTooHigh() {
	s.MessageFactory.SetNextSeqNum(6)
	logon := s.Logon()
	logon.Body.SetField(tagHeartBtInt, FIXInt(32))

	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("OnLogon")
	s.MockApp.On("ToAdmin")
	s.fixMsgIn(s.Session, logon)

	s.State(resendState{})
	s.NextTargetMsgSeqNum(1)

	// Session should send logon, and then queues resend request for send.
	s.MockApp.AssertNumberOfCalls(s.T(), "ToAdmin", 2)
	msgBytesSent, ok := s.Receiver.LastMessage()
	s.Require().True(ok)
	sentMessage := NewMessage()
	err := ParseMessage(sentMessage, bytes.NewBuffer(msgBytesSent))
	s.Require().Nil(err)
	s.MessageType(string(msgTypeLogon), sentMessage)

	s.Session.sendQueued(true)
	s.MessageType(string(msgTypeResendRequest), s.MockApp.lastToAdmin)
	s.FieldEquals(tagBeginSeqNo, 1, s.MockApp.lastToAdmin.Body)

	s.MockApp.On("FromAdmin").Return(nil)
	s.MessageFactory.SetNextSeqNum(1)
	s.fixMsgIn(s.Session, s.SequenceReset(3))
	s.State(resendState{})
	s.NextTargetMsgSeqNum(3)

	s.MessageFactory.SetNextSeqNum(3)
	s.MockApp.On("FromAdmin").Return(nil)
	s.fixMsgIn(s.Session, s.SequenceReset(7))
	s.State(inSession{})
	s.NextTargetMsgSeqNum(7)
}

func (s *LogonStateTestSuite) TestFixMsgInLogonSeqNumTooLow() {
	s.IncrNextSenderMsgSeqNum()
	s.IncrNextTargetMsgSeqNum()

	logon := s.Logon()
	logon.Body.SetField(tagHeartBtInt, FIXInt(32))
	logon.Header.SetInt(tagMsgSeqNum, 1)

	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("ToAdmin")
	s.NextTargetMsgSeqNum(2)
	s.fixMsgIn(s.Session, logon)

	s.State(latentState{})
	s.NextTargetMsgSeqNum(2)

	s.MockApp.AssertNumberOfCalls(s.T(), "ToAdmin", 1)
	msgBytesSent, ok := s.Receiver.LastMessage()
	s.Require().True(ok)
	sentMessage := NewMessage()
	err := ParseMessage(sentMessage, bytes.NewBuffer(msgBytesSent))
	s.Require().Nil(err)
	s.MessageType(string(msgTypeLogout), sentMessage)

	s.Session.sendQueued(true)
	s.MessageType(string(msgTypeLogout), s.MockApp.lastToAdmin)
	s.FieldEquals(tagText, "MsgSeqNum too low, expecting 2 but received 1", s.MockApp.lastToAdmin.Body)
}

func (s *LogonStateTestSuite) TestStayLoggedInOnReset() {
	s.IncrNextTargetMsgSeqNum()
	s.IncrNextSenderMsgSeqNum()

	logon := s.Logon()
	logon.Body.SetField(tagResetSeqNumFlag, FIXBoolean(true))

	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("OnLogon")
	s.MockApp.On("ToAdmin")
	s.fixMsgIn(s.Session, logon)

	s.MockApp.AssertExpectations(s.T())

	s.State(inSession{})

	s.IncrNextTargetMsgSeqNum()
	s.IncrNextSenderMsgSeqNum()

	s.NextTargetMsgSeqNum(3)
	s.NextSenderMsgSeqNum(3)

	s.fixMsgIn(s.Session, logon)

	s.True(s.Session.IsConnected())
	s.True(s.Session.IsLoggedOn())

	s.NextTargetMsgSeqNum(2)
	s.NextSenderMsgSeqNum(2)
}
