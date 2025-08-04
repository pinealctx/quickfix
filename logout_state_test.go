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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/quickfixgo/quickfix/internal"
)

type LogoutStateTestSuite struct {
	SessionSuiteRig
}

func TestLogoutStateTestSuite(t *testing.T) {
	suite.Run(t, new(LogoutStateTestSuite))
}

func (s *LogoutStateTestSuite) SetupTest() {
	s.Init()
	s.Session.State = logoutState{}
}

func (s *LogoutStateTestSuite) TestPreliminary() {
	s.False(s.Session.IsLoggedOn())
	s.True(s.Session.IsConnected())
	s.True(s.Session.IsSessionTime())
}

func (s *LogoutStateTestSuite) TestTimeoutLogoutTimeout() {
	s.MockApp.On("OnLogout").Return(nil)
	s.Timeout(s.Session, internal.LogoutTimeout)

	s.MockApp.AssertExpectations(s.T())
	s.State(latentState{})
}

func (s *LogoutStateTestSuite) TestTimeoutNotLogoutTimeout() {
	tests := []internal.Event{internal.PeerTimeout, internal.NeedHeartbeat, internal.LogonTimeout}

	for _, test := range tests {
		s.Timeout(s.Session, test)
		s.State(logoutState{})
	}
}

func (s *LogoutStateTestSuite) TestDisconnected() {
	s.MockApp.On("OnLogout").Return(nil)
	s.Session.Disconnected(s.Session)

	s.MockApp.AssertExpectations(s.T())
	s.State(latentState{})
}

func (s *LogoutStateTestSuite) TestFixMsgInNotLogout() {
	s.MockApp.On("FromApp").Return(nil)
	s.fixMsgIn(s.Session, s.NewOrderSingle())

	s.MockApp.AssertExpectations(s.T())
	s.State(logoutState{})
	s.NextTargetMsgSeqNum(2)
}

func (s *LogoutStateTestSuite) TestFixMsgInNotLogoutReject() {
	s.MockApp.On("FromApp").Return(ConditionallyRequiredFieldMissing(Tag(11)))
	s.MockApp.On("ToApp").Return(nil)
	s.fixMsgIn(s.Session, s.NewOrderSingle())

	s.MockApp.AssertExpectations(s.T())
	s.State(logoutState{})
	s.NextTargetMsgSeqNum(2)
	s.NextSenderMsgSeqNum(2)

	s.NoMessageSent()
}

func (s *LogoutStateTestSuite) TestFixMsgInLogout() {
	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("OnLogout").Return(nil)
	s.fixMsgIn(s.Session, s.Logout())

	s.MockApp.AssertExpectations(s.T())
	s.State(latentState{})
	s.NextTargetMsgSeqNum(2)
	s.NextSenderMsgSeqNum(1)
	s.NoMessageSent()
}

func (s *LogoutStateTestSuite) TestFixMsgInLogoutResetOnLogout() {
	s.Session.ResetOnLogout = true

	s.MockApp.On("ToApp").Return(nil)
	s.Nil(s.queueForSend(s.NewOrderSingle()))
	s.MockApp.AssertExpectations(s.T())

	s.MockApp.On("FromAdmin").Return(nil)
	s.MockApp.On("OnLogout").Return(nil)
	s.fixMsgIn(s.Session, s.Logout())

	s.MockApp.AssertExpectations(s.T())
	s.State(latentState{})
	s.NextTargetMsgSeqNum(1)
	s.NextSenderMsgSeqNum(1)

	s.NoMessageSent()
	s.NoMessageQueued()
}

func (s *LogoutStateTestSuite) TestStop() {
	s.Session.Stop(s.Session)
	s.State(logoutState{})
	s.NotStopped()
}
