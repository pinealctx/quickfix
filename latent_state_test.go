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
)

type LatentStateTestSuite struct {
	SessionSuiteRig
}

func TestLatentStateTestSuite(t *testing.T) {
	suite.Run(t, new(LatentStateTestSuite))
}

func (s *LatentStateTestSuite) SetupTest() {
	s.Init()
	s.Session.State = latentState{}
}

func (s *LatentStateTestSuite) TestPreliminary() {
	s.False(s.Session.IsLoggedOn())
	s.False(s.Session.IsConnected())
	s.True(s.Session.IsSessionTime())
}

func (s *LatentStateTestSuite) TestDisconnected() {
	s.Session.Disconnected(s.Session)
	s.State(latentState{})
}

func (s *LatentStateTestSuite) TestStop() {
	s.Session.Stop(s.Session)
	s.Stopped()
}
