package quickfix

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/pinealctx/quickfix/datadictionary"
	"github.com/stretchr/testify/require"
)

func TestNestedGroupBodyBoundaries(t *testing.T) {
	dictionary, err := datadictionary.Parse("spec/FIX44.xml")
	require.NoError(t, err)
	for _, tc := range []struct {
		name, body string
		parties    int
		contra     bool
	}{
		{"body after child", "453=1|448=party1|447=D|452=3|802=1|523=sub1|17=execution|1=account|", 1, false},
		{"repeated parent after child", "453=2|448=party1|447=D|452=3|802=1|523=sub1|448=party2|447=D|452=3|802=1|523=sub2|17=execution|1=account|", 2, false},
		{"sibling root after child", "453=1|448=party1|447=D|452=3|802=1|523=sub1|382=1|375=broker|17=execution|1=account|", 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire := groupBoundaryWire("8", tc.body)
			message := NewMessage()
			require.NoError(t, ParseMessageWithDataDictionary(message, bytes.NewBuffer(wire), dictionary, dictionary))
			execution, e := message.Body.GetString(17)
			require.NoError(t, e)
			require.Equal(t, "execution", execution)
			account, e := message.Body.GetString(1)
			require.NoError(t, e)
			require.Equal(t, "account", account)
			group := NewRepeatingGroup(453, GroupTemplate{GroupElement(448), GroupElement(447), GroupElement(452), NewRepeatingGroup(802, GroupTemplate{GroupElement(523), GroupElement(803)})})
			require.NoError(t, message.Body.GetGroup(group))
			require.Equal(t, tc.parties, group.Len())
			for i := 0; i < tc.parties; i++ {
				sub := NewRepeatingGroup(802, GroupTemplate{GroupElement(523), GroupElement(803)})
				require.NoError(t, group.Get(i).GetGroup(sub))
				require.Equal(t, 1, sub.Len())
				v, e := sub.Get(0).GetString(523)
				require.NoError(t, e)
				require.Equal(t, fmt.Sprintf("sub%d", i+1), v)
			}
			if tc.contra {
				g := NewRepeatingGroup(382, GroupTemplate{GroupElement(375)})
				require.NoError(t, message.Body.GetGroup(g))
				require.Equal(t, 1, g.Len())
			}
			rebuilt := message.build()
			require.Equal(t, 1, bytes.Count(rebuilt, []byte("\x0110=")))
			reparsed := NewMessage()
			require.NoError(t, ParseMessageWithDataDictionary(reparsed, bytes.NewBuffer(rebuilt), dictionary, dictionary))
			v, e := reparsed.Body.GetString(17)
			require.NoError(t, e)
			require.Equal(t, "execution", v)
		})
	}
}

func groupBoundaryWire(kind, body string) []byte {
	payload := bytes.ReplaceAll([]byte("35="+kind+"|34=1|49=SENDER|56=TARGET|52=20260101-00:00:00.000|"+body), []byte("|"), []byte{1})
	wire := append([]byte(fmt.Sprintf("8=FIX.4.4\x019=%d\x01", len(payload))), payload...)
	checksum := 0
	for _, b := range wire {
		checksum += int(b)
	}
	return append(wire, []byte(fmt.Sprintf("10=%03d\x01", checksum%256))...)
}

func TestNestedGroupUnterminatedInputReturnsError(t *testing.T) {
	dictionary, err := datadictionary.Parse("spec/FIX44.xml")
	require.NoError(t, err)
	for _, tail := range []string{"", "17", "17=broken"} {
		t.Run(fmt.Sprintf("tail-%d", len(tail)), func(t *testing.T) {
			raw := groupBoundaryWire("8", "453=1|448=party1|447=D|452=3|802=1|523=sub1|")
			raw = append(raw[:bytes.LastIndex(raw, []byte("10="))], []byte(tail)...)
			require.NotPanics(t, func() {
				require.Error(t, ParseMessageWithDataDictionary(NewMessage(), bytes.NewBuffer(raw), dictionary, dictionary))
			})
		})
	}
}
