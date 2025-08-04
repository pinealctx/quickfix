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

type PendingTimeoutTestSuite struct {
	SessionSuiteRig
}

func TestPendingTimeoutTestSuite(t *testing.T) {
	suite.Run(t, new(PendingTimeoutTestSuite))
}

func (s *PendingTimeoutTestSuite) SetupTest() {
	s.Init()
}

func (s *PendingTimeoutTestSuite) TestIsConnectedIsLoggedOn() {
	tests := []pendingTimeout{
		{inSession{}},
		{resendState{}},
	}

	for _, state := range tests {
		s.Session.State = state

		s.True(s.Session.IsConnected())
		s.True(s.Session.IsLoggedOn())
	}
}

func (s *PendingTimeoutTestSuite) TestSessionTimeout() {
	tests := []pendingTimeout{
		{inSession{}},
		{resendState{}},
	}

	for _, state := range tests {
		s.Session.State = state

		s.MockApp.On("OnLogout").Return(nil)
		s.Session.Timeout(s.Session, internal.PeerTimeout)

		s.MockApp.AssertExpectations(s.T())
		s.State(latentState{})
	}
}

func (s *PendingTimeoutTestSuite) TestTimeoutUnchangedState() {
	tests := []pendingTimeout{
		{inSession{}},
		{resendState{}},
	}

	testEvents := []internal.Event{internal.NeedHeartbeat, internal.LogonTimeout, internal.LogoutTimeout}

	for _, state := range tests {
		s.Session.State = state

		for _, event := range testEvents {
			s.Session.Timeout(s.Session, event)
			s.State(state)
		}
	}
}

func (s *PendingTimeoutTestSuite) TestDisconnected() {
	tests := []pendingTimeout{
		{inSession{}},
		{resendState{}},
	}

	for _, state := range tests {
		s.SetupTest()
		s.Session.State = state

		s.MockApp.On("OnLogout").Return(nil)
		s.Session.Disconnected(s.Session)

		s.MockApp.AssertExpectations(s.T())
		s.State(latentState{})
	}
}
