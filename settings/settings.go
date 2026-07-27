package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/quickfixgo/quickfix"
)

// ConnectionType identifies whether a session accepts or initiates FIX
// connections.
type ConnectionType string

const (
	ConnectionTypeAcceptor  ConnectionType = "acceptor"
	ConnectionTypeInitiator ConnectionType = "initiator"
)

func (c ConnectionType) valid() bool {
	return c == ConnectionTypeAcceptor || c == ConnectionTypeInitiator
}

// SessionIDSettings identifies a FIX session.
type SessionIDSettings struct {
	BeginString      *string `json:"beginString,omitempty" quickfix:"BeginString"`
	SenderCompID     *string `json:"senderCompID,omitempty" quickfix:"SenderCompID"`
	SenderSubID      *string `json:"senderSubID,omitempty" quickfix:"SenderSubID"`
	SenderLocationID *string `json:"senderLocationID,omitempty" quickfix:"SenderLocationID"`
	TargetCompID     *string `json:"targetCompID,omitempty" quickfix:"TargetCompID"`
	TargetSubID      *string `json:"targetSubID,omitempty" quickfix:"TargetSubID"`
	TargetLocationID *string `json:"targetLocationID,omitempty" quickfix:"TargetLocationID"`
	SessionQualifier *string `json:"sessionQualifier,omitempty" quickfix:"SessionQualifier"`
}

// ScheduleSettings controls session activation and sequence resets.
type ScheduleSettings struct {
	DefaultApplVerID   *string  `json:"defaultApplVerID,omitempty" quickfix:"DefaultApplVerID"`
	StartTime          *string  `json:"startTime,omitempty" quickfix:"StartTime"`
	EndTime            *string  `json:"endTime,omitempty" quickfix:"EndTime"`
	StartDay           *string  `json:"startDay,omitempty" quickfix:"StartDay"`
	EndDay             *string  `json:"endDay,omitempty" quickfix:"EndDay"`
	Weekdays           *string  `json:"weekdays,omitempty" quickfix:"Weekdays"`
	TimeZone           *string  `json:"timeZone,omitempty" quickfix:"TimeZone"`
	TimeStampPrecision *string  `json:"timeStampPrecision,omitempty" quickfix:"TimeStampPrecision"`
	ResetOnLogon       *Boolean `json:"resetOnLogon,omitempty" quickfix:"ResetOnLogon"`
	RefreshOnLogon     *Boolean `json:"refreshOnLogon,omitempty" quickfix:"RefreshOnLogon"`
	ResetOnLogout      *Boolean `json:"resetOnLogout,omitempty" quickfix:"ResetOnLogout"`
	ResetOnDisconnect  *Boolean `json:"resetOnDisconnect,omitempty" quickfix:"ResetOnDisconnect"`
	ResetSeqTime       *string  `json:"resetSeqTime,omitempty" quickfix:"ResetSeqTime"`
}

// ValidationSettings controls data dictionaries and message validation.
type ValidationSettings struct {
	DataDictionary            *string  `json:"dataDictionary,omitempty" quickfix:"DataDictionary"`
	TransportDataDictionary   *string  `json:"transportDataDictionary,omitempty" quickfix:"TransportDataDictionary"`
	AppDataDictionary         *string  `json:"appDataDictionary,omitempty" quickfix:"AppDataDictionary"`
	RejectInvalidMessage      *Boolean `json:"rejectInvalidMessage,omitempty" quickfix:"RejectInvalidMessage"`
	AllowUnknownMessageFields *Boolean `json:"allowUnknownMsgFields,omitempty" quickfix:"AllowUnknownMsgFields"`
	CheckUserDefinedFields    *Boolean `json:"validateUserDefinedFields,omitempty" quickfix:"ValidateUserDefinedFields"`
	ValidateFieldsOutOfOrder  *Boolean `json:"validateFieldsOutOfOrder,omitempty" quickfix:"ValidateFieldsOutOfOrder"`
	ValidateFieldsHaveValues  *Boolean `json:"validateFieldsHaveValues,omitempty" quickfix:"ValidateFieldsHaveValues"`
	CheckLatency              *Boolean `json:"checkLatency,omitempty" quickfix:"CheckLatency"`
	MaxLatency                *Integer `json:"maxLatency,omitempty" quickfix:"MaxLatency"`
	InChanCapacity            *Integer `json:"inChanCapacity,omitempty" quickfix:"InChanCapacity"`
}

// TLSSettings controls TLS connections.
type TLSSettings struct {
	SocketPrivateKeyFile     *string  `json:"socketPrivateKeyFile,omitempty" quickfix:"SocketPrivateKeyFile"`
	SocketCertificateFile    *string  `json:"socketCertificateFile,omitempty" quickfix:"SocketCertificateFile"`
	SocketCAFile             *string  `json:"socketCAFile,omitempty" quickfix:"SocketCAFile"`
	SocketPrivateKeyBytes    *string  `json:"socketPrivateKeyBytes,omitempty" quickfix:"SocketPrivateKeyBytes"`
	SocketCertificateBytes   *string  `json:"socketCertificateBytes,omitempty" quickfix:"SocketCertificateBytes"`
	SocketCABytes            *string  `json:"socketCABytes,omitempty" quickfix:"SocketCABytes"`
	SocketInsecureSkipVerify *Boolean `json:"socketInsecureSkipVerify,omitempty" quickfix:"SocketInsecureSkipVerify"`
	SocketServerName         *string  `json:"socketServerName,omitempty" quickfix:"SocketServerName"`
	SocketMinimumTLSVersion  *string  `json:"socketMinimumTLSVersion,omitempty" quickfix:"SocketMinimumTLSVersion"`
	SocketUseSSL             *Boolean `json:"socketUseSSL,omitempty" quickfix:"SocketUseSSL"`
}

// LoggingSettings configures built-in log factories.
type LoggingSettings struct {
	FileLogPath           *string `json:"fileLogPath,omitempty" quickfix:"FileLogPath"`
	SQLLogDriver          *string `json:"sqlLogDriver,omitempty" quickfix:"SQLLogDriver"`
	SQLLogDataSourceName  *string `json:"sqlLogDataSourceName,omitempty" quickfix:"SQLLogDataSourceName"`
	SQLLogConnMaxLifetime *string `json:"sqlLogConnMaxLifetime,omitempty" quickfix:"SQLLogConnMaxLifetime"`
	MongoLogConnection    *string `json:"mongoLogConnection,omitempty" quickfix:"MongoLogConnection"`
	MongoLogDatabase      *string `json:"mongoLogDatabase,omitempty" quickfix:"MongoLogDatabase"`
	MongoLogReplicaSet    *string `json:"mongoLogReplicaSet,omitempty" quickfix:"MongoLogReplicaSet"`
}

// StorageSettings configures built-in message stores.
type StorageSettings struct {
	PersistMessages           *Boolean `json:"persistMessages,omitempty" quickfix:"PersistMessages"`
	FileStorePath             *string  `json:"fileStorePath,omitempty" quickfix:"FileStorePath"`
	FileStoreSync             *Boolean `json:"fileStoreSync,omitempty" quickfix:"FileStoreSync"`
	SQLStoreDriver            *string  `json:"sqlStoreDriver,omitempty" quickfix:"SQLStoreDriver"`
	SQLStoreDataSourceName    *string  `json:"sqlStoreDataSourceName,omitempty" quickfix:"SQLStoreDataSourceName"`
	SQLStoreConnMaxLifetime   *string  `json:"sqlStoreConnMaxLifetime,omitempty" quickfix:"SQLStoreConnMaxLifetime"`
	SQLStoreMessagesTableName *string  `json:"sqlStoreMessagesTableName,omitempty" quickfix:"SQLStoreMessagesTableName"`
	SQLStoreSessionsTableName *string  `json:"sqlStoreSessionsTableName,omitempty" quickfix:"SQLStoreSessionsTableName"`
	MongoStoreConnection      *string  `json:"mongoStoreConnection,omitempty" quickfix:"MongoStoreConnection"`
	MongoStoreDatabase        *string  `json:"mongoStoreDatabase,omitempty" quickfix:"MongoStoreDatabase"`
	MongoStoreReplicaSet      *string  `json:"mongoStoreReplicaSet,omitempty" quickfix:"MongoStoreReplicaSet"`
}

// Session is a typed set of QuickFIX settings. Fields that do not apply to a
// session's connection type are ignored by QuickFIX itself.
type Session struct {
	SessionIDSettings
	ScheduleSettings
	ValidationSettings
	TLSSettings
	LoggingSettings
	StorageSettings

	ConnectionType *ConnectionType `json:"connectionType,omitempty" quickfix:"ConnectionType"`
	HeartBtInt     *Integer        `json:"heartBtInt,omitempty" quickfix:"HeartBtInt"`

	ReconnectInterval *Integer `json:"reconnectInterval,omitempty" quickfix:"ReconnectInterval"`
	LogoutTimeout     *Integer `json:"logoutTimeout,omitempty" quickfix:"LogoutTimeout"`
	LogonTimeout      *Integer `json:"logonTimeout,omitempty" quickfix:"LogonTimeout"`
	SocketConnectHost *string  `json:"socketConnectHost,omitempty" quickfix:"SocketConnectHost"`
	SocketConnectPort *Integer `json:"socketConnectPort,omitempty" quickfix:"SocketConnectPort"`
	SocketTimeout     *string  `json:"socketTimeout,omitempty" quickfix:"SocketTimeout"`
	ProxyType         *string  `json:"proxyType,omitempty" quickfix:"ProxyType"`
	ProxyHost         *string  `json:"proxyHost,omitempty" quickfix:"ProxyHost"`
	ProxyPort         *Integer `json:"proxyPort,omitempty" quickfix:"ProxyPort"`
	ProxyUser         *string  `json:"proxyUser,omitempty" quickfix:"ProxyUser"`
	ProxyPassword     *string  `json:"proxyPassword,omitempty" quickfix:"ProxyPassword"`

	SocketAcceptHost   *string  `json:"socketAcceptHost,omitempty" quickfix:"SocketAcceptHost"`
	SocketAcceptPort   *Integer `json:"socketAcceptPort,omitempty" quickfix:"SocketAcceptPort"`
	HeartBtIntOverride *Boolean `json:"heartBtIntOverride,omitempty" quickfix:"HeartBtIntOverride"`
	UseTCPProxy        *Boolean `json:"useTCPProxy,omitempty" quickfix:"UseTCPProxy"`
	DynamicSessions    *Boolean `json:"dynamicSessions,omitempty" quickfix:"DynamicSessions"`
	DynamicQualifier   *string  `json:"dynamicQualifier,omitempty" quickfix:"DynamicQualifier"`

	ResendRequestChunkSize       *Integer `json:"resendRequestChunkSize,omitempty" quickfix:"ResendRequestChunkSize"`
	EnableLastMsgSeqNumProcessed *Boolean `json:"enableLastMsgSeqNumProcessed,omitempty" quickfix:"EnableLastMsgSeqNumProcessed"`
	EnableNextExpectedMsgSeqNum  *Boolean `json:"enableNextExpectedMsgSeqNum,omitempty" quickfix:"EnableNextExpectedMsgSeqNum"`

	// AdditionalSettings supports custom settings, alternate SocketConnectHostN
	// entries, and version-qualified AppDataDictionary keys. Keys use the exact
	// QuickFIX setting name.
	AdditionalSettings map[string]Value `json:"additionalSettings,omitempty"`
}

// Settings contains global defaults and one or more FIX sessions.
type Settings struct {
	Global   *Session   `json:"global"`
	Sessions []*Session `json:"sessions"`
}

// ParseJSON decodes strict JSON settings from reader.
func ParseJSON(reader io.Reader) (*Settings, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var settings Settings
	if err := decoder.Decode(&settings); err != nil {
		return nil, fmt.Errorf("settings: decode JSON: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, err
	}
	if settings.Global == nil {
		return nil, errors.New("settings: global settings are required")
	}
	if err := settings.validate(); err != nil {
		return nil, err
	}
	return &settings, nil
}

// FromJSON decodes strict JSON settings from data.
func FromJSON(data []byte) (*Settings, error) {
	return ParseJSON(bytes.NewReader(data))
}

// ToJSON encodes settings as JSON.
func (s *Settings) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// ToQuickFIX converts typed JSON settings to the engine's Settings type.
func (s *Settings) ToQuickFIX() (*quickfix.Settings, error) {
	if s == nil || s.Global == nil {
		return nil, errors.New("settings: global settings are required")
	}
	if err := s.validate(); err != nil {
		return nil, err
	}

	result := quickfix.NewSettings()
	if err := applySession(result.GlobalSettings(), s.Global); err != nil {
		return nil, err
	}
	for index, session := range s.Sessions {
		if session == nil {
			return nil, fmt.Errorf("settings: session %d is null", index)
		}
		quickFIXSession := quickfix.NewSessionSettings()
		if err := applySession(quickFIXSession, session); err != nil {
			return nil, fmt.Errorf("settings: convert session %d: %w", index, err)
		}
		if _, err := result.AddSession(quickFIXSession); err != nil {
			return nil, fmt.Errorf("settings: add session %d: %w", index, err)
		}
	}
	return result, nil
}

func (s *Settings) validate() error {
	if len(s.Sessions) == 0 {
		return errors.New("settings: at least one session is required")
	}

	all := append([]*Session{s.Global}, s.Sessions...)
	for index, session := range all {
		if session == nil {
			continue
		}
		if session.ConnectionType != nil && !session.ConnectionType.valid() {
			return fmt.Errorf("settings: invalid connectionType %q at index %d", *session.ConnectionType, index)
		}
		for key := range session.AdditionalSettings {
			if strings.TrimSpace(key) == "" {
				return fmt.Errorf("settings: additional setting key cannot be empty at index %d", index)
			}
			if _, exists := knownSettingKeys[key]; exists {
				return fmt.Errorf("settings: additional setting %q duplicates a typed setting at index %d", key, index)
			}
		}
	}

	for index, session := range s.Sessions {
		if session == nil {
			continue
		}
		if session.ConnectionType == nil && s.Global.ConnectionType == nil {
			return fmt.Errorf("settings: connectionType is required for session %d", index)
		}
	}
	return nil
}

func applySession(target *quickfix.SessionSettings, source *Session) error {
	if err := applyStruct(target, reflect.ValueOf(source).Elem()); err != nil {
		return err
	}
	for key, value := range source.AdditionalSettings {
		target.Set(key, string(value))
	}
	return nil
}

func applyStruct(target *quickfix.SessionSettings, value reflect.Value) error {
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldType := valueType.Field(i)
		if fieldType.Anonymous {
			if err := applyStruct(target, field); err != nil {
				return err
			}
			continue
		}
		key := fieldType.Tag.Get("quickfix")
		if key == "" || field.Kind() != reflect.Pointer || field.IsNil() {
			continue
		}

		var text string
		switch setting := field.Elem().Interface().(type) {
		case string:
			text = setting
		case Integer:
			text = setting.quickFIXValue()
		case Boolean:
			text = setting.quickFIXValue()
		case ConnectionType:
			text = string(setting)
		default:
			return fmt.Errorf("settings: unsupported field type %s", field.Elem().Type())
		}
		target.Set(key, text)
	}
	return nil
}

var knownSettingKeys = collectSettingKeys(reflect.TypeOf(Session{}))

func collectSettingKeys(valueType reflect.Type) map[string]struct{} {
	keys := make(map[string]struct{})
	for i := 0; i < valueType.NumField(); i++ {
		field := valueType.Field(i)
		if field.Anonymous {
			for key := range collectSettingKeys(field.Type) {
				keys[key] = struct{}{}
			}
			continue
		}
		if key := field.Tag.Get("quickfix"); key != "" {
			keys[key] = struct{}{}
		}
	}
	return keys
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("settings: decode trailing JSON: %w", err)
	}
	return errors.New("settings: multiple JSON values are not allowed")
}
