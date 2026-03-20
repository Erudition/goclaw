package store

// PairingRequest represents a pending pairing code.
type PairingRequestData struct {
	Code      string            `json:"code"`
	SenderID  string            `json:"sender_id"`
	Channel   string            `json:"channel"`
	ChatID    string            `json:"chat_id"`
	AccountID string            `json:"account_id"`
	CreatedAt int64             `json:"created_at"`
	ExpiresAt int64             `json:"expires_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// PairedDeviceData represents an approved pairing.
type PairedDeviceData struct {
	SenderID string            `json:"sender_id"`
	Channel  string            `json:"channel"`
	ChatID   string            `json:"chat_id"`
	PairedAt int64             `json:"paired_at"`
	PairedBy string            `json:"paired_by"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// PairingStore manages device pairing.
type PairingStore interface {
	RequestPairing(senderID, channel, chatID, accountID string, metadata map[string]string) (string, error)
	ApprovePairing(code, approvedBy string) (*PairedDeviceData, error)
	DenyPairing(code string) error
	RevokePairing(senderID, channel string) error
	IsPaired(senderID, channel string) (bool, error)
	ListPending() []PairingRequestData
	ListPaired() []PairedDeviceData
}
