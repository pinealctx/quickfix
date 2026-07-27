package settings

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pinealctx/quickfix"
)

func TestJSONSettingsToQuickFIX(t *testing.T) {
	input := []byte(`{
		"global": {
			"connectionType": "initiator",
			"reconnectInterval": "10",
			"logonTimeout": 10,
			"heartBtInt": 30,
			"timezone": "America/New_York",
			"dataDictionary": "FIX44.xml",
			"allowUnknownMsgFields": true,
			"validateUserDefinedFields": false,
			"fileStoreSync": false,
			"maxLatency": 60000
		},
		"sessions": [{
			"beginString": "FIX.4.4",
			"senderCompID": "SENDER",
			"targetCompID": "TARGET",
			"socketConnectHost": "fix.example.com",
			"socketConnectPort": 9299,
			"socketUseSSL": false,
			"validateFieldsOutOfOrder": false,
			"validateFieldsHaveValues": false,
			"resetOnLogon": "Y",
			"additionalSettings": {
				"SocketConnectHost1": "backup.example.com",
				"SocketConnectPort1": 9300
			}
		}]
	}`)

	settings, err := FromJSON(input)
	require.NoError(t, err)
	encoded, err := settings.ToJSON()
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"validateFieldsHaveValues":false`)

	quickFIXSettings, err := settings.ToQuickFIX()
	require.NoError(t, err)
	sessionID := quickfix.SessionID{
		BeginString:  quickfix.BeginStringFIX44,
		SenderCompID: "SENDER",
		TargetCompID: "TARGET",
	}
	session := quickFIXSettings.SessionSettings()[sessionID]
	require.NotNil(t, session)

	validateValues, err := session.BoolSetting("ValidateFieldsHaveValues")
	require.NoError(t, err)
	require.False(t, validateValues)
	reconnect, err := session.IntSetting("ReconnectInterval")
	require.NoError(t, err)
	require.Equal(t, 10, reconnect)
	logonTimeout, err := session.IntSetting("LogonTimeout")
	require.NoError(t, err)
	require.Equal(t, 10, logonTimeout)
	backupHost, err := session.Setting("SocketConnectHost1")
	require.NoError(t, err)
	require.Equal(t, "backup.example.com", backupHost)
	backupPort, err := session.IntSetting("SocketConnectPort1")
	require.NoError(t, err)
	require.Equal(t, 9300, backupPort)
}

func TestJSONSettingsRejectUnknownFields(t *testing.T) {
	_, err := FromJSON([]byte(`{
		"global": {"connectionType": "initiator"},
		"sessions": [{"validateFieldHaveValues": false}]
	}`))
	require.ErrorContains(t, err, "unknown field")
}

func TestJSONSettingsRejectInvalidConnectionType(t *testing.T) {
	_, err := FromJSON([]byte(`{
		"global": {"connectionType": "invalid"},
		"sessions": [{}]
	}`))
	require.ErrorContains(t, err, "invalid connectionType")
}

func TestJSONSettingsRequireGlobalSettings(t *testing.T) {
	_, err := FromJSON([]byte(`{"sessions": []}`))
	require.ErrorContains(t, err, "global settings are required")
}

func TestJSONSettingsRequireSession(t *testing.T) {
	_, err := FromJSON([]byte(`{
		"global": {"connectionType": "initiator"},
		"sessions": []
	}`))
	require.ErrorContains(t, err, "at least one session is required")
}

func TestJSONSettingsRequireConnectionType(t *testing.T) {
	_, err := FromJSON([]byte(`{
		"global": {},
		"sessions": [{}]
	}`))
	require.ErrorContains(t, err, "connectionType is required")
}

func TestJSONSettingsRejectTypedSettingOverride(t *testing.T) {
	_, err := FromJSON([]byte(`{
		"global": {"connectionType": "initiator"},
		"sessions": [{
			"additionalSettings": {"BeginString": "FIX.4.4"}
		}]
	}`))
	require.ErrorContains(t, err, "duplicates a typed setting")
}
