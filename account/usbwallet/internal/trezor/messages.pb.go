// Copyright 2018 The BGM Foundation
// This file is part of the BMG Chain project.
//
//
//
// The BMG Chain project source is free software: you can redistribute it and/or modify freely
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later versions.
//
//
//
// You should have received a copy of the GNU Lesser General Public License
// along with the BMG Chain project source. If not, you can see <http://www.gnu.org/licenses/> for detail.
package trezor

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"


func (mPtr *Features) GetBootloaderMode() bool {
	if m != nil && mPtr.BootloaderMode != nil {
		return *mPtr.BootloaderMode
	}
	return false
}

func (mPtr *Features) GetDeviceId() string {
	if m != nil && mPtr.DeviceId != nil {
		return *mPtr.DeviceId
	}
	return ""
}

func (mPtr *Features) GetVendor() string {
	if m != nil && mPtr.Vendor != nil {
		return *mPtr.Vendor
	}
	return ""
}

func (mPtr *Features) GetMajorVersion() uint32 {
	if m != nil && mPtr.MajorVersion != nil {
		return *mPtr.MajorVersion
	}
	return 0
}

func (mPtr *Features) GetPinProtection() bool {
	if m != nil && mPtr.PinProtection != nil {
		return *mPtr.PinProtection
	}
	return false
}

func (mPtr *Features) GetLabel() string {
	if m != nil && mPtr.Label != nil {
		return *mPtr.Label
	}
	return ""
}

func (mPtr *Features) GetCoins() []*CoinsType {
	if m != nil {
		return mPtr.Coins
	}
	return nil
}

func (mPtr *Features) GetInitialized() bool {
	if m != nil && mPtr.Initialized != nil {
		return *mPtr.Initialized
	}
	return false
}

func (mPtr *Features) GetRevision() []byte {
	if m != nil {
		return mPtr.Revision
	}
	return nil
}

func (mPtr *Features) GetBootloaderHash() []byte {
	if m != nil {
		return mPtr.BootloaderHash
	}
	return nil
}

func (mPtr *Features) GetPassphraseProtection() bool {
	if m != nil && mPtr.PassphraseProtection != nil {
		return *mPtr.PassphraseProtection
	}
	return false
}

func (mPtr *Features) GetLanguage() string {
	if m != nil && mPtr.Language != nil {
		return *mPtr.Language
	}
	return ""
}

func (mPtr *Features) GetImported() bool {
	if m != nil && mPtr.Imported != nil {
		return *mPtr.Imported
	}
	return false
}


func (mPtr *Features) GetPassphraseCached() bool {
	if m != nil && mPtr.PassphraseCached != nil {
		return *mPtr.PassphraseCached
	}
	return false
}

func (mPtr *Features) GetFirmwarePresent() bool {
	if m != nil && mPtr.FirmwarePresent != nil {
		return *mPtr.FirmwarePresent
	}
	return false
}

func (mPtr *Features) GetNeedsBackup() bool {
	if m != nil && mPtr.NeedsBackup != nil {
		return *mPtr.NeedsBackup
	}
	return false
}

func (mPtr *Features) GetPinCached() bool {
	if m != nil && mPtr.PinCached != nil {
		return *mPtr.PinCached
	}
	return false
}
func (mPtr *Features) GetFlags() uint32 {
	if m != nil && mPtr.Flags != nil {
		return *mPtr.Flags
	}
	return 0
}

// *
// Request: clear session (removes cached PIN, passphrase, etc).
// @next Success
type ClearSession struct {
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *ClearSession) Reset()                    { *m = ClearSession{} }
func (mPtr *ClearSession) String() string            { return proto.CompactTextString(m) }
func (*ClearSession) ProtoMessage()               {}
func (*ClearSession) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{3} }

// *
// Request: change language and/or label of the device
// @next Success
// @next Failure
// @next ButtonRequest
// @next PinMatrixRequest
type ApplySettings struct {
	Language         *string `protobuf:"bytes,1,opt,name=language" json:"language,omitempty"`
	Label            *string `protobuf:"bytes,2,opt,name=label" json:"label,omitempty"`
	UsePassphrase    *bool   `protobuf:"varint,3,opt,name=use_passphrase,json=usePassphrase" json:"use_passphrase,omitempty"`
	Homescreen       []byte  `protobuf:"bytes,4,opt,name=homescreen" json:"homescreen,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *ApplySettings) Reset()                    { *m = ApplySettings{} }
func (mPtr *ApplySettings) String() string            { return proto.CompactTextString(m) }
func (*ApplySettings) ProtoMessage()               {}
func (*ApplySettings) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{4} }


func (mPtr *ApplySettings) GetLabel() string {
	if m != nil && mPtr.Label != nil {
		return *mPtr.Label
	}
	return ""
}

func (mPtr *ApplySettings) GetUsePassphrase() bool {
	if m != nil && mPtr.UsePassphrase != nil {
		return *mPtr.UsePassphrase
	}
	return false
}

func (mPtr *ApplySettings) GetHomescreen() []byte {
	if m != nil {
		return mPtr.Homescreen
	}
	return nil
}

func (mPtr *ApplySettings) GetLanguage() string {
	if m != nil && mPtr.Language != nil {
		return *mPtr.Language
	}
	return ""
}
// *
// Request: set flags of the device
// @next Success
// @next Failure
type ApplyFlags struct {
	Flags            *uint32 `protobuf:"varint,1,opt,name=flags" json:"flags,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *ApplyFlags) Reset()                    { *m = ApplyFlags{} }
func (mPtr *ApplyFlags) String() string            { return proto.CompactTextString(m) }
func (*ApplyFlags) ProtoMessage()               {}
func (*ApplyFlags) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{5} }

// *
// Request: Starts workflow for setting/changing/removing the PIN
// @next ButtonRequest
// @next PinMatrixRequest
type ChangePin struct {
	Remove           *bool  `protobuf:"varint,1,opt,name=remove" json:"remove,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *ChangePin) Reset()                    { *m = ChangePin{} }
func (mPtr *ChangePin) String() string            { return proto.CompactTextString(m) }
func (*ChangePin) ProtoMessage()               {}
func (*ChangePin) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{6} }

func (mPtr *ChangePin) GetRemove() bool {
	if m != nil && mPtr.Remove != nil {
		return *mPtr.Remove
	}
	return false
}

func (mPtr *ApplyFlags) GetFlags() uint32 {
	if m != nil && mPtr.Flags != nil {
		return *mPtr.Flags
	}
	return 0
}

// *
// Request: Test if the device is alive, device sends back the message in Success response
// @next Success
type Ping struct {
	Message              *string `protobuf:"bytes,1,opt,name=message" json:"message,omitempty"`
	ButtonProtection     *bool   `protobuf:"varint,2,opt,name=button_protection,json=buttonProtection" json:"button_protection,omitempty"`
	PinProtection        *bool   `protobuf:"varint,3,opt,name=pin_protection,json=pinProtection" json:"pin_protection,omitempty"`
	PassphraseProtection *bool   `protobuf:"varint,4,opt,name=passphrase_protection,json=passphraseProtection" json:"passphrase_protection,omitempty"`
	XXX_unrecognized     []byte  `json:"-"`
}

func (mPtr *Ping) GetMessage() string {
	if m != nil && mPtr.Message != nil {
		return *mPtr.Message
	}
	return ""
}

func (mPtr *Ping) GetButtonProtection() bool {
	if m != nil && mPtr.ButtonProtection != nil {
		return *mPtr.ButtonProtection
	}
	return false
}

func (mPtr *Ping) GetPinProtection() bool {
	if m != nil && mPtr.PinProtection != nil {
		return *mPtr.PinProtection
	}
	return false
}

func (mPtr *Ping) Reset()                    { *m = Ping{} }
func (mPtr *Ping) String() string            { return proto.CompactTextString(m) }
func (*Ping) ProtoMessage()               {}
func (*Ping) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{7} }

func (mPtr *Ping) GetPassphraseProtection() bool {
	if m != nil && mPtr.PassphraseProtection != nil {
		return *mPtr.PassphraseProtection
	}
	return false
}

// *
// Response: Success of the previous request
type Success struct {
	Message          *string `protobuf:"bytes,1,opt,name=message" json:"message,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *Success) String() string            { return proto.CompactTextString(m) }
func (*Success) ProtoMessage()               {}
func (*Success) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{8} }

func (mPtr *Success) GetMessage() string {
	if m != nil && mPtr.Message != nil {
		return *mPtr.Message
	}
	return ""
}

func (mPtr *Success) Reset()                    { *m = Success{} }
// *
// Response: Failure of the previous request
type Failure struct {
	Code             *FailuresType `protobuf:"varint,1,opt,name=code,enum=FailuresType" json:"code,omitempty"`
	Message          *string      `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
	XXX_unrecognized []byte       `json:"-"`
}

func (mPtr *Failure) Reset()                    { *m = Failure{} }
func (mPtr *Failure) String() string            { return proto.CompactTextString(m) }

func (mPtr *Failure) GetCode() FailuresType {
	if m != nil && mPtr.Code != nil {
		return *mPtr.Code
	}
	return FailuresType_Failures_UnexpectedMessage
}

func (mPtr *Failure) GetMessage() string {
	if m != nil && mPtr.Message != nil {
		return *mPtr.Message
	}
	return ""
}

// *
// Response: Device is waiting for HW button press.
// @next ButtonAck
// @next Cancel
type ButtonRequest struct {
	Code             *ButtonRequestType `protobuf:"varint,1,opt,name=code,enum=ButtonRequestType" json:"code,omitempty"`
	Data             *string            `protobuf:"bytes,2,opt,name=data" json:"data,omitempty"`
	XXX_unrecognized []byte             `json:"-"`
}
func (*Failure) ProtoMessage()               {}
func (*Failure) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{9} }

func (mPtr *ButtonRequest) Reset()                    { *m = ButtonRequest{} }
func (mPtr *ButtonRequest) String() string            { return proto.CompactTextString(m) }
func (*ButtonRequest) ProtoMessage()               {}
func (*ButtonRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{10} }

func (mPtr *ButtonRequest) GetCode() ButtonRequestType {
	if m != nil && mPtr.Code != nil {
		return *mPtr.Code
	}
	return ButtonRequestType_ButtonRequest_Other
}

func (mPtr *ButtonRequest) GetData() string {
	if m != nil && mPtr.Data != nil {
		return *mPtr.Data
	}
	return ""
}

// *
// Request: Computer agrees to wait for HW button press
// @prev ButtonRequest
type ButtonAck struct {
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *ButtonAck) Reset()                    { *m = ButtonAck{} }
func (mPtr *ButtonAck) String() string            { return proto.CompactTextString(m) }
func (*ButtonAck) ProtoMessage()               {}
func (*ButtonAck) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{11} }

// *
// Response: Device is asking computer to show PIN matrix and awaits PIN encoded using this matrix scheme
// @next PinMatrixAck
// @next Cancel
type PinMatrixRequest struct {
	Type             *PinMatrixRequestType `protobuf:"varint,1,opt,name=type,enum=PinMatrixRequestType" json:"type,omitempty"`
	XXX_unrecognized []byte                `json:"-"`
}
func (*PinMatrixRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{12} }

func (mPtr *PinMatrixRequest) GetType() PinMatrixRequestType {
	if m != nil && mPtr.Type != nil {
		return *mPtr.Type
	}
	return PinMatrixRequestType_PinMatrixRequestType_Current
}

func (mPtr *PinMatrixRequest) Reset()                    { *m = PinMatrixRequest{} }
func (mPtr *PinMatrixRequest) String() string            { return proto.CompactTextString(m) }
func (*PinMatrixRequest) ProtoMessage()               {}

// *
// Request: Computer responds with encoded PIN
// @prev PinMatrixRequest
type PinMatrixAck struct {
	Pin              *string `protobuf:"bytes,1,req,name=pin" json:"pin,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *PinMatrixAck) Reset()                    { *m = PinMatrixAck{} }
func (mPtr *PinMatrixAck) String() string            { return proto.CompactTextString(m) }

func (mPtr *PinMatrixAck) GetPin() string {
	if m != nil && mPtr.Pin != nil {
		return *mPtr.Pin
	}
	return ""
}

// *
// Request: Abort last operation that required user interaction
// @prev ButtonRequest
// @prev PinMatrixRequest
// @prev PassphraseRequest
type Cancel struct {
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *Cancel) Reset()                    { *m = Cancel{} }
func (mPtr *Cancel) String() string            { return proto.CompactTextString(m) }
func (*Cancel) ProtoMessage()               {}
func (*Cancel) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{14} }
func (*PinMatrixAck) ProtoMessage()               {}
func (*PinMatrixAck) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{13} }

// *
// Response: Device awaits encryption passphrase
// @next PassphraseAck
// @next Cancel
type PassphraseRequest struct {
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *PassphraseRequest) Reset()                    { *m = PassphraseRequest{} }
func (mPtr *PassphraseRequest) String() string            { return proto.CompactTextString(m) }
func (*PassphraseRequest) ProtoMessage()               {}
func (*PassphraseRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{15} }

// *
// Request: Send passphrase back
// @prev PassphraseRequest
type PassphraseAck struct {
	Passphrase       *string `protobuf:"bytes,1,req,name=passphrase" json:"passphrase,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *PassphraseAck) Reset()                    { *m = PassphraseAck{} }
func (mPtr *PassphraseAck) String() string            { return proto.CompactTextString(m) }
func (*PassphraseAck) ProtoMessage()               {}
func (*PassphraseAck) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{16} }

func (mPtr *PassphraseAck) GetPassphrase() string {
	if m != nil && mPtr.Passphrase != nil {
		return *mPtr.Passphrase
	}
	return ""
}

// *
// Request: Request a sample of random data generated by hardware RNG. May be used for testing.
// @next ButtonRequest
// @next Entropy
// @next Failure
type GetEntropy struct {
	Size             *uint32 `protobuf:"varint,1,req,name=size" json:"size,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *GetEntropy) Reset()                    { *m = GetEntropy{} }
func (mPtr *GetEntropy) String() string            { return proto.CompactTextString(m) }
func (*GetEntropy) ProtoMessage()               {}
func (*GetEntropy) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{17} }

func (mPtr *GetEntropy) GetSize() uint32 {
	if m != nil && mPtr.Size != nil {
		return *mPtr.Size
	}
	return 0
}

// *
// Response: Reply with random data generated by internal RNG
// @prev GetEntropy
type Entropy struct {
	Entropy          []byte `protobuf:"bytes,1,req,name=entropy" json:"entropy,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *Entropy) Reset()                    { *m = Entropy{} }
func (mPtr *Entropy) String() string            { return proto.CompactTextString(m) }
func (*Entropy) ProtoMessage()               {}
func (*Entropy) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{18} }

func (mPtr *Entropy) GetEntropy() []byte {
	if m != nil {
		return mPtr.Entropy
	}
	return nil
}

// *
// Request: Ask device for public key corresponding to address_n path
// @next PassphraseRequest
// @next PublicKey
// @next Failure
type GetPublicKey struct {
	AddressN         []uint32 `protobuf:"varint,1,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	EcdsaCurveName   *string  `protobuf:"bytes,2,opt,name=ecdsa_curve_name,json=ecdsaCurveName" json:"ecdsa_curve_name,omitempty"`
	ShowDisplay      *bool    `protobuf:"varint,3,opt,name=show_display,json=showDisplay" json:"show_display,omitempty"`
	CoinName         *string  `protobuf:"bytes,4,opt,name=coin_name,json=coinName,def=Bitcoin" json:"coin_name,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (mPtr *GetPublicKey) Reset()                    { *m = GetPublicKey{} }
func (mPtr *GetPublicKey) String() string            { return proto.CompactTextString(m) }
func (*GetPublicKey) ProtoMessage()               {}
func (*GetPublicKey) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{19} }

const Default_GetPublicKey_CoinName string = "Bitcoin"

func (mPtr *GetPublicKey) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

func (mPtr *GetPublicKey) GetEcdsaCurveName() string {
	if m != nil && mPtr.EcdsaCurveName != nil {
		return *mPtr.EcdsaCurveName
	}
	return ""
}

func (mPtr *GetPublicKey) GetShowDisplay() bool {
	if m != nil && mPtr.ShowDisplay != nil {
		return *mPtr.ShowDisplay
	}
	return false
}

func (mPtr *GetPublicKey) GetCoinName() string {
	if m != nil && mPtr.CoinName != nil {
		return *mPtr.CoinName
	}
	return Default_GetPublicKey_CoinName
}

// *
// Response: Contains public key derived from device private seed
// @prev GetPublicKey
type PublicKey struct {
	Node             *HDNodeType `protobuf:"bytes,1,req,name=node" json:"node,omitempty"`
	Xpub             *string     `protobuf:"bytes,2,opt,name=xpub" json:"xpub,omitempty"`
	XXX_unrecognized []byte      `json:"-"`
}


func (*PublicKey) ProtoMessage()               {}
func (*PublicKey) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{20} }

func (mPtr *PublicKey) GetNode() *HDNodeType {
	if m != nil {
		return mPtr.Node
	}
	return nil
}
func (mPtr *PublicKey) Reset()                    { *m = PublicKey{} }
func (mPtr *PublicKey) String() string            { return proto.CompactTextString(m) }
func (mPtr *PublicKey) GetXpub() string {
	if m != nil && mPtr.Xpub != nil {
		return *mPtr.Xpub
	}
	return ""
}

// *
// Request: Ask device for address corresponding to address_n path
// @next PassphraseRequest
// @next Address
// @next Failure
type GetAddress struct {
	AddressN         []uint32                  `protobuf:"varint,1,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	CoinName         *string                   `protobuf:"bytes,2,opt,name=coin_name,json=coinName,def=Bitcoin" json:"coin_name,omitempty"`
	ShowDisplay      *bool                     `protobuf:"varint,3,opt,name=show_display,json=showDisplay" json:"show_display,omitempty"`
	Multisig         *MultisigRedeemScriptTypes `protobuf:"bytes,4,opt,name=multisig" json:"multisig,omitempty"`
	ScriptType       *InputScriptType          `protobuf:"varint,5,opt,name=script_type,json=scriptType,enum=InputScriptType,def=0" json:"script_type,omitempty"`
	XXX_unrecognized []byte                    `json:"-"`
}

func (mPtr *GetAddress) Reset()                    { *m = GetAddress{} }
func (mPtr *GetAddress) String() string            { return proto.CompactTextString(m) }
func (*GetAddress) ProtoMessage()               {}
func (*GetAddress) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{21} }

const Default_GetAddress_CoinName string = "Bitcoin"
const Default_GetAddress_ScriptType InputScriptType = InputScriptType_SPENDADDRESS

func (mPtr *GetAddress) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

func (mPtr *GetAddress) GetCoinName() string {
	if m != nil && mPtr.CoinName != nil {
		return *mPtr.CoinName
	}
	return Default_GetAddress_CoinName
}

func (mPtr *GetAddress) GetShowDisplay() bool {
	if m != nil && mPtr.ShowDisplay != nil {
		return *mPtr.ShowDisplay
	}
	return false
}

func (mPtr *GetAddress) GetMultisig() *MultisigRedeemScriptTypes {
	if m != nil {
		return mPtr.Multisig
	}
	return nil
}

func (mPtr *GetAddress) GetScriptType() InputScriptType {
	if m != nil && mPtr.ScriptType != nil {
		return *mPtr.ScriptType
	}
	return Default_GetAddress_ScriptType
}

// *
// Request: Ask device for Bgmchain address corresponding to address_n path
// @next PassphraseRequest
// @next BgmchainAddress
// @next Failure
type BgmchainGetAddress struct {
	AddressN         []uint32 `protobuf:"varint,1,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	ShowDisplay      *bool    `protobuf:"varint,2,opt,name=show_display,json=showDisplay" json:"show_display,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (mPtr *BgmchainGetAddress) Reset()                    { *m = BgmchainGetAddress{} }
func (mPtr *BgmchainGetAddress) String() string            { return proto.CompactTextString(m) }
func (*BgmchainGetAddress) ProtoMessage()               {}
func (*BgmchainGetAddress) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{22} }

func (mPtr *BgmchainGetAddress) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

func (mPtr *BgmchainGetAddress) GetShowDisplay() bool {
	if m != nil && mPtr.ShowDisplay != nil {
		return *mPtr.ShowDisplay
	}
	return false
}

// *
// Response: Contains address derived from device private seed
// @prev GetAddress
type Address struct {
	Address          *string `protobuf:"bytes,1,req,name=address" json:"address,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *Address) Reset()                    { *m = Address{} }
func (mPtr *Address) String() string            { return proto.CompactTextString(m) }
func (*Address) ProtoMessage()               {}
func (*Address) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{23} }

func (mPtr *Address) GetAddress() string {
	if m != nil && mPtr.Address != nil {
		return *mPtr.Address
	}
	return ""
}

// *
// Response: Contains an Bgmchain address derived from device private seed
// @prev BgmchainGetAddress
type BgmchainAddress struct {
	Address          []byte `protobuf:"bytes,1,req,name=address" json:"address,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *BgmchainAddress) Reset()                    { *m = BgmchainAddress{} }
func (mPtr *BgmchainAddress) String() string            { return proto.CompactTextString(m) }
func (*BgmchainAddress) ProtoMessage()               {}
func (*BgmchainAddress) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{24} }

func (mPtr *BgmchainAddress) GetAddress() []byte {
	if m != nil {
		return mPtr.Address
	}
	return nil
}

// *
// Request: Request device to wipe all sensitive data and settings
// @next ButtonRequest
type WipeDevice struct {
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *WipeDevice) Reset()                    { *m = WipeDevice{} }
func (mPtr *WipeDevice) String() string            { return proto.CompactTextString(m) }
func (*WipeDevice) ProtoMessage()               {}
func (*WipeDevice) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{25} }

// *
// Request: Load seed and related internal settings from the computer
// @next ButtonRequest
// @next Success
// @next Failure
type LoadDevice struct {
	Mnemonic             *string     `protobuf:"bytes,1,opt,name=mnemonic" json:"mnemonic,omitempty"`
	Node                 *HDNodeType `protobuf:"bytes,2,opt,name=node" json:"node,omitempty"`
	Pin                  *string     `protobuf:"bytes,3,opt,name=pin" json:"pin,omitempty"`
	PassphraseProtection *bool       `protobuf:"varint,4,opt,name=passphrase_protection,json=passphraseProtection" json:"passphrase_protection,omitempty"`
	Language             *string     `protobuf:"bytes,5,opt,name=language,def=english" json:"language,omitempty"`
	Label                *string     `protobuf:"bytes,6,opt,name=label" json:"label,omitempty"`
	SkipChecksum         *bool       `protobuf:"varint,7,opt,name=skip_checksum,json=skipChecksum" json:"skip_checksum,omitempty"`
	U2FCounter           *uint32     `protobuf:"varint,8,opt,name=u2f_counter,json=u2fCounter" json:"u2f_counter,omitempty"`
	XXX_unrecognized     []byte      `json:"-"`
}

func (mPtr *LoadDevice) Reset()                    { *m = LoadDevice{} }
func (mPtr *LoadDevice) String() string            { return proto.CompactTextString(m) }

const Default_LoadDevice_Language string = "english"

func (mPtr *LoadDevice) GetMnemonic() string {
	if m != nil && mPtr.Mnemonic != nil {
		return *mPtr.Mnemonic
	}
	return ""
}

func (mPtr *LoadDevice) GetNode() *HDNodeType {
	if m != nil {
		return mPtr.Node
	}
	return nil
}

func (mPtr *LoadDevice) GetPin() string {
	if m != nil && mPtr.Pin != nil {
		return *mPtr.Pin
	}
	return ""
}
func (*LoadDevice) ProtoMessage()               {}
func (*LoadDevice) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{26} }

func (mPtr *LoadDevice) GetPassphraseProtection() bool {
	if m != nil && mPtr.PassphraseProtection != nil {
		return *mPtr.PassphraseProtection
	}
	return false
}

func (mPtr *LoadDevice) GetLanguage() string {
	if m != nil && mPtr.Language != nil {
		return *mPtr.Language
	}
	return Default_LoadDevice_Language
}

func (mPtr *LoadDevice) GetLabel() string {
	if m != nil && mPtr.Label != nil {
		return *mPtr.Label
	}
	return ""
}

func (mPtr *LoadDevice) GetSkipChecksum() bool {
	if m != nil && mPtr.SkipChecksum != nil {
		return *mPtr.SkipChecksum
	}
	return false
}

func (mPtr *LoadDevice) GetU2FCounter() uint32 {
	if m != nil && mPtr.U2FCounter != nil {
		return *mPtr.U2FCounter
	}
	return 0
}

// *
// Request: Ask device to do initialization involving user interaction
// @next EntropyRequest
// @next Failure
type ResetDevice struct {
	DisplayRandom        *bool   `protobuf:"varint,1,opt,name=display_random,json=displayRandom" json:"display_random,omitempty"`
	Strength             *uint32 `protobuf:"varint,2,opt,name=strength,def=256" json:"strength,omitempty"`
	PassphraseProtection *bool   `protobuf:"varint,3,opt,name=passphrase_protection,json=passphraseProtection" json:"passphrase_protection,omitempty"`
	PinProtection        *bool   `protobuf:"varint,4,opt,name=pin_protection,json=pinProtection" json:"pin_protection,omitempty"`
	Language             *string `protobuf:"bytes,5,opt,name=language,def=english" json:"language,omitempty"`
	Label                *string `protobuf:"bytes,6,opt,name=label" json:"label,omitempty"`
	U2FCounter           *uint32 `protobuf:"varint,7,opt,name=u2f_counter,json=u2fCounter" json:"u2f_counter,omitempty"`
	SkipBackup           *bool   `protobuf:"varint,8,opt,name=skip_backup,json=skipBackup" json:"skip_backup,omitempty"`
	XXX_unrecognized     []byte  `json:"-"`
}

func (mPtr *ResetDevice) Reset()                    { *m = ResetDevice{} }
func (mPtr *ResetDevice) String() string            { return proto.CompactTextString(m) }
func (*ResetDevice) ProtoMessage()               {}
func (*ResetDevice) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{27} }

const Default_ResetDevice_Strength uint32 = 256
const Default_ResetDevice_Language string = "english"

func (mPtr *ResetDevice) GetDisplayRandom() bool {
	if m != nil && mPtr.DisplayRandom != nil {
		return *mPtr.DisplayRandom
	}
	return false
}

func (mPtr *ResetDevice) GetStrength() uint32 {
	if m != nil && mPtr.Strength != nil {
		return *mPtr.Strength
	}
	return Default_ResetDevice_Strength
}


func (mPtr *ResetDevice) GetPinProtection() bool {
	if m != nil && mPtr.PinProtection != nil {
		return *mPtr.PinProtection
	}
	return false
}

func (mPtr *ResetDevice) GetLanguage() string {
	if m != nil && mPtr.Language != nil {
		return *mPtr.Language
	}
	return Default_ResetDevice_Language
}

func (mPtr *ResetDevice) GetLabel() string {
	if m != nil && mPtr.Label != nil {
		return *mPtr.Label
	}
	return ""
}

func (mPtr *ResetDevice) GetPassphraseProtection() bool {
	if m != nil && mPtr.PassphraseProtection != nil {
		return *mPtr.PassphraseProtection
	}
	return false
}
func (mPtr *ResetDevice) GetU2FCounter() uint32 {
	if m != nil && mPtr.U2FCounter != nil {
		return *mPtr.U2FCounter
	}
	return 0
}

func (mPtr *ResetDevice) GetSkipBackup() bool {
	if m != nil && mPtr.SkipBackup != nil {
		return *mPtr.SkipBackup
	}
	return false
}

// *
// Request: Perform backup of the device seed if not backed up using ResetDevice
// @next ButtonRequest
type BackupDevice struct {
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *BackupDevice) Reset()                    { *m = BackupDevice{} }
func (mPtr *BackupDevice) String() string            { return proto.CompactTextString(m) }
func (*BackupDevice) ProtoMessage()               {}
func (*BackupDevice) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{28} }

// *
// Response: Ask for additional entropy from host computer
// @prev ResetDevice
// @next EntropyAck
type EntropyRequest struct {
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *EntropyRequest) Reset()                    { *m = EntropyRequest{} }
func (mPtr *EntropyRequest) String() string            { return proto.CompactTextString(m) }
func (*EntropyRequest) ProtoMessage()               {}
func (*EntropyRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{29} }

// *
// Request: Provide additional entropy for seed generation function
// @prev EntropyRequest
// @next ButtonRequest
type EntropyAck struct {
	Entropy          []byte `protobuf:"bytes,1,opt,name=entropy" json:"entropy,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *EntropyAck) Reset()                    { *m = EntropyAck{} }
func (mPtr *EntropyAck) String() string            { return proto.CompactTextString(m) }
func (*EntropyAck) ProtoMessage()               {}
func (*EntropyAck) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{30} }

func (mPtr *EntropyAck) GetEntropy() []byte {
	if m != nil {
		return mPtr.Entropy
	}
	return nil
}

// *
// Request: Start recovery workflow asking user for specific words of mnemonic
// Used to recovery device safely even on untrusted computer.
// @next WordRequest
type RecoveryDevice struct {
	WordCount            *uint32 `protobuf:"varint,1,opt,name=word_count,json=wordCount" json:"word_count,omitempty"`
	PassphraseProtection *bool   `protobuf:"varint,2,opt,name=passphrase_protection,json=passphraseProtection" json:"passphrase_protection,omitempty"`
	PinProtection        *bool   `protobuf:"varint,3,opt,name=pin_protection,json=pinProtection" json:"pin_protection,omitempty"`
	Language             *string `protobuf:"bytes,4,opt,name=language,def=english" json:"language,omitempty"`
	Label                *string `protobuf:"bytes,5,opt,name=label" json:"label,omitempty"`
	EnforceWordlist      *bool   `protobuf:"varint,6,opt,name=enforce_wordlist,json=enforceWordlist" json:"enforce_wordlist,omitempty"`
	// 7 reserved for unused recovery method
	Type             *uint32 `protobuf:"varint,8,opt,name=type" json:"type,omitempty"`
	U2FCounter       *uint32 `protobuf:"varint,9,opt,name=u2f_counter,json=u2fCounter" json:"u2f_counter,omitempty"`
	DryRun           *bool   `protobuf:"varint,10,opt,name=dry_run,json=dryRun" json:"dry_run,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *RecoveryDevice) Reset()                    { *m = RecoveryDevice{} }
func (mPtr *RecoveryDevice) String() string            { return proto.CompactTextString(m) }
func (*RecoveryDevice) ProtoMessage()               {}
func (*RecoveryDevice) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{31} }

const Default_RecoveryDevice_Language string = "english"

func (mPtr *RecoveryDevice) GetWordCount() uint32 {
	if m != nil && mPtr.WordCount != nil {
		return *mPtr.WordCount
	}
	return 0
}

func (mPtr *RecoveryDevice) GetPassphraseProtection() bool {
	if m != nil && mPtr.PassphraseProtection != nil {
		return *mPtr.PassphraseProtection
	}
	return false
}

func (mPtr *RecoveryDevice) GetPinProtection() bool {
	if m != nil && mPtr.PinProtection != nil {
		return *mPtr.PinProtection
	}
	return false
}

func (mPtr *RecoveryDevice) GetLanguage() string {
	if m != nil && mPtr.Language != nil {
		return *mPtr.Language
	}
	return Default_RecoveryDevice_Language
}

func (mPtr *RecoveryDevice) GetLabel() string {
	if m != nil && mPtr.Label != nil {
		return *mPtr.Label
	}
	return ""
}

func (mPtr *RecoveryDevice) GetEnforceWordlist() bool {
	if m != nil && mPtr.EnforceWordlist != nil {
		return *mPtr.EnforceWordlist
	}
	return false
}

func (mPtr *RecoveryDevice) GetType() uint32 {
	if m != nil && mPtr.Type != nil {
		return *mPtr.Type
	}
	return 0
}

func (mPtr *RecoveryDevice) GetU2FCounter() uint32 {
	if m != nil && mPtr.U2FCounter != nil {
		return *mPtr.U2FCounter
	}
	return 0
}

func (mPtr *RecoveryDevice) GetDryRun() bool {
	if m != nil && mPtr.DryRun != nil {
		return *mPtr.DryRun
	}
	return false
}

// *
// Response: Device is waiting for user to enter word of the mnemonic
// Its position is shown only on device's internal display.
// @prev RecoveryDevice
// @prev WordAck
type WordRequest struct {
	Type             *WordRequestType `protobuf:"varint,1,opt,name=type,enum=WordRequestType" json:"type,omitempty"`
	XXX_unrecognized []byte           `json:"-"`
}

func (mPtr *WordRequest) Reset()                    { *m = WordRequest{} }
func (mPtr *WordRequest) String() string            { return proto.CompactTextString(m) }
func (*WordRequest) ProtoMessage()               {}
func (*WordRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{32} }

func (mPtr *WordRequest) GetType() WordRequestType {
	if m != nil && mPtr.Type != nil {
		return *mPtr.Type
	}
	return WordRequestType_WordRequestType_Plain
}

// *
// Request: Computer replies with word from the mnemonic
// @prev WordRequest
// @next WordRequest
// @next Success
// @next Failure
type WordAck struct {
	Word             *string `protobuf:"bytes,1,req,name=word" json:"word,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *WordAck) Reset()                    { *m = WordAck{} }
func (mPtr *WordAck) String() string            { return proto.CompactTextString(m) }
func (*WordAck) ProtoMessage()               {}
func (*WordAck) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{33} }

func (mPtr *WordAck) GetWord() string {
	if m != nil && mPtr.Word != nil {
		return *mPtr.Word
	}
	return ""
}

// *
// Request: Ask device to sign message
// @next MessageSignature
// @next Failure
type SignMessage struct {
	AddressN         []uint32         `protobuf:"varint,1,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	Message          []byte           `protobuf:"bytes,2,req,name=message" json:"message,omitempty"`
	CoinName         *string          `protobuf:"bytes,3,opt,name=coin_name,json=coinName,def=Bitcoin" json:"coin_name,omitempty"`
	ScriptType       *InputScriptType `protobuf:"varint,4,opt,name=script_type,json=scriptType,enum=InputScriptType,def=0" json:"script_type,omitempty"`
	XXX_unrecognized []byte           `json:"-"`
}

func (mPtr *SignMessage) Reset()                    { *m = SignMessage{} }
func (mPtr *SignMessage) String() string            { return proto.CompactTextString(m) }
func (*SignMessage) ProtoMessage()               {}
func (*SignMessage) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{34} }

const Default_SignMessage_CoinName string = "Bitcoin"
const Default_SignMessage_ScriptType InputScriptType = InputScriptType_SPENDADDRESS

func (mPtr *SignMessage) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

func (mPtr *SignMessage) GetMessage() []byte {
	if m != nil {
		return mPtr.Message
	}
	return nil
}

func (mPtr *SignMessage) GetCoinName() string {
	if m != nil && mPtr.CoinName != nil {
		return *mPtr.CoinName
	}
	return Default_SignMessage_CoinName
}

func (mPtr *SignMessage) GetScriptType() InputScriptType {
	if m != nil && mPtr.ScriptType != nil {
		return *mPtr.ScriptType
	}
	return Default_SignMessage_ScriptType
}

// *
// Request: Ask device to verify message
// @next Success
// @next Failure
type VerifyMessage struct {
	Address          *string `protobuf:"bytes,1,opt,name=address" json:"address,omitempty"`
	Signature        []byte  `protobuf:"bytes,2,opt,name=signature" json:"signature,omitempty"`
	Message          []byte  `protobuf:"bytes,3,opt,name=message" json:"message,omitempty"`
	CoinName         *string `protobuf:"bytes,4,opt,name=coin_name,json=coinName,def=Bitcoin" json:"coin_name,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *VerifyMessage) Reset()                    { *m = VerifyMessage{} }
func (mPtr *VerifyMessage) String() string            { return proto.CompactTextString(m) }
func (*VerifyMessage) ProtoMessage()               {}
func (*VerifyMessage) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{35} }

const Default_VerifyMessage_CoinName string = "Bitcoin"

func (mPtr *VerifyMessage) GetAddress() string {
	if m != nil && mPtr.Address != nil {
		return *mPtr.Address
	}
	return ""
}

func (mPtr *VerifyMessage) GetSignature() []byte {
	if m != nil {
		return mPtr.Signature
	}
	return nil
}

func (mPtr *VerifyMessage) GetMessage() []byte {
	if m != nil {
		return mPtr.Message
	}
	return nil
}

func (mPtr *VerifyMessage) GetCoinName() string {
	if m != nil && mPtr.CoinName != nil {
		return *mPtr.CoinName
	}
	return Default_VerifyMessage_CoinName
}

// *
// Response: Signed message
// @prev SignMessage
type MessageSignature struct {
	Address          *string `protobuf:"bytes,1,opt,name=address" json:"address,omitempty"`
	Signature        []byte  `protobuf:"bytes,2,opt,name=signature" json:"signature,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *MessageSignature) Reset()                    { *m = MessageSignature{} }
func (mPtr *MessageSignature) String() string            { return proto.CompactTextString(m) }
func (*MessageSignature) ProtoMessage()               {}
func (*MessageSignature) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{36} }

func (mPtr *MessageSignature) GetAddress() string {
	if m != nil && mPtr.Address != nil {
		return *mPtr.Address
	}
	return ""
}

func (mPtr *MessageSignature) GetSignature() []byte {
	if m != nil {
		return mPtr.Signature
	}
	return nil
}

// *
// Request: Ask device to encrypt message
// @next EncryptedMessage
// @next Failure
type EncryptMessage struct {
	Pubkey           []byte   `protobuf:"bytes,1,opt,name=pubkey" json:"pubkey,omitempty"`
	Message          []byte   `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
	DisplayOnly      *bool    `protobuf:"varint,3,opt,name=display_only,json=displayOnly" json:"display_only,omitempty"`
	AddressN         []uint32 `protobuf:"varint,4,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	CoinName         *string  `protobuf:"bytes,5,opt,name=coin_name,json=coinName,def=Bitcoin" json:"coin_name,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (mPtr *EncryptMessage) Reset()                    { *m = EncryptMessage{} }
func (mPtr *EncryptMessage) String() string            { return proto.CompactTextString(m) }
func (*EncryptMessage) ProtoMessage()               {}
func (*EncryptMessage) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{37} }

const Default_EncryptMessage_CoinName string = "Bitcoin"

func (mPtr *EncryptMessage) GetPubkey() []byte {
	if m != nil {
		return mPtr.Pubkey
	}
	return nil
}

func (mPtr *EncryptMessage) GetMessage() []byte {
	if m != nil {
		return mPtr.Message
	}
	return nil
}

func (mPtr *EncryptMessage) GetDisplayOnly() bool {
	if m != nil && mPtr.DisplayOnly != nil {
		return *mPtr.DisplayOnly
	}
	return false
}

func (mPtr *EncryptMessage) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

func (mPtr *EncryptMessage) GetCoinName() string {
	if m != nil && mPtr.CoinName != nil {
		return *mPtr.CoinName
	}
	return Default_EncryptMessage_CoinName
}

// *
// Response: Encrypted message
// @prev EncryptMessage
type EncryptedMessage struct {
	Nonce            []byte `protobuf:"bytes,1,opt,name=nonce" json:"nonce,omitempty"`
	Message          []byte `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
	Hmac             []byte `protobuf:"bytes,3,opt,name=hmac" json:"hmac,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *EncryptedMessage) Reset()                    { *m = EncryptedMessage{} }
func (mPtr *EncryptedMessage) String() string            { return proto.CompactTextString(m) }
func (*EncryptedMessage) ProtoMessage()               {}
func (*EncryptedMessage) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{38} }

func (mPtr *EncryptedMessage) GetNonce() []byte {
	if m != nil {
		return mPtr.Nonce
	}
	return nil
}

func (mPtr *EncryptedMessage) GetMessage() []byte {
	if m != nil {
		return mPtr.Message
	}
	return nil
}

func (mPtr *EncryptedMessage) GetHmac() []byte {
	if m != nil {
		return mPtr.Hmac
	}
	return nil
}

// *
// Request: Ask device to decrypt message
// @next Success
// @next Failure
type DecryptMessage struct {
	AddressN         []uint32 `protobuf:"varint,1,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	Nonce            []byte   `protobuf:"bytes,2,opt,name=nonce" json:"nonce,omitempty"`
	Message          []byte   `protobuf:"bytes,3,opt,name=message" json:"message,omitempty"`
	Hmac             []byte   `protobuf:"bytes,4,opt,name=hmac" json:"hmac,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (mPtr *DecryptMessage) Reset()                    { *m = DecryptMessage{} }
func (mPtr *DecryptMessage) String() string            { return proto.CompactTextString(m) }
func (*DecryptMessage) ProtoMessage()               {}
func (*DecryptMessage) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{39} }

func (mPtr *DecryptMessage) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

func (mPtr *DecryptMessage) GetNonce() []byte {
	if m != nil {
		return mPtr.Nonce
	}
	return nil
}

func (mPtr *DecryptMessage) GetMessage() []byte {
	if m != nil {
		return mPtr.Message
	}
	return nil
}

func (mPtr *DecryptMessage) GetHmac() []byte {
	if m != nil {
		return mPtr.Hmac
	}
	return nil
}

// *
// Response: Decrypted message
// @prev DecryptedMessage
type DecryptedMessage struct {
	Message          []byte  `protobuf:"bytes,1,opt,name=message" json:"message,omitempty"`
	Address          *string `protobuf:"bytes,2,opt,name=address" json:"address,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *DecryptedMessage) Reset()                    { *m = DecryptedMessage{} }
func (mPtr *DecryptedMessage) String() string            { return proto.CompactTextString(m) }
func (*DecryptedMessage) ProtoMessage()               {}
func (*DecryptedMessage) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{40} }

func (mPtr *DecryptedMessage) GetMessage() []byte {
	if m != nil {
		return mPtr.Message
	}
	return nil
}

func (mPtr *DecryptedMessage) GetAddress() string {
	if m != nil && mPtr.Address != nil {
		return *mPtr.Address
	}
	return ""
}

// *
// Request: Ask device to encrypt or decrypt value of given key
// @next CipheredKeyValue
// @next Failure
type CipherKeyValue struct {
	AddressN         []uint32 `protobuf:"varint,1,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	Key              *string  `protobuf:"bytes,2,opt,name=key" json:"key,omitempty"`
	Value            []byte   `protobuf:"bytes,3,opt,name=value" json:"value,omitempty"`
	Encrypt          *bool    `protobuf:"varint,4,opt,name=encrypt" json:"encrypt,omitempty"`
	AskOnEncrypt     *bool    `protobuf:"varint,5,opt,name=ask_on_encrypt,json=askOnEncrypt" json:"ask_on_encrypt,omitempty"`
	AskOnDecrypt     *bool    `protobuf:"varint,6,opt,name=ask_on_decrypt,json=askOnDecrypt" json:"ask_on_decrypt,omitempty"`
	Iv               []byte   `protobuf:"bytes,7,opt,name=iv" json:"iv,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (mPtr *CipherKeyValue) Reset()                    { *m = CipherKeyValue{} }
func (mPtr *CipherKeyValue) String() string            { return proto.CompactTextString(m) }
func (*CipherKeyValue) ProtoMessage()               {}
func (*CipherKeyValue) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{41} }

func (mPtr *CipherKeyValue) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

func (mPtr *CipherKeyValue) GetKey() string {
	if m != nil && mPtr.Key != nil {
		return *mPtr.Key
	}
	return ""
}

func (mPtr *CipherKeyValue) GetValue() []byte {
	if m != nil {
		return mPtr.Value
	}
	return nil
}

func (mPtr *CipherKeyValue) GetEncrypt() bool {
	if m != nil && mPtr.Encrypt != nil {
		return *mPtr.Encrypt
	}
	return false
}

func (mPtr *CipherKeyValue) GetAskOnEncrypt() bool {
	if m != nil && mPtr.AskOnEncrypt != nil {
		return *mPtr.AskOnEncrypt
	}
	return false
}

func (mPtr *CipherKeyValue) GetAskOnDecrypt() bool {
	if m != nil && mPtr.AskOnDecrypt != nil {
		return *mPtr.AskOnDecrypt
	}
	return false
}

func (mPtr *CipherKeyValue) GetIv() []byte {
	if m != nil {
		return mPtr.Iv
	}
	return nil
}

// *
// Response: Return ciphered/deciphered value
// @prev CipherKeyValue
type CipheredKeyValue struct {
	Value            []byte `protobuf:"bytes,1,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *CipheredKeyValue) Reset()                    { *m = CipheredKeyValue{} }
func (mPtr *CipheredKeyValue) String() string            { return proto.CompactTextString(m) }
func (*CipheredKeyValue) ProtoMessage()               {}
func (*CipheredKeyValue) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{42} }

func (mPtr *CipheredKeyValue) GetValue() []byte {
	if m != nil {
		return mPtr.Value
	}
	return nil
}

// *
// Request: Estimated size of the transaction
// This behaves exactly like SignTx, which means that it can ask using TxRequest
// This call is non-blocking (except possible PassphraseRequest to unlock the seed)
// @next TxSize
// @next Failure
type EstimateTxSize struct {
	OutputsCount     *uint32 `protobuf:"varint,1,req,name=outputs_count,json=outputsCount" json:"outputs_count,omitempty"`
	InputsCount      *uint32 `protobuf:"varint,2,req,name=inputs_count,json=inputsCount" json:"inputs_count,omitempty"`
	CoinName         *string `protobuf:"bytes,3,opt,name=coin_name,json=coinName,def=Bitcoin" json:"coin_name,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *EstimateTxSize) Reset()                    { *m = EstimateTxSize{} }
func (mPtr *EstimateTxSize) String() string            { return proto.CompactTextString(m) }
func (*EstimateTxSize) ProtoMessage()               {}
func (*EstimateTxSize) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{43} }

const Default_EstimateTxSize_CoinName string = "Bitcoin"

func (mPtr *EstimateTxSize) GetOutputsCount() uint32 {
	if m != nil && mPtr.OutputsCount != nil {
		return *mPtr.OutputsCount
	}
	return 0
}

func (mPtr *EstimateTxSize) GetInputsCount() uint32 {
	if m != nil && mPtr.InputsCount != nil {
		return *mPtr.InputsCount
	}
	return 0
}

func (mPtr *EstimateTxSize) GetCoinName() string {
	if m != nil && mPtr.CoinName != nil {
		return *mPtr.CoinName
	}
	return Default_EstimateTxSize_CoinName
}

// *
// Response: Estimated size of the transaction
// @prev EstimateTxSize
type TxSize struct {
	TxSize           *uint32 `protobuf:"varint,1,opt,name=tx_size,json=txSize" json:"tx_size,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *TxSize) Reset()                    { *m = TxSize{} }
func (mPtr *TxSize) String() string            { return proto.CompactTextString(m) }
func (*TxSize) ProtoMessage()               {}
func (*TxSize) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{44} }

func (mPtr *TxSize) GetTxSize() uint32 {
	if m != nil && mPtr.TxSize != nil {
		return *mPtr.TxSize
	}
	return 0
}

// *
// Request: Ask device to sign transaction
// @next PassphraseRequest
// @next PinMatrixRequest
// @next TxRequest
// @next Failure
type SignTx struct {
	OutputsCount     *uint32 `protobuf:"varint,1,req,name=outputs_count,json=outputsCount" json:"outputs_count,omitempty"`
	InputsCount      *uint32 `protobuf:"varint,2,req,name=inputs_count,json=inputsCount" json:"inputs_count,omitempty"`
	CoinName         *string `protobuf:"bytes,3,opt,name=coin_name,json=coinName,def=Bitcoin" json:"coin_name,omitempty"`
	Version          *uint32 `protobuf:"varint,4,opt,name=version,def=1" json:"version,omitempty"`
	Locktime         *uint32 `protobuf:"varint,5,opt,name=lock_time,json=locktime,def=0" json:"lock_time,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *SignTx) Reset()                    { *m = SignTx{} }
func (mPtr *SignTx) String() string            { return proto.CompactTextString(m) }
func (*SignTx) ProtoMessage()               {}
func (*SignTx) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{45} }

const Default_SignTx_CoinName string = "Bitcoin"
const Default_SignTx_Version uint32 = 1
const Default_SignTx_Locktime uint32 = 0

func (mPtr *SignTx) GetOutputsCount() uint32 {
	if m != nil && mPtr.OutputsCount != nil {
		return *mPtr.OutputsCount
	}
	return 0
}

func (mPtr *SignTx) GetInputsCount() uint32 {
	if m != nil && mPtr.InputsCount != nil {
		return *mPtr.InputsCount
	}
	return 0
}

func (mPtr *SignTx) GetCoinName() string {
	if m != nil && mPtr.CoinName != nil {
		return *mPtr.CoinName
	}
	return Default_SignTx_CoinName
}

func (mPtr *SignTx) GetVersion() uint32 {
	if m != nil && mPtr.Version != nil {
		return *mPtr.Version
	}
	return Default_SignTx_Version
}

func (mPtr *SignTx) GetLocktime() uint32 {
	if m != nil && mPtr.Locktime != nil {
		return *mPtr.Locktime
	}
	return Default_SignTx_Locktime
}

// *
// Request: Simplified transaction signing
// This method doesn't support streaming, so there are hardware limits in number of inputs and outputs.
// In case of success, the result is returned using TxRequest message.
// @next PassphraseRequest
// @next PinMatrixRequest
// @next TxRequest
// @next Failure
type SimpleSignTx struct {
	Inputs           []*TxInputType     `protobuf:"bytes,1,rep,name=inputs" json:"inputs,omitempty"`
	Outputs          []*TxOutputType    `protobuf:"bytes,2,rep,name=outputs" json:"outputs,omitempty"`
	Transactions     []*TransactionType `protobuf:"bytes,3,rep,name=transactions" json:"transactions,omitempty"`
	CoinName         *string            `protobuf:"bytes,4,opt,name=coin_name,json=coinName,def=Bitcoin" json:"coin_name,omitempty"`
	Version          *uint32            `protobuf:"varint,5,opt,name=version,def=1" json:"version,omitempty"`
	Locktime         *uint32            `protobuf:"varint,6,opt,name=lock_time,json=locktime,def=0" json:"lock_time,omitempty"`
	XXX_unrecognized []byte             `json:"-"`
}

func (mPtr *SimpleSignTx) Reset()                    { *m = SimpleSignTx{} }
func (mPtr *SimpleSignTx) String() string            { return proto.CompactTextString(m) }
func (*SimpleSignTx) ProtoMessage()               {}
func (*SimpleSignTx) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{46} }

const Default_SimpleSignTx_CoinName string = "Bitcoin"
const Default_SimpleSignTx_Version uint32 = 1
const Default_SimpleSignTx_Locktime uint32 = 0

func (mPtr *SimpleSignTx) GetInputs() []*TxInputType {
	if m != nil {
		return mPtr.Inputs
	}
	return nil
}

func (mPtr *SimpleSignTx) GetOutputs() []*TxOutputType {
	if m != nil {
		return mPtr.Outputs
	}
	return nil
}

func (mPtr *SimpleSignTx) GetTransactions() []*TransactionType {
	if m != nil {
		return mPtr.Transactions
	}
	return nil
}

func (mPtr *SimpleSignTx) GetCoinName() string {
	if m != nil && mPtr.CoinName != nil {
		return *mPtr.CoinName
	}
	return Default_SimpleSignTx_CoinName
}

func (mPtr *SimpleSignTx) GetVersion() uint32 {
	if m != nil && mPtr.Version != nil {
		return *mPtr.Version
	}
	return Default_SimpleSignTx_Version
}

func (mPtr *SimpleSignTx) GetLocktime() uint32 {
	if m != nil && mPtr.Locktime != nil {
		return *mPtr.Locktime
	}
	return Default_SimpleSignTx_Locktime
}

// *
// Response: Device asks for information for signing transaction or returns the last result
// If request_index is set, device awaits TxAck message (with fields filled in according to request_type)
// If signature_index is set, 'signature' contains signed input of signature_index's input
// @prev SignTx
// @prev SimpleSignTx
// @prev TxAck
type TxRequest struct {
	RequestType      *RequestType             `protobuf:"varint,1,opt,name=request_type,json=requestType,enum=RequestType" json:"request_type,omitempty"`
	Details          *TxRequestDetailsType    `protobuf:"bytes,2,opt,name=details" json:"details,omitempty"`
	Serialized       *TxRequestSerializedType `protobuf:"bytes,3,opt,name=serialized" json:"serialized,omitempty"`
	XXX_unrecognized []byte                   `json:"-"`
}

func (mPtr *TxRequest) Reset()                    { *m = TxRequest{} }
func (mPtr *TxRequest) String() string            { return proto.CompactTextString(m) }
func (*TxRequest) ProtoMessage()               {}
func (*TxRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{47} }

func (mPtr *TxRequest) GetRequestType() RequestType {
	if m != nil && mPtr.RequestType != nil {
		return *mPtr.RequestType
	}
	return RequestType_TXINPUT
}

func (mPtr *TxRequest) GetDetails() *TxRequestDetailsType {
	if m != nil {
		return mPtr.Details
	}
	return nil
}

func (mPtr *TxRequest) GetSerialized() *TxRequestSerializedType {
	if m != nil {
		return mPtr.Serialized
	}
	return nil
}

// *
// Request: Reported transaction data
// @prev TxRequest
// @next TxRequest
type TxAck struct {
	Tx               *TransactionType `protobuf:"bytes,1,opt,name=tx" json:"tx,omitempty"`
	XXX_unrecognized []byte           `json:"-"`
}

func (mPtr *TxAck) Reset()                    { *m = TxAck{} }
func (mPtr *TxAck) String() string            { return proto.CompactTextString(m) }
func (*TxAck) ProtoMessage()               {}
func (*TxAck) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{48} }

func (mPtr *TxAck) GetTx() *TransactionType {
	if m != nil {
		return mPtr.Tx
	}
	return nil
}

// *
// Request: Ask device to sign transaction
// All fields are optional from the protocol's point of view. Each field defaults to value `0` if missing.
// Note: the first at most 1024 bytes of data MUST be transmitted as part of this message.
// @next PassphraseRequest
// @next PinMatrixRequest
// @next BgmchainTxRequest
// @next Failure
type BgmchainSignTx struct {
	AddressN         []uint32 `protobuf:"varint,1,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	Nonce            []byte   `protobuf:"bytes,2,opt,name=nonce" json:"nonce,omitempty"`
	GasPrice         []byte   `protobuf:"bytes,3,opt,name=gas_price,json=gasPrice" json:"gas_price,omitempty"`
	GasLimit         []byte   `protobuf:"bytes,4,opt,name=gas_limit,json=gasLimit" json:"gas_limit,omitempty"`
	To               []byte   `protobuf:"bytes,5,opt,name=to" json:"to,omitempty"`
	Value            []byte   `protobuf:"bytes,6,opt,name=value" json:"value,omitempty"`
	DataInitialChunk []byte   `protobuf:"bytes,7,opt,name=data_initial_chunk,json=dataInitialChunk" json:"data_initial_chunk,omitempty"`
	DataLength       *uint32  `protobuf:"varint,8,opt,name=data_length,json=dataLength" json:"data_length,omitempty"`
	ChainId          *uint32  `protobuf:"varint,9,opt,name=chain_id,json=chainId" json:"chain_id,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (mPtr *BgmchainSignTx) Reset()                    { *m = BgmchainSignTx{} }
func (mPtr *BgmchainSignTx) String() string            { return proto.CompactTextString(m) }
func (*BgmchainSignTx) ProtoMessage()               {}
func (*BgmchainSignTx) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{49} }

func (mPtr *BgmchainSignTx) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

func (mPtr *BgmchainSignTx) GetNonce() []byte {
	if m != nil {
		return mPtr.Nonce
	}
	return nil
}

func (mPtr *BgmchainSignTx) GetGasPrice() []byte {
	if m != nil {
		return mPtr.GasPrice
	}
	return nil
}

func (mPtr *BgmchainSignTx) GetGasLimit() []byte {
	if m != nil {
		return mPtr.GasLimit
	}
	return nil
}

func (mPtr *BgmchainSignTx) GetTo() []byte {
	if m != nil {
		return mPtr.To
	}
	return nil
}

func (mPtr *BgmchainSignTx) GetValue() []byte {
	if m != nil {
		return mPtr.Value
	}
	return nil
}

func (mPtr *BgmchainSignTx) GetDataInitialChunk() []byte {
	if m != nil {
		return mPtr.DataInitialChunk
	}
	return nil
}

func (mPtr *BgmchainSignTx) GetDataLength() uint32 {
	if m != nil && mPtr.DataLength != nil {
		return *mPtr.DataLength
	}
	return 0
}

func (mPtr *BgmchainSignTx) GetChainId() uint32 {
	if m != nil && mPtr.ChainId != nil {
		return *mPtr.ChainId
	}
	return 0
}

// *
// Response: Device asks for more data from transaction payload, or returns the signature.
// If data_length is set, device awaits that many more bytes of payload.
// Otherwise, the signature_* fields contain the computed transaction signature. All three fields will be present.
// @prev BgmchainSignTx
// @next BgmchainTxAck
type BgmchainTxRequest struct {
	DataLength       *uint32 `protobuf:"varint,1,opt,name=data_length,json=dataLength" json:"data_length,omitempty"`
	SignatureV       *uint32 `protobuf:"varint,2,opt,name=signature_v,json=signatureV" json:"signature_v,omitempty"`
	SignatureR       []byte  `protobuf:"bytes,3,opt,name=signature_r,json=signatureR" json:"signature_r,omitempty"`
	SignatureS       []byte  `protobuf:"bytes,4,opt,name=signature_s,json=signatureS" json:"signature_s,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *BgmchainTxRequest) Reset()                    { *m = BgmchainTxRequest{} }
func (mPtr *BgmchainTxRequest) String() string            { return proto.CompactTextString(m) }
func (*BgmchainTxRequest) ProtoMessage()               {}
func (*BgmchainTxRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{50} }

func (mPtr *BgmchainTxRequest) GetDataLength() uint32 {
	if m != nil && mPtr.DataLength != nil {
		return *mPtr.DataLength
	}
	return 0
}

func (mPtr *BgmchainTxRequest) GetSignatureV() uint32 {
	if m != nil && mPtr.SignatureV != nil {
		return *mPtr.SignatureV
	}
	return 0
}

func (mPtr *BgmchainTxRequest) GetSignatureR() []byte {
	if m != nil {
		return mPtr.SignatureR
	}
	return nil
}

func (mPtr *BgmchainTxRequest) GetSignatureS() []byte {
	if m != nil {
		return mPtr.SignatureS
	}
	return nil
}

// *
// Request: Transaction payload data.
// @prev BgmchainTxRequest
// @next BgmchainTxRequest
type BgmchainTxAck struct {
	DataChunk        []byte `protobuf:"bytes,1,opt,name=data_chunk,json=dataChunk" json:"data_chunk,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *BgmchainTxAck) Reset()                    { *m = BgmchainTxAck{} }
func (mPtr *BgmchainTxAck) String() string            { return proto.CompactTextString(m) }
func (*BgmchainTxAck) ProtoMessage()               {}
func (*BgmchainTxAck) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{51} }

func (mPtr *BgmchainTxAck) GetDataChunk() []byte {
	if m != nil {
		return mPtr.DataChunk
	}
	return nil
}

// *
// Request: Ask device to sign message
// @next BgmchainMessageSignature
// @next Failure
type BgmchainSignMessage struct {
	AddressN         []uint32 `protobuf:"varint,1,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	Message          []byte   `protobuf:"bytes,2,req,name=message" json:"message,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (mPtr *BgmchainSignMessage) Reset()                    { *m = BgmchainSignMessage{} }
func (mPtr *BgmchainSignMessage) String() string            { return proto.CompactTextString(m) }
func (*BgmchainSignMessage) ProtoMessage()               {}
func (*BgmchainSignMessage) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{52} }

func (mPtr *BgmchainSignMessage) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

func (mPtr *BgmchainSignMessage) GetMessage() []byte {
	if m != nil {
		return mPtr.Message
	}
	return nil
}

// *
// Request: Ask device to verify message
// @next Success
// @next Failure
type BgmchainVerifyMessage struct {
	Address          []byte `protobuf:"bytes,1,opt,name=address" json:"address,omitempty"`
	Signature        []byte `protobuf:"bytes,2,opt,name=signature" json:"signature,omitempty"`
	Message          []byte `protobuf:"bytes,3,opt,name=message" json:"message,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *BgmchainVerifyMessage) Reset()                    { *m = BgmchainVerifyMessage{} }
func (mPtr *BgmchainVerifyMessage) String() string            { return proto.CompactTextString(m) }
func (*BgmchainVerifyMessage) ProtoMessage()               {}
func (*BgmchainVerifyMessage) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{53} }

func (mPtr *BgmchainVerifyMessage) GetAddress() []byte {
	if m != nil {
		return mPtr.Address
	}
	return nil
}

func (mPtr *BgmchainVerifyMessage) GetSignature() []byte {
	if m != nil {
		return mPtr.Signature
	}
	return nil
}

func (mPtr *BgmchainVerifyMessage) GetMessage() []byte {
	if m != nil {
		return mPtr.Message
	}
	return nil
}

// *
// Response: Signed message
// @prev BgmchainSignMessage
type BgmchainMessageSignature struct {
	Address          []byte `protobuf:"bytes,1,opt,name=address" json:"address,omitempty"`
	Signature        []byte `protobuf:"bytes,2,opt,name=signature" json:"signature,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *BgmchainMessageSignature) Reset()                    { *m = BgmchainMessageSignature{} }
func (mPtr *BgmchainMessageSignature) String() string            { return proto.CompactTextString(m) }
func (*BgmchainMessageSignature) ProtoMessage()               {}
func (*BgmchainMessageSignature) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{54} }

func (mPtr *BgmchainMessageSignature) GetAddress() []byte {
	if m != nil {
		return mPtr.Address
	}
	return nil
}

func (mPtr *BgmchainMessageSignature) GetSignature() []byte {
	if m != nil {
		return mPtr.Signature
	}
	return nil
}

// *
// Request: Ask device to sign identity
// @next SignedIdentity
// @next Failure
type SignIdentity struct {
	Identity         *IdentityType `protobuf:"bytes,1,opt,name=identity" json:"identity,omitempty"`
	ChallengeHidden  []byte        `protobuf:"bytes,2,opt,name=challenge_hidden,json=challengeHidden" json:"challenge_hidden,omitempty"`
	ChallengeVisual  *string       `protobuf:"bytes,3,opt,name=challenge_visual,json=challengeVisual" json:"challenge_visual,omitempty"`
	EcdsaCurveName   *string       `protobuf:"bytes,4,opt,name=ecdsa_curve_name,json=ecdsaCurveName" json:"ecdsa_curve_name,omitempty"`
	XXX_unrecognized []byte        `json:"-"`
}

func (mPtr *SignIdentity) Reset()                    { *m = SignIdentity{} }
func (mPtr *SignIdentity) String() string            { return proto.CompactTextString(m) }
func (*SignIdentity) ProtoMessage()               {}
func (*SignIdentity) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{55} }

func (mPtr *SignIdentity) GetIdentity() *IdentityType {
	if m != nil {
		return mPtr.Identity
	}
	return nil
}

func (mPtr *SignIdentity) GetChallengeHidden() []byte {
	if m != nil {
		return mPtr.ChallengeHidden
	}
	return nil
}

func (mPtr *SignIdentity) GetChallengeVisual() string {
	if m != nil && mPtr.ChallengeVisual != nil {
		return *mPtr.ChallengeVisual
	}
	return ""
}

func (mPtr *SignIdentity) GetEcdsaCurveName() string {
	if m != nil && mPtr.EcdsaCurveName != nil {
		return *mPtr.EcdsaCurveName
	}
	return ""
}

// *
// Response: Device provides signed identity
// @prev SignIdentity
type SignedIdentity struct {
	Address          *string `protobuf:"bytes,1,opt,name=address" json:"address,omitempty"`
	PublicKey        []byte  `protobuf:"bytes,2,opt,name=public_key,json=publicKey" json:"public_key,omitempty"`
	Signature        []byte  `protobuf:"bytes,3,opt,name=signature" json:"signature,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *SignedIdentity) Reset()                    { *m = SignedIdentity{} }
func (mPtr *SignedIdentity) String() string            { return proto.CompactTextString(m) }
func (*SignedIdentity) ProtoMessage()               {}
func (*SignedIdentity) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{56} }

func (mPtr *SignedIdentity) GetAddress() string {
	if m != nil && mPtr.Address != nil {
		return *mPtr.Address
	}
	return ""
}

func (mPtr *SignedIdentity) GetPublicKey() []byte {
	if m != nil {
		return mPtr.PublicKey
	}
	return nil
}

func (mPtr *SignedIdentity) GetSignature() []byte {
	if m != nil {
		return mPtr.Signature
	}
	return nil
}

// *
// Request: Ask device to generate ECDH session key
// @next ECDHSessionKey
// @next Failure
type GetECDHSessionKey struct {
	Identity         *IdentityType `protobuf:"bytes,1,opt,name=identity" json:"identity,omitempty"`
	PeerPublicKey    []byte        `protobuf:"bytes,2,opt,name=peer_public_key,json=peerPublicKey" json:"peer_public_key,omitempty"`
	EcdsaCurveName   *string       `protobuf:"bytes,3,opt,name=ecdsa_curve_name,json=ecdsaCurveName" json:"ecdsa_curve_name,omitempty"`
	XXX_unrecognized []byte        `json:"-"`
}

func (mPtr *GetECDHSessionKey) Reset()                    { *m = GetECDHSessionKey{} }
func (mPtr *GetECDHSessionKey) String() string            { return proto.CompactTextString(m) }
func (*GetECDHSessionKey) ProtoMessage()               {}
func (*GetECDHSessionKey) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{57} }

func (mPtr *GetECDHSessionKey) GetIdentity() *IdentityType {
	if m != nil {
		return mPtr.Identity
	}
	return nil
}

func (mPtr *GetECDHSessionKey) GetPeerPublicKey() []byte {
	if m != nil {
		return mPtr.PeerPublicKey
	}
	return nil
}

func (mPtr *GetECDHSessionKey) GetEcdsaCurveName() string {
	if m != nil && mPtr.EcdsaCurveName != nil {
		return *mPtr.EcdsaCurveName
	}
	return ""
}

// *
// Response: Device provides ECDH session key
// @prev GetECDHSessionKey
type ECDHSessionKey struct {
	SessionKey       []byte `protobuf:"bytes,1,opt,name=session_key,json=sessionKey" json:"session_key,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *ECDHSessionKey) Reset()                    { *m = ECDHSessionKey{} }
func (mPtr *ECDHSessionKey) String() string            { return proto.CompactTextString(m) }
func (*ECDHSessionKey) ProtoMessage()               {}
func (*ECDHSessionKey) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{58} }

func (mPtr *ECDHSessionKey) GetSessionKey() []byte {
	if m != nil {
		return mPtr.SessionKey
	}
	return nil
}

// *
// Request: Set U2F counter
// @next Success
type SetU2FCounter struct {
	U2FCounter       *uint32 `protobuf:"varint,1,opt,name=u2f_counter,json=u2fCounter" json:"u2f_counter,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *SetU2FCounter) Reset()                    { *m = SetU2FCounter{} }
func (mPtr *SetU2FCounter) String() string            { return proto.CompactTextString(m) }
func (*SetU2FCounter) ProtoMessage()               {}
func (*SetU2FCounter) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{59} }

func (mPtr *SetU2FCounter) GetU2FCounter() uint32 {
	if m != nil && mPtr.U2FCounter != nil {
		return *mPtr.U2FCounter
	}
	return 0
}

// *
// Request: Ask device to erase its firmware (so it can be replaced via FirmwareUpload)
// @next Success
// @next FirmwareRequest
// @next Failure
type FirmwareErase struct {
	Length           *uint32 `protobuf:"varint,1,opt,name=length" json:"length,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *FirmwareErase) Reset()                    { *m = FirmwareErase{} }
func (mPtr *FirmwareErase) String() string            { return proto.CompactTextString(m) }
func (*FirmwareErase) ProtoMessage()               {}
func (*FirmwareErase) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{60} }

func (mPtr *FirmwareErase) GetLen() uint32 {
	if m != nil && mPtr.Length != nil {
		return *mPtr.Length
	}
	return 0
}

// *
// Response: Ask for firmware chunk
// @next FirmwareUpload
type FirmwareRequest struct {
	Offset           *uint32 `protobuf:"varint,1,opt,name=offset" json:"offset,omitempty"`
	Length           *uint32 `protobuf:"varint,2,opt,name=length" json:"length,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *FirmwareRequest) Reset()                    { *m = FirmwareRequest{} }
func (mPtr *FirmwareRequest) String() string            { return proto.CompactTextString(m) }
func (*FirmwareRequest) ProtoMessage()               {}
func (*FirmwareRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{61} }

func (mPtr *FirmwareRequest) GetOffset() uint32 {
	if m != nil && mPtr.Offset != nil {
		return *mPtr.Offset
	}
	return 0
}

func (mPtr *FirmwareRequest) GetLen() uint32 {
	if m != nil && mPtr.Length != nil {
		return *mPtr.Length
	}
	return 0
}

// *
// Request: Send firmware in binary form to the device
// @next Success
// @next Failure
type FirmwareUpload struct {
	Payload          []byte `protobuf:"bytes,1,req,name=payload" json:"payload,omitempty"`
	Hash             []byte `protobuf:"bytes,2,opt,name=hash" json:"hash,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *FirmwareUpload) Reset()                    { *m = FirmwareUpload{} }
func (mPtr *FirmwareUpload) String() string            { return proto.CompactTextString(m) }
func (*FirmwareUpload) ProtoMessage()               {}
func (*FirmwareUpload) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{62} }

func (mPtr *FirmwareUpload) GetPayload() []byte {
	if m != nil {
		return mPtr.Payload
	}
	return nil
}

func (mPtr *FirmwareUpload) GetHash() []byte {
	if m != nil {
		return mPtr.Hash
	}
	return nil
}

// *
// Request: Perform a device self-test
// @next Success
// @next Failure
type SelfTest struct {
	Payload          []byte `protobuf:"bytes,1,opt,name=payload" json:"payload,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *SelfTest) Reset()                    { *m = SelfTest{} }
func (mPtr *SelfTest) String() string            { return proto.CompactTextString(m) }
func (*SelfTest) ProtoMessage()               {}
func (*SelfTest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{63} }

func (mPtr *SelfTest) GetPayload() []byte {
	if m != nil {
		return mPtr.Payload
	}
	return nil
}

// *
// Request: "Press" the button on the device
// @next Success
type DebugLinkDecision struct {
	YesNo            *bool  `protobuf:"varint,1,req,name=yes_no,json=yesNo" json:"yes_no,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *DebugLinkDecision) Reset()                    { *m = DebugLinkDecision{} }
func (mPtr *DebugLinkDecision) String() string            { return proto.CompactTextString(m) }
func (*DebugLinkDecision) ProtoMessage()               {}
func (*DebugLinkDecision) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{64} }

func (mPtr *DebugLinkDecision) GetYesNo() bool {
	if m != nil && mPtr.YesNo != nil {
		return *mPtr.YesNo
	}
	return false
}

// *
// Request: Computer asks for device state
// @next DebugLinkState
type DebugLinkGetState struct {
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *DebugLinkGetState) Reset()                    { *m = DebugLinkGetState{} }
func (mPtr *DebugLinkGetState) String() string            { return proto.CompactTextString(m) }
func (*DebugLinkGetState) ProtoMessage()               {}
func (*DebugLinkGetState) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{65} }

// *
// Response: Device current state
// @prev DebugLinkGetState
type DebugLinkState struct {
	Layout               []byte      `protobuf:"bytes,1,opt,name=layout" json:"layout,omitempty"`
	Pin                  *string     `protobuf:"bytes,2,opt,name=pin" json:"pin,omitempty"`
	Matrix               *string     `protobuf:"bytes,3,opt,name=matrix" json:"matrix,omitempty"`
	Mnemonic             *string     `protobuf:"bytes,4,opt,name=mnemonic" json:"mnemonic,omitempty"`
	Node                 *HDNodeType `protobuf:"bytes,5,opt,name=node" json:"node,omitempty"`
	PassphraseProtection *bool       `protobuf:"varint,6,opt,name=passphrase_protection,json=passphraseProtection" json:"passphrase_protection,omitempty"`
	ResetWord            *string     `protobuf:"bytes,7,opt,name=reset_word,json=resetWord" json:"reset_word,omitempty"`
	ResetEntropy         []byte      `protobuf:"bytes,8,opt,name=reset_entropy,json=resetEntropy" json:"reset_entropy,omitempty"`
	RecoveryFakeWord     *string     `protobuf:"bytes,9,opt,name=recovery_fake_word,json=recoveryFakeWord" json:"recovery_fake_word,omitempty"`
	RecoveryWordPos      *uint32     `protobuf:"varint,10,opt,name=recovery_word_pos,json=recoveryWordPos" json:"recovery_word_pos,omitempty"`
	XXX_unrecognized     []byte      `json:"-"`
}

func (mPtr *DebugLinkState) Reset()                    { *m = DebugLinkState{} }
func (mPtr *DebugLinkState) String() string            { return proto.CompactTextString(m) }
func (*DebugLinkState) ProtoMessage()               {}
func (*DebugLinkState) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{66} }

func (mPtr *DebugLinkState) GetLayout() []byte {
	if m != nil {
		return mPtr.Layout
	}
	return nil
}

func (mPtr *DebugLinkState) GetPin() string {
	if m != nil && mPtr.Pin != nil {
		return *mPtr.Pin
	}
	return ""
}

func (mPtr *DebugLinkState) GetMatrix() string {
	if m != nil && mPtr.Matrix != nil {
		return *mPtr.Matrix
	}
	return ""
}

func (mPtr *DebugLinkState) GetMnemonic() string {
	if m != nil && mPtr.Mnemonic != nil {
		return *mPtr.Mnemonic
	}
	return ""
}

func (mPtr *DebugLinkState) GetNode() *HDNodeType {
	if m != nil {
		return mPtr.Node
	}
	return nil
}

func (mPtr *DebugLinkState) GetPassphraseProtection() bool {
	if m != nil && mPtr.PassphraseProtection != nil {
		return *mPtr.PassphraseProtection
	}
	return false
}

func (mPtr *DebugLinkState) GetResetWord() string {
	if m != nil && mPtr.ResetWord != nil {
		return *mPtr.ResetWord
	}
	return ""
}

func (mPtr *DebugLinkState) GetResetEntropy() []byte {
	if m != nil {
		return mPtr.ResetEntropy
	}
	return nil
}

func (mPtr *DebugLinkState) GetRecoveryFakeWord() string {
	if m != nil && mPtr.RecoveryFakeWord != nil {
		return *mPtr.RecoveryFakeWord
	}
	return ""
}

func (mPtr *DebugLinkState) GetRecoveryWordPos() uint32 {
	if m != nil && mPtr.RecoveryWordPos != nil {
		return *mPtr.RecoveryWordPos
	}
	return 0
}

// *
// Request: Ask device to restart
type DebugLinkStop struct {
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *DebugLinkStop) Reset()                    { *m = DebugLinkStop{} }
func (mPtr *DebugLinkStop) String() string            { return proto.CompactTextString(m) }
func (*DebugLinkStop) ProtoMessage()               {}
func (*DebugLinkStop) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{67} }

// *
// Response: Device wants host to bgmlogs event
type DebugLinkbgmlogs struct {
	Level            *uint32 `protobuf:"varint,1,opt,name=level" json:"level,omitempty"`
	Bucket           *string `protobuf:"bytes,2,opt,name=bucket" json:"bucket,omitempty"`
	Text             *string `protobuf:"bytes,3,opt,name=text" json:"text,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *DebugLinkbgmlogs) Reset()                    { *m = DebugLinkbgmlogs{} }
func (mPtr *DebugLinkbgmlogs) String() string            { return proto.CompactTextString(m) }
func (*DebugLinkbgmlogs) ProtoMessage()               {}
func (*DebugLinkbgmlogs) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{68} }

func (mPtr *DebugLinkbgmlogs) GetLevel() uint32 {
	if m != nil && mPtr.Level != nil {
		return *mPtr.Level
	}
	return 0
}

func (mPtr *DebugLinkbgmlogs) GetBucket() string {
	if m != nil && mPtr.Bucket != nil {
		return *mPtr.Bucket
	}
	return ""
}

func (mPtr *DebugLinkbgmlogs) GetText() string {
	if m != nil && mPtr.Text != nil {
		return *mPtr.Text
	}
	return ""
}

// *
// Request: Read memory from device
// @next DebugLinkMemory
type DebugLinkMemoryRead struct {
	Address          *uint32 `protobuf:"varint,1,opt,name=address" json:"address,omitempty"`
	Length           *uint32 `protobuf:"varint,2,opt,name=length" json:"length,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *DebugLinkMemoryRead) Reset()                    { *m = DebugLinkMemoryRead{} }
func (mPtr *DebugLinkMemoryRead) String() string            { return proto.CompactTextString(m) }
func (*DebugLinkMemoryRead) ProtoMessage()               {}
func (*DebugLinkMemoryRead) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{69} }

func (mPtr *DebugLinkMemoryRead) GetAddress() uint32 {
	if m != nil && mPtr.Address != nil {
		return *mPtr.Address
	}
	return 0
}

func (mPtr *DebugLinkMemoryRead) GetLen() uint32 {
	if m != nil && mPtr.Length != nil {
		return *mPtr.Length
	}
	return 0
}

// *
// Response: Device sends memory back
// @prev DebugLinkMemoryRead
type DebugLinkMemory struct {
	Memory           []byte `protobuf:"bytes,1,opt,name=memory" json:"memory,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *DebugLinkMemory) Reset()                    { *m = DebugLinkMemory{} }
func (mPtr *DebugLinkMemory) String() string            { return proto.CompactTextString(m) }
func (*DebugLinkMemory) ProtoMessage()               {}
func (*DebugLinkMemory) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{70} }

func (mPtr *DebugLinkMemory) GetMemory() []byte {
	if m != nil {
		return mPtr.Memory
	}
	return nil
}

// *
// Request: Write memory to device.
// WARNING: Writing to the wrong location can irreparably break the device.
type DebugLinkMemoryWrite struct {
	Address          *uint32 `protobuf:"varint,1,opt,name=address" json:"address,omitempty"`
	Memory           []byte  `protobuf:"bytes,2,opt,name=memory" json:"memory,omitempty"`
	Flash            *bool   `protobuf:"varint,3,opt,name=flash" json:"flash,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *DebugLinkMemoryWrite) Reset()                    { *m = DebugLinkMemoryWrite{} }
func (mPtr *DebugLinkMemoryWrite) String() string            { return proto.CompactTextString(m) }
func (*DebugLinkMemoryWrite) ProtoMessage()               {}
func (*DebugLinkMemoryWrite) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{71} }

func (mPtr *DebugLinkMemoryWrite) GetAddress() uint32 {
	if m != nil && mPtr.Address != nil {
		return *mPtr.Address
	}
	return 0
}

func (mPtr *DebugLinkMemoryWrite) GetMemory() []byte {
	if m != nil {
		return mPtr.Memory
	}
	return nil
}

func (mPtr *DebugLinkMemoryWrite) GetFlash() bool {
	if m != nil && mPtr.Flash != nil {
		return *mPtr.Flash
	}
	return false
}

// *
// Request: Erase block of flash on device
// WARNING: Writing to the wrong location can irreparably break the device.
type DebugLinkFlashErase struct {
	Sector           *uint32 `protobuf:"varint,1,opt,name=sector" json:"sector,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *DebugLinkFlashErase) Reset()                    { *m = DebugLinkFlashErase{} }
func (mPtr *DebugLinkFlashErase) String() string            { return proto.CompactTextString(m) }
func (*DebugLinkFlashErase) ProtoMessage()               {}
func (*DebugLinkFlashErase) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{72} }

func (mPtr *DebugLinkFlashErase) GetSector() uint32 {
	if m != nil && mPtr.Sector != nil {
		return *mPtr.Sector
	}
	return 0
}

func init() {
	proto.RegisterType((*Initialize)(nil), "Initialize")
	proto.RegisterType((*GetFeatures)(nil), "GetFeatures")
	proto.RegisterType((*Features)(nil), "Features")
	proto.RegisterType((*ClearSession)(nil), "ClearSession")
	proto.RegisterType((*ApplySettings)(nil), "ApplySettings")
	proto.RegisterType((*ApplyFlags)(nil), "ApplyFlags")
	proto.RegisterType((*ChangePin)(nil), "ChangePin")
	proto.RegisterType((*Ping)(nil), "Ping")
	proto.RegisterType((*Success)(nil), "Success")
	proto.RegisterType((*Failure)(nil), "Failure")
	proto.RegisterType((*ButtonRequest)(nil), "ButtonRequest")
	proto.RegisterType((*ButtonAck)(nil), "ButtonAck")
	proto.RegisterType((*PinMatrixRequest)(nil), "PinMatrixRequest")
	proto.RegisterType((*PinMatrixAck)(nil), "PinMatrixAck")
	proto.RegisterType((*Cancel)(nil), "Cancel")
	proto.RegisterType((*PassphraseRequest)(nil), "PassphraseRequest")
	proto.RegisterType((*PassphraseAck)(nil), "PassphraseAck")
	proto.RegisterType((*GetEntropy)(nil), "GetEntropy")
	proto.RegisterType((*Entropy)(nil), "Entropy")
	proto.RegisterType((*GetPublicKey)(nil), "GetPublicKey")
	proto.RegisterType((*PublicKey)(nil), "PublicKey")
	proto.RegisterType((*GetAddress)(nil), "GetAddress")
	proto.RegisterType((*BgmchainGetAddress)(nil), "BgmchainGetAddress")
	proto.RegisterType((*Address)(nil), "Address")
	proto.RegisterType((*BgmchainAddress)(nil), "BgmchainAddress")
	proto.RegisterType((*WipeDevice)(nil), "WipeDevice")
	proto.RegisterType((*LoadDevice)(nil), "LoadDevice")
	proto.RegisterType((*ResetDevice)(nil), "ResetDevice")
	proto.RegisterType((*BackupDevice)(nil), "BackupDevice")
	proto.RegisterType((*EntropyRequest)(nil), "EntropyRequest")
	proto.RegisterType((*EntropyAck)(nil), "EntropyAck")
	proto.RegisterType((*RecoveryDevice)(nil), "RecoveryDevice")
	proto.RegisterType((*WordRequest)(nil), "WordRequest")
	proto.RegisterType((*WordAck)(nil), "WordAck")
	proto.RegisterType((*SignMessage)(nil), "SignMessage")
	proto.RegisterType((*VerifyMessage)(nil), "VerifyMessage")
	proto.RegisterType((*MessageSignature)(nil), "MessageSignature")
	proto.RegisterType((*EncryptMessage)(nil), "EncryptMessage")
	proto.RegisterType((*EncryptedMessage)(nil), "EncryptedMessage")
	proto.RegisterType((*DecryptMessage)(nil), "DecryptMessage")
	proto.RegisterType((*DecryptedMessage)(nil), "DecryptedMessage")
	proto.RegisterType((*CipherKeyValue)(nil), "CipherKeyValue")
	proto.RegisterType((*CipheredKeyValue)(nil), "CipheredKeyValue")
	proto.RegisterType((*EstimateTxSize)(nil), "EstimateTxSize")
	proto.RegisterType((*TxSize)(nil), "TxSize")
	proto.RegisterType((*SignTx)(nil), "SignTx")
	proto.RegisterType((*SimpleSignTx)(nil), "SimpleSignTx")
	proto.RegisterType((*TxRequest)(nil), "TxRequest")
	proto.RegisterType((*TxAck)(nil), "TxAck")
	proto.RegisterType((*BgmchainSignTx)(nil), "BgmchainSignTx")
	proto.RegisterType((*BgmchainTxRequest)(nil), "BgmchainTxRequest")
	proto.RegisterType((*BgmchainTxAck)(nil), "BgmchainTxAck")
	proto.RegisterType((*BgmchainSignMessage)(nil), "BgmchainSignMessage")
	proto.RegisterType((*BgmchainVerifyMessage)(nil), "BgmchainVerifyMessage")
	proto.RegisterType((*BgmchainMessageSignature)(nil), "BgmchainMessageSignature")
	proto.RegisterType((*SignIdentity)(nil), "SignIdentity")
	proto.RegisterType((*SignedIdentity)(nil), "SignedIdentity")
	proto.RegisterType((*GetECDHSessionKey)(nil), "GetECDHSessionKey")
	proto.RegisterType((*ECDHSessionKey)(nil), "ECDHSessionKey")
	proto.RegisterType((*SetU2FCounter)(nil), "SetU2FCounter")
	proto.RegisterType((*FirmwareErase)(nil), "FirmwareErase")
	proto.RegisterType((*FirmwareRequest)(nil), "FirmwareRequest")
	proto.RegisterType((*FirmwareUpload)(nil), "FirmwareUpload")
	proto.RegisterType((*SelfTest)(nil), "SelfTest")
	proto.RegisterType((*DebugLinkDecision)(nil), "DebugLinkDecision")
	proto.RegisterType((*DebugLinkGetState)(nil), "DebugLinkGetState")
	proto.RegisterType((*DebugLinkState)(nil), "DebugLinkState")
	proto.RegisterType((*DebugLinkStop)(nil), "DebugLinkStop")
	proto.RegisterType((*DebugLinkbgmlogs)(nil), "DebugLinkbgmlogs")
	proto.RegisterType((*DebugLinkMemoryRead)(nil), "DebugLinkMemoryRead")
	proto.RegisterType((*DebugLinkMemory)(nil), "DebugLinkMemory")
	proto.RegisterType((*DebugLinkMemoryWrite)(nil), "DebugLinkMemoryWrite")
	proto.RegisterType((*DebugLinkFlashErase)(nil), "DebugLinkFlashErase")
	proto.RegisterEnum("MessagesType", MessagesType_name, MessagesType_value)
}

func init() { proto.RegisterFile("messages.proto", fileDescriptor1) }

var fileDescriptor1 = []byte{
	// 3424 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xbc, 0x5a, 0xcb, 0x6f, 0xdc, 0x46,
	0x9a, 0x5f, 0x76, 0xb7, 0xfa, 0xf1, 0x35, 0xbb, 0x55, 0xa2, 0x2d, 0xbb, 0x2d, 0x5b, 0xb6, 0x4c,
	0xc9, 0xb6, 0x64, 0x27, 0xed, 0x44, 0x79, 0x6c, 0xd6, 0xbb, 0x79, 0xc8, 0x7a, 0xd8, 0xde, 0xd8,
	0x8e, 0xc0, 0x56, 0x9c, 0xdb, 0x12, 0x14, 0x59, 0xea, 0xae, 0x55, 0x37, 0xc9, 0xf0, 0xa1, 0xa8,
	0x7d, 0xd8, 0xeb, 0xee, 0x65, 0x81, 0xec, 0x69, 0x73, 0x1a, 0xe4, 0x36, 0x19, 0x04, 0x18, 0x0c,
	0x30, 0x18, 0x60, 0x72, 0x9a, 0x3f, 0x60, 0xfe, 0x8b, 0x39, 0xce, 0x1f, 0x30, 0xe7, 0x41, 0x3d,
	0x48, 0x16, 0x29, 0xb6, 0x6c, 0x27, 0xc0, 0x5c, 0x04, 0xd6, 0x57, 0xbf, 0xfe, 0xea, 0x7b, 0xd5,
	0x57, 0x5f, 0x7d, 0x25, 0xe8, 0x4e, 0x70, 0x18, 0x5a, 0x43, 0x1c, 0xf6, 0xfd, 0xc0, 0x8b, 0xbc,
	0xa5, 0x76, 0x34, 0xf5, 0x93, 0x81, 0xae, 0x02, 0x3c, 0x71, 0x49, 0x44, 0xac, 0x31, 0x79, 0x89,
	0xf5, 0x0e, 0xb4, 0x1f, 0xe1, 0x68, 0x0f, 0x5b, 0x51, 0x1c, 0xe0, 0x50, 0xff, 0x69, 0x0e, 0x9a,
	0xc9, 0x40, 0xbb, 0x04, 0xf5, 0x13, 0xec, 0x3a, 0x5e, 0xd0, 0x53, 0x56, 0x94, 0xf5, 0x96, 0x21,
	0x46, 0xda, 0x2a, 0x74, 0x26, 0xd6, 0x7f, 0x7a, 0x81, 0x79, 0x82, 0x83, 0x90, 0x78, 0x6e, 0xaf,
	0xb2, 0xa2, 0xac, 0x77, 0x0c, 0x95, 0x11, 0x5f, 0x70, 0x1a, 0x03, 0x11, 0x57, 0x02, 0x55, 0x05,
	0x88, 0x12, 0x25, 0x90, 0x6f, 0x45, 0xf6, 0x28, 0x05, 0xd5, 0x38, 0x88, 0x11, 0x13, 0xd0, 0x1d,
	0x98, 0x3f, 0xf4, 0xbc, 0x68, 0xec, 0x59, 0x0e, 0x0e, 0xcc, 0x89, 0xe7, 0xe0, 0xde, 0xdc, 0x8a,
	0xb2, 0xde, 0x34, 0xba, 0x19, 0xf9, 0x99, 0xe7, 0x60, 0xed, 0x2a, 0xb4, 0x1c, 0x7c, 0x42, 0x6c,
	0x6c, 0x12, 0xa7, 0x57, 0x67, 0x22, 0x37, 0x39, 0xe1, 0x89, 0xa3, 0xdd, 0x82, 0xae, 0x4f, 0x5c,
	0x93, 0xda, 0x00, 0xdb, 0x11, 0x5d, 0xab, 0xc1, 0x98, 0x74, 0x7c, 0xe2, 0xee, 0xa7, 0x44, 0xed,
	0x3d, 0x58, 0xf4, 0xad, 0x30, 0xf4, 0x47, 0x81, 0x15, 0x62, 0x19, 0xdd, 0x64, 0xe8, 0x8b, 0xd9,
	0xa4, 0xf4, 0xa3, 0x25, 0x68, 0x8e, 0x2d, 0x77, 0x18, 0x5b, 0x43, 0xdc, 0x6b, 0xf1, 0x75, 0x93,
	0xb1, 0x76, 0x11, 0xe6, 0xc6, 0xd6, 0x21, 0x1e, 0xf7, 0x80, 0x4d, 0xf0, 0x81, 0x76, 0x03, 0xe6,
	0x6c, 0x8f, 0xb8, 0x61, 0xaf, 0xbd, 0x52, 0x5d, 0x6f, 0x6f, 0xb6, 0xfa, 0xdb, 0x1e, 0x71, 0x0f,
	0xa6, 0x3e, 0x36, 0x38, 0x5d, 0x5b, 0x81, 0x36, 0x49, 0xbd, 0xe4, 0xf4, 0x54, 0xb6, 0xba, 0x4c,
	0xa2, 0x8b, 0x06, 0xf8, 0x84, 0x30, 0xb3, 0x75, 0x56, 0x94, 0x75, 0xd5, 0x48, 0xc7, 0x05, 0x93,
	0x8d, 0xac, 0x70, 0xd4, 0xeb, 0x32, 0x88, 0x64, 0xb2, 0xc7, 0x56, 0x38, 0xa2, 0x4c, 0xc8, 0xc4,
	0xf7, 0x82, 0x08, 0x3b, 0xbd, 0x79, 0xb6, 0x46, 0x3a, 0xd6, 0x96, 0x01, 0xa8, 0xc5, 0x6c, 0xcb,
	0x1e, 0x61, 0xa7, 0x87, 0xd8, 0x6c, 0xcb, 0x27, 0xee, 0x36, 0x23, 0x68, 0xf7, 0x60, 0x41, 0xb2,
	0x94, 0x40, 0x2d, 0x30, 0x14, 0xca, 0x26, 0x04, 0x78, 0x03, 0xd0, 0x11, 0x09, 0x26, 0xdf, 0x58,
	0x01, 0x35, 0x2a, 0x0e, 0xb1, 0x1b, 0xf5, 0x34, 0x86, 0x9d, 0x4f, 0xe8, 0xfb, 0x9c, 0xac, 0xdd,
	0x04, 0xd5, 0xc5, 0xd8, 0x09, 0xcd, 0x43, 0xcb, 0x3e, 0x8e, 0xfd, 0xde, 0x05, 0xae, 0x3a, 0xa3,
	0x3d, 0x64, 0x24, 0x6a, 0xd3, 0xa3, 0xb1, 0x35, 0x0c, 0x7b, 0x17, 0x59, 0xb8, 0xf0, 0x81, 0xde,
	0x05, 0x75, 0x7b, 0x8c, 0xad, 0x60, 0x80, 0x43, 0x6a, 0x04, 0xfd, 0x7f, 0x14, 0xe8, 0x6c, 0xf9,
	0xfe, 0x78, 0x3a, 0xc0, 0x51, 0x44, 0xdc, 0x61, 0x98, 0xf3, 0x93, 0x32, 0xcb, 0x4f, 0x15, 0xd9,
	0x4f, 0xb7, 0xa0, 0x1b, 0xd3, 0x38, 0x48, 0xf5, 0x61, 0x61, 0xdc, 0x34, 0x3a, 0x71, 0x88, 0xf7,
	0x53, 0xa2, 0x76, 0x1d, 0x60, 0xe4, 0x4d, 0x70, 0x68, 0x07, 0x18, 0xf3, 0x20, 0x56, 0x0d, 0x89,
	0xa2, 0xeb, 0x00, 0x4c, 0x92, 0x3d, 0x2a, 0x68, 0x26, 0xbe, 0x22, 0x8b, 0xbf, 0x0a, 0xad, 0xed,
	0x91, 0xe5, 0x0e, 0xf1, 0x3e, 0x71, 0xe9, 0xd6, 0x0b, 0xf0, 0xc4, 0x3b, 0xe1, 0x72, 0x36, 0x0d,
	0x31, 0xd2, 0x7f, 0xa3, 0x40, 0x6d, 0x9f, 0xb8, 0x43, 0xad, 0x07, 0x0d, 0xb1, 0xc9, 0x85, 0x26,
	0xc9, 0x90, 0xfa, 0xe5, 0x30, 0x8e, 0x22, 0x2f, 0x17, 0xeb, 0x15, 0xee, 0x17, 0x3e, 0x21, 0x45,
	0xee, 0xd9, 0x5d, 0x51, 0x7d, 0xa3, 0x5d, 0x51, 0x9b, 0xbd, 0x2b, 0xf4, 0x55, 0x68, 0x0c, 0x62,
	0xdb, 0xc6, 0x61, 0x38, 0x5b, 0x5a, 0x7d, 0x17, 0x1a, 0x7b, 0x16, 0x19, 0xc7, 0x01, 0xd6, 0x56,
	0xa0, 0x66, 0xd3, 0xcd, 0x4d, 0x11, 0xdd, 0x4d, 0xb5, 0x2f, 0xe8, 0x6c, 0x57, 0xb0, 0x19, 0x99,
	0x4d, 0x25, 0xcf, 0xe6, 0x73, 0xe8, 0x3c, 0x64, 0xba, 0x19, 0xf8, 0xeb, 0x18, 0x87, 0x91, 0x76,
	0x3b, 0xc7, 0x4c, 0xeb, 0xe7, 0x66, 0x25, 0x96, 0x1a, 0xd4, 0x1c, 0x2b, 0xb2, 0x04, 0x3f, 0xf6,
	0xad, 0xb7, 0xa1, 0xc5, 0xe1, 0x5b, 0xf6, 0xb1, 0xfe, 0x31, 0xa0, 0x7d, 0xe2, 0x3e, 0xb3, 0xa2,
	0x80, 0x9c, 0x26, 0xcc, 0x37, 0xa0, 0x46, 0x33, 0xaa, 0x60, 0xbe, 0xd8, 0x2f, 0x02, 0x38, 0x7f,
	0x0a, 0xd1, 0x57, 0x40, 0x4d, 0x67, 0xb7, 0xec, 0x63, 0x0d, 0x41, 0xd5, 0x27, 0x6e, 0x4f, 0x59,
	0xa9, 0xac, 0xb7, 0x0c, 0xfa, 0xa9, 0x37, 0xa1, 0xbe, 0x6d, 0xb9, 0x36, 0x1e, 0xeb, 0x17, 0x60,
	0x21, 0x8b, 0x29, 0xc1, 0x4a, 0xbf, 0x0f, 0x9d, 0x8c, 0x48, 0x39, 0x5c, 0x07, 0x90, 0xc2, 0x91,
	0x33, 0x92, 0x28, 0xfa, 0x0a, 0xc0, 0x23, 0x1c, 0xed, 0xba, 0x51, 0xe0, 0xf9, 0x53, 0xaa, 0x5f,
	0x48, 0x5e, 0x72, 0x5c, 0xc7, 0x60, 0xdf, 0xd4, 0x31, 0xc9, 0x74, 0x0f, 0x1a, 0x98, 0x7f, 0x32,
	0x84, 0x6a, 0x24, 0x43, 0xfd, 0x57, 0x0a, 0xa8, 0x8f, 0x70, 0xb4, 0x1f, 0x1f, 0x8e, 0x89, 0xfd,
	0x39, 0x9e, 0xd2, 0xec, 0x6a, 0x39, 0x4e, 0x80, 0xc3, 0xd0, 0xa4, 0xf2, 0x57, 0xd7, 0x3b, 0x46,
	0x53, 0x10, 0x9e, 0x6b, 0xeb, 0x80, 0xb0, 0xed, 0x84, 0x96, 0x69, 0xc7, 0xc1, 0x09, 0x36, 0x5d,
	0x6b, 0x92, 0xb8, 0xa8, 0xcb, 0xe8, 0xdb, 0x94, 0xfc, 0xdc, 0x9a, 0x60, 0xba, 0xbd, 0xc3, 0x91,
	0xf7, 0x8d, 0xe9, 0x90, 0xd0, 0x1f, 0x5b, 0x53, 0x11, 0x6f, 0x6d, 0x4a, 0xdb, 0xe1, 0x24, 0x6d,
	0x0d, 0x5a, 0x34, 0x09, 0x72, 0x2e, 0x34, 0xc2, 0x5a, 0x0f, 0x1a, 0x0f, 0x49, 0x44, 0x69, 0x46,
	0x93, 0xfe, 0xa5, 0x8c, 0xf4, 0xcf, 0xa0, 0x95, 0x09, 0x77, 0x03, 0x6a, 0x2e, 0x77, 0x77, 0x65,
	0xbd, 0xbd, 0xd9, 0xee, 0x3f, 0xde, 0x79, 0xee, 0x39, 0x22, 0x74, 0x5c, 0xe1, 0xe7, 0x53, 0x3f,
	0x3e, 0x4c, 0xfc, 0x4c, 0xbf, 0xf5, 0xbf, 0x2a, 0xcc, 0x54, 0x5b, 0x5c, 0x89, 0xf3, 0x15, 0xcc,
	0xc9, 0x54, 0x99, 0x21, 0xd3, 0xeb, 0x28, 0xf7, 0x01, 0x34, 0x27, 0xf1, 0x38, 0x22, 0x21, 0x19,
	0x32, 0xdd, 0xda, 0x9b, 0x57, 0xfa, 0xcf, 0x04, 0xc1, 0xc0, 0x0e, 0xc6, 0x93, 0x81, 0x1d, 0x10,
	0x9f, 0xc7, 0x50, 0x0a, 0xd5, 0x3e, 0x85, 0x76, 0xc8, 0xe8, 0x26, 0x8b, 0xbc, 0x39, 0x16, 0x79,
	0xa8, 0xff, 0xc4, 0xf5, 0xe3, 0x28, 0xfb, 0xc1, 0x03, 0x75, 0xb0, 0xbf, 0xfb, 0x7c, 0x67, 0x6b,
	0x67, 0xc7, 0xd8, 0x1d, 0x0c, 0x0c, 0x08, 0xd3, 0x19, 0xfd, 0x00, 0xb4, 0xdd, 0x68, 0x84, 0x03,
	0x1c, 0x4f, 0x5e, 0x57, 0xe7, 0xa2, 0x36, 0x95, 0x33, 0xda, 0xd0, 0x50, 0x4a, 0x58, 0xf5, 0xa0,
	0x21, 0x7e, 0x29, 0x82, 0x32, 0x19, 0xea, 0xf7, 0x60, 0x3e, 0x59, 0x7a, 0x06, 0x58, 0xcd, 0xc0,
	0x2a, 0xc0, 0x57, 0xc4, 0xc7, 0x3b, 0xec, 0xdc, 0xd6, 0xff, 0xaf, 0x02, 0xf0, 0xd4, 0xb3, 0x1c,
	0x3e, 0xa4, 0x09, 0x7c, 0xe2, 0xe2, 0x89, 0xe7, 0x12, 0x3b, 0x49, 0xe0, 0xc9, 0x38, 0x0d, 0x81,
	0x0a, 0x33, 0x6a, 0x49, 0x08, 0x88, 0xad, 0x57, 0x65, 0xbf, 0xa3, 0x9f, 0x3f, 0x2b, 0xad, 0x69,
	0xab, 0xd2, 0x21, 0x32, 0xc7, 0x03, 0x01, 0xbb, 0xc3, 0x31, 0x09, 0x47, 0x65, 0xa7, 0x49, 0x5d,
	0x3e, 0x4d, 0x56, 0xa1, 0x13, 0x1e, 0x13, 0xdf, 0xb4, 0x47, 0xd8, 0x3e, 0x0e, 0xe3, 0x89, 0x28,
	0x41, 0x54, 0x4a, 0xdc, 0x16, 0x34, 0xed, 0x06, 0xb4, 0xe3, 0xcd, 0x23, 0xd3, 0xf6, 0x62, 0x37,
	0xc2, 0x01, 0xab, 0x3b, 0x3a, 0x06, 0xc4, 0x9b, 0x47, 0xdb, 0x9c, 0xa2, 0xff, 0xb6, 0x02, 0x6d,
	0x03, 0x87, 0x38, 0x12, 0x46, 0xb9, 0x05, 0x5d, 0xe1, 0x21, 0x33, 0xb0, 0x5c, 0xc7, 0x9b, 0x88,
	0x33, 0xa3, 0x23, 0xa8, 0x06, 0x23, 0x6a, 0x37, 0xa0, 0x19, 0x46, 0x01, 0x76, 0x87, 0xd1, 0x88,
	0x17, 0x6c, 0x0f, 0xaa, 0x9b, 0x1f, 0x7c, 0x68, 0xa4, 0xc4, 0xd9, 0xd6, 0xa8, 0x9e, 0x63, 0x8d,
	0xb3, 0x07, 0x48, 0xad, 0xec, 0x00, 0xf9, 0x05, 0x46, 0x2b, 0xd8, 0xa3, 0x51, 0xb4, 0x07, 0x05,
	0x30, 0xab, 0x8a, 0x7a, 0x81, 0x17, 0x6a, 0x40, 0x49, 0xbc, 0x5c, 0xa0, 0x85, 0x01, 0xff, 0x12,
	0x41, 0x85, 0xa0, 0x2b, 0xf2, 0x5f, 0x92, 0x64, 0x6f, 0x03, 0x08, 0x0a, 0xcd, 0xb0, 0xb9, 0xa4,
	0xa8, 0xc8, 0x49, 0xf1, 0x4f, 0x15, 0xe8, 0x1a, 0xd8, 0xf6, 0x4e, 0x70, 0x30, 0x15, 0xd6, 0x5f,
	0x06, 0xf8, 0xc6, 0x0b, 0x1c, 0x2e, 0x9f, 0x38, 0xd1, 0x5b, 0x94, 0xc2, 0xc4, 0x9b, 0x6d, 0xd4,
	0xca, 0x1b, 0x19, 0xb5, 0xfa, 0x2a, 0xa3, 0xd6, 0x5e, 0x69, 0xd4, 0x39, 0xd9, 0xa8, 0x1b, 0x80,
	0xb0, 0x7b, 0xe4, 0x05, 0x36, 0x36, 0xa9, 0xac, 0x63, 0x12, 0x46, 0xcc, 0xea, 0x4d, 0x63, 0x5e,
	0xd0, 0xbf, 0x12, 0x64, 0x9a, 0x39, 0x59, 0xca, 0xe1, 0x81, 0xc8, 0xbe, 0x8b, 0x3e, 0x69, 0x9d,
	0xf1, 0xc9, 0x65, 0x68, 0x38, 0xc1, 0xd4, 0x0c, 0x62, 0x97, 0xd5, 0xbd, 0x4d, 0xa3, 0xee, 0x04,
	0x53, 0x23, 0x76, 0xf5, 0xf7, 0xa0, 0x4d, 0x39, 0x27, 0x27, 0xe9, 0x5a, 0xee, 0x24, 0x45, 0x7d,
	0x69, 0x4e, 0x3a, 0x44, 0x97, 0xa1, 0x41, 0x27, 0xa8, 0x6f, 0x34, 0xa8, 0x51, 0x81, 0x45, 0x8a,
	0x61, 0xdf, 0xfa, 0x8f, 0x0a, 0xb4, 0x07, 0x64, 0xe8, 0x3e, 0x13, 0x15, 0xd0, 0xb9, 0x49, 0x2d,
	0x57, 0x43, 0xb0, 0xcc, 0x93, 0x14, 0x4e, 0xb9, 0x14, 0x5f, 0x9d, 0x95, 0xe2, 0x0b, 0x89, 0xb8,
	0xf6, 0xc6, 0x89, 0xf8, 0xbf, 0x15, 0xe8, 0xbc, 0xc0, 0x01, 0x39, 0x9a, 0x26, 0xf2, 0xe6, 0x92,
	0xa1, 0x22, 0x65, 0x4e, 0xed, 0x1a, 0xb4, 0x42, 0x32, 0x74, 0xd9, 0x7d, 0x8c, 0x45, 0x8c, 0x6a,
	0x64, 0x04, 0x59, 0x95, 0x2a, 0x8f, 0xd3, 0x52, 0x55, 0x66, 0x9e, 0xa0, 0xff, 0x0e, 0x48, 0x88,
	0x30, 0x90, 0x79, 0xfe, 0x1c, 0x59, 0xf4, 0x1f, 0x14, 0xba, 0xa9, 0xec, 0x60, 0xea, 0x47, 0x89,
	0x5a, 0x97, 0xa0, 0xee, 0xc7, 0x87, 0xc7, 0x38, 0xd9, 0x45, 0x62, 0x54, 0xac, 0xe2, 0x24, 0xb1,
	0x6f, 0x82, 0x9a, 0x64, 0x32, 0xcf, 0x1d, 0xa7, 0xc7, 0xa7, 0xa0, 0x7d, 0xe1, 0x8e, 0x0b, 0x55,
	0x48, 0xed, 0xbc, 0x43, 0x7a, 0x6e, 0x96, 0xda, 0x2f, 0x00, 0x09, 0x49, 0xb1, 0x93, 0xc8, 0x7a,
	0x11, 0xe6, 0x5c, 0xcf, 0xb5, 0xb1, 0x10, 0x95, 0x0f, 0xce, 0x91, 0x54, 0x83, 0xda, 0x68, 0x62,
	0xd9, 0xc2, 0xee, 0xec, 0x5b, 0xff, 0x1a, 0xba, 0x3b, 0x38, 0x67, 0x81, 0x73, 0x03, 0x31, 0x5d,
	0xb2, 0x32, 0x63, 0xc9, 0x6a, 0xf9, 0x92, 0x35, 0x69, 0xc9, 0x3d, 0x40, 0x62, 0xc9, 0x4c, 0x95,
	0x42, 0xad, 0x2d, 0x71, 0x90, 0x7c, 0x5b, 0xc9, 0xf9, 0x56, 0xff, 0xb3, 0x02, 0xdd, 0x6d, 0xe2,
	0x8f, 0x70, 0xf0, 0x39, 0x9e, 0xbe, 0xb0, 0xc6, 0xf1, 0x2b, 0x64, 0x47, 0x50, 0xa5, 0x7e, 0xe5,
	0x5c, 0xe8, 0x27, 0xd5, 0xe6, 0x84, 0xfe, 0x4e, 0x48, 0xcd, 0x07, 0x3c, 0x93, 0x32, 0xf9, 0xc4,
	0xb1, 0x90, 0x0c, 0xb5, 0x35, 0xe8, 0x5a, 0xe1, 0xb1, 0xe9, 0xb9, 0x66, 0x02, 0xe0, 0x77, 0x7a,
	0xd5, 0x0a, 0x8f, 0xbf, 0x70, 0x77, 0xcf, 0xa0, 0x1c, 0xae, 0xa6, 0x48, 0x52, 0x1c, 0x25, 0x54,
	0xd7, 0xba, 0x50, 0x21, 0x27, 0xec, 0x60, 0x50, 0x8d, 0x0a, 0x39, 0xd1, 0xd7, 0x01, 0x71, 0x65,
	0xb0, 0x93, 0xaa, 0x93, 0xca, 0xa7, 0x48, 0xf2, 0xe9, 0xff, 0x05, 0xdd, 0xdd, 0x30, 0x22, 0x13,
	0x2b, 0xc2, 0x07, 0xa7, 0x03, 0xf2, 0x12, 0xd3, 0x23, 0xda, 0x8b, 0x23, 0x3f, 0x8e, 0xc2, 0x34,
	0xa3, 0xd3, 0xc2, 0x59, 0x15, 0x44, 0x9e, 0xd4, 0x6f, 0x82, 0x4a, 0x5c, 0x09, 0x53, 0x61, 0x98,
	0x36, 0xa7, 0x71, 0xc8, 0x6b, 0x25, 0x13, 0xfd, 0x26, 0xd4, 0xc5, 0xba, 0x97, 0xa1, 0x11, 0x9d,
	0x9a, 0xa2, 0x54, 0xa7, 0xd9, 0xb4, 0x1e, 0xb1, 0x09, 0xfd, 0xf7, 0x0a, 0xd4, 0xe9, 0xf6, 0x3c,
	0x38, 0xfd, 0xc7, 0xca, 0xa6, 0x5d, 0x85, 0x46, 0xae, 0x2b, 0xf3, 0x40, 0x79, 0xd7, 0x48, 0x28,
	0xda, 0x75, 0x68, 0x8d, 0x3d, 0xfb, 0xd8, 0x8c, 0x88, 0xd8, 0x69, 0x9d, 0x07, 0xca, 0x3b, 0x46,
	0x93, 0xd2, 0x0e, 0xc8, 0x04, 0xeb, 0x7f, 0x53, 0x40, 0x1d, 0x90, 0x89, 0x3f, 0xc6, 0x42, 0xf6,
	0x35, 0xa8, 0x73, 0x11, 0x58, 0x2c, 0xb5, 0x37, 0xd5, 0xfe, 0xc1, 0x29, 0xcb, 0x99, 0x2c, 0xcd,
	0x8b, 0x39, 0xed, 0x0e, 0x34, 0x84, 0x32, 0xbd, 0x0a, 0x83, 0x75, 0xfa, 0x07, 0xa7, 0x5f, 0x30,
	0x0a, 0xc3, 0x25, 0xb3, 0xda, 0xfb, 0xa0, 0x46, 0x81, 0xe5, 0x86, 0x16, 0x3b, 0x09, 0xc3, 0x5e,
	0x95, 0xa1, 0x51, 0xff, 0x20, 0x23, 0xb2, 0x1f, 0xe4, 0x50, 0xaf, 0x97, 0x16, 0x65, 0xc5, 0xe7,
	0xce, 0x57, 0xbc, 0x7e, 0x56, 0xf1, 0x5f, 0x2b, 0xd0, 0x3a, 0x48, 0x2f, 0x8a, 0xf7, 0x41, 0x0d,
	0xf8, 0xa7, 0x29, 0x1d, 0x73, 0x6a, 0x5f, 0x3e, 0xe2, 0xda, 0x41, 0x36, 0xd0, 0xee, 0x43, 0xc3,
	0xc1, 0x91, 0x45, 0xc6, 0xa1, 0xa8, 0x63, 0x17, 0xfb, 0x29, 0xb7, 0x1d, 0x3e, 0xc1, 0x0d, 0x21,
	0x50, 0xda, 0x47, 0x00, 0x21, 0x0e, 0x92, 0x36, 0x51, 0x95, 0xfd, 0xa6, 0x97, 0xfd, 0x66, 0x90,
	0xce, 0xb1, 0x9f, 0x49, 0x58, 0x7d, 0x03, 0xe6, 0x0e, 0xd8, 0x95, 0x74, 0x05, 0x2a, 0xd1, 0x29,
	0x13, 0xad, 0xcc, 0x82, 0x95, 0xe8, 0x54, 0xff, 0xdf, 0x0a, 0x74, 0x93, 0x0a, 0x5e, 0xf8, 0xf3,
	0x67, 0xa4, 0xb6, 0xab, 0xd0, 0x1a, 0x5a, 0xa1, 0xe9, 0x07, 0xc4, 0x4e, 0xd2, 0x44, 0x73, 0x68,
	0x85, 0xfb, 0x74, 0x9c, 0x4c, 0x8e, 0xc9, 0x84, 0x44, 0x22, 0xc5, 0xd1, 0xc9, 0xa7, 0x74, 0x4c,
	0x37, 0x78, 0xe4, 0x31, 0x67, 0xa8, 0x46, 0x25, 0xf2, 0xb2, 0xcd, 0x5c, 0x97, 0x93, 0xcd, 0x5b,
	0xa0, 0xd1, 0xeb, 0xbb, 0x29, 0x9a, 0x64, 0xa6, 0x3d, 0x8a, 0xdd, 0x63, 0x91, 0x16, 0x10, 0x9d,
	0x11, 0x6d, 0xcf, 0x6d, 0x4a, 0xa7, 0x25, 0x0c, 0x43, 0x8f, 0x79, 0x45, 0x2c, 0xca, 0x6c, 0x4a,
	0x7a, 0xca, 0xcb, 0xe1, 0x2b, 0xd0, 0xb4, 0x47, 0x16, 0x71, 0x4d, 0xe2, 0x88, 0x02, 0xa7, 0xc1,
	0xc6, 0x4f, 0x1c, 0xfd, 0xff, 0x15, 0x58, 0x48, 0xec, 0x91, 0x39, 0xbb, 0xc0, 0x51, 0x39, 0xc3,
	0x91, 0x16, 0xaa, 0xc9, 0x81, 0x69, 0x9e, 0x88, 0xae, 0x29, 0xa4, 0xa4, 0x17, 0x79, 0x40, 0x20,
	0x6c, 0x94, 0x01, 0x8c, 0x3c, 0x20, 0x4c, 0x1a, 0x4d, 0x29, 0x69, 0xa0, 0xf7, 0xa1, 0x93, 0x09,
	0x46, 0x9d, 0xbb, 0x0c, 0x4c, 0x02, 0x61, 0x0c, 0x9e, 0xfc, 0x5a, 0x94, 0xc2, 0xac, 0xa0, 0x3f,
	0x85, 0x0b, 0xb2, 0x63, 0x7f, 0x59, 0x05, 0xa5, 0x13, 0x58, 0x4c, 0xb8, 0x9d, 0x5b, 0xe1, 0xa8,
	0xbf, 0xb8, 0xc2, 0xd1, 0x0d, 0xe8, 0x25, 0x4b, 0xbd, 0xaa, 0x86, 0x79, 0xdd, 0xd5, 0xf4, 0x9f,
	0x58, 0xd2, 0x1a, 0xba, 0x4f, 0x1c, 0xec, 0x46, 0x24, 0x9a, 0x6a, 0x1b, 0xd0, 0x24, 0xe2, 0x5b,
	0xec, 0x8f, 0x4e, 0x3f, 0x99, 0xe4, 0xf7, 0x73, 0x92, 0x41, 0x91, 0x3d, 0xb2, 0xc6, 0xd4, 0xf7,
	0xd8, 0x1c, 0x11, 0xc7, 0xc1, 0xae, 0x58, 0x60, 0x3e, 0xa5, 0x3f, 0x66, 0xe4, 0x3c, 0xf4, 0x84,
	0x84, 0xb1, 0x35, 0x16, 0x97, 0xd2, 0x0c, 0xfa, 0x82, 0x91, 0x4b, 0xdb, 0x2a, 0xb5, 0xb2, 0xb6,
	0x8a, 0x3e, 0x84, 0x2e, 0x15, 0x1d, 0x3b, 0xa9, 0xf0, 0xb3, 0x2b, 0xb9, 0x65, 0x00, 0x9f, 0x75,
	0x4e, 0xcc, 0xe4, 0x10, 0x57, 0x8d, 0x96, 0x9f, 0xf6, 0x52, 0x72, 0x46, 0xaa, 0x16, 0x8d, 0xf4,
	0xad, 0x02, 0x0b, 0x8f, 0x70, 0xb4, 0xbb, 0xbd, 0xf3, 0x58, 0x34, 0x5a, 0xe9, 0x6f, 0xde, 0xc0,
	0x52, 0xb7, 0x61, 0xde, 0xc7, 0x38, 0x30, 0xcf, 0x88, 0xd0, 0xa1, 0xe4, 0xac, 0xa5, 0x53, 0xa6,
	0x7b, 0xb5, 0x54, 0xf7, 0x77, 0xa1, 0x5b, 0x10, 0x87, 0xee, 0x13, 0x3e, 0x32, 0xb3, 0xfa, 0x13,
	0xc2, 0x14, 0xa0, 0xbf, 0x03, 0x9d, 0x01, 0x8e, 0xbe, 0xdc, 0xdc, 0x93, 0x2e, 0x91, 0xf2, 0x8d,
	0x46, 0x39, 0x73, 0xeb, 0xbe, 0x03, 0x9d, 0x3d, 0xd1, 0xa9, 0xde, 0x65, 0x3d, 0xdf, 0x4b, 0x50,
	0xcf, 0xed, 0x74, 0x31, 0xd2, 0xb7, 0x60, 0x3e, 0x01, 0x26, 0x99, 0xe1, 0x12, 0xd4, 0xbd, 0xa3,
	0xa3, 0x10, 0x27, 0xf7, 0x43, 0x31, 0x92, 0x58, 0x54, 0x72, 0x2c, 0x3e, 0x81, 0x6e, 0xc2, 0xe2,
	0x4b, 0x7f, 0xec, 0x59, 0x0e, 0x75, 0xa6, 0x6f, 0x4d, 0xe9, 0x67, 0xd2, 0x2f, 0x11, 0x43, 0x56,
	0x16, 0x5a, 0xe1, 0x48, 0xd8, 0x90, 0x7d, 0xeb, 0x6b, 0xd0, 0x1c, 0xe0, 0xf1, 0xd1, 0x01, 0x5d,
	0x3b, 0xf7, 0x4b, 0x45, 0xfa, 0xa5, 0x7e, 0x17, 0x16, 0x76, 0xf0, 0x61, 0x3c, 0x7c, 0x4a, 0xdc,
	0xe3, 0x1d, 0x6c, 0xf3, 0x97, 0x83, 0x45, 0xa8, 0x4f, 0x71, 0x68, 0xba, 0x1e, 0x5b, 0xa7, 0x69,
	0xcc, 0x4d, 0x71, 0xf8, 0xdc, 0xd3, 0x2f, 0x48, 0xd8, 0x47, 0x38, 0x1a, 0x44, 0x56, 0x84, 0xf5,
	0xbf, 0x54, 0x68, 0xc5, 0x2b, 0xa8, 0x8c, 0xc4, 0x34, 0xb2, 0xa6, 0x5e, 0x1c, 0x25, 0x35, 0x3f,
	0x1f, 0x25, 0xbd, 0x97, 0x4a, 0xd6, 0x7b, 0xb9, 0x04, 0xf5, 0x09, 0xeb, 0x8a, 0x0a, 0xa7, 0x8a,
	0x51, 0xae, 0xc5, 0x53, 0x9b, 0xd1, 0xe2, 0x99, 0x9b, 0xd5, 0xe2, 0x99, 0x79, 0xdb, 0xae, 0x9f,
	0x73, 0xdb, 0x5e, 0x06, 0x08, 0x70, 0x88, 0x23, 0x76, 0x13, 0x66, 0xe7, 0x45, 0xcb, 0x68, 0x31,
	0x0a, 0xbd, 0x74, 0xd2, 0xaa, 0x8b, 0x4f, 0x27, 0x3d, 0x81, 0x26, 0xd3, 0x4c, 0x65, 0xc4, 0xa4,
	0x8f, 0xfa, 0x16, 0x68, 0x81, 0xe8, 0x0b, 0x98, 0x47, 0xd6, 0x31, 0xbf, 0x55, 0x8b, 0xb7, 0x20,
	0x94, 0xcc, 0xec, 0x59, 0xc7, 0xec, 0x5a, 0xad, 0xdd, 0x85, 0x85, 0x14, 0xcd, 0x9a, 0x07, 0xbe,
	0x17, 0xb2, 0x7b, 0x72, 0xc7, 0x98, 0x4f, 0x26, 0x28, 0x70, 0xdf, 0x0b, 0xf5, 0x79, 0xe8, 0x48,
	0x36, 0xf6, 0x7c, 0x7d, 0x1f, 0xd4, 0x94, 0xf0, 0xd4, 0x1b, 0xb2, 0x0b, 0x3e, 0x3e, 0xc1, 0xe3,
	0xe4, 0x35, 0x81, 0x0d, 0xa8, 0x79, 0x0f, 0x63, 0xfb, 0x18, 0x47, 0xc2, 0xe6, 0x62, 0xc4, 0x6e,
	0xf3, 0xf8, 0x34, 0x12, 0x46, 0x67, 0xdf, 0xfa, 0x23, 0xb8, 0x90, 0x72, 0x7c, 0x86, 0x27, 0x5e,
	0x30, 0x35, 0x30, 0x8f, 0x39, 0x39, 0x81, 0x74, 0xb2, 0x04, 0x32, 0x2b, 0x6e, 0x37, 0x60, 0xbe,
	0xc0, 0x88, 0xb9, 0x99, 0x7d, 0x25, 0x01, 0xc1, 0x47, 0xfa, 0x7f, 0xc0, 0xc5, 0x02, 0xf4, 0xab,
	0x80, 0x44, 0xf8, 0xfc, 0x45, 0x05, 0xa7, 0x8a, 0xcc, 0x49, 0xbc, 0xa6, 0x84, 0x23, 0x71, 0x5b,
	0xe4, 0x03, 0xfd, 0x6d, 0x49, 0xa7, 0x3d, 0x4a, 0x49, 0x37, 0x6d, 0x88, 0xed, 0xc8, 0x4b, 0x76,
	0xb8, 0x18, 0xdd, 0xfd, 0x71, 0x11, 0xda, 0xe2, 0x1c, 0x61, 0x75, 0xd8, 0x0a, 0x5c, 0x92, 0x86,
	0x66, 0xf6, 0x60, 0x8a, 0xfe, 0x69, 0xa9, 0xf6, 0xed, 0x1f, 0x7a, 0x8a, 0xb6, 0x94, 0x5e, 0x9e,
	0x19, 0x62, 0x9f, 0xb8, 0x43, 0xa4, 0x88, 0xb9, 0x65, 0xb8, 0x20, 0xcf, 0x89, 0x57, 0x10, 0x54,
	0x59, 0xaa, 0x7d, 0x57, 0x32, 0x2d, 0xde, 0x39, 0x50, 0x55, 0x4c, 0xdf, 0x80, 0x45, 0x79, 0x3a,
	0x7d, 0x14, 0x42, 0x35, 0xc1, 0xbe, 0x20, 0x5c, 0xd6, 0x2e, 0x45, 0x73, 0x02, 0x71, 0x07, 0xae,
	0xe4, 0x56, 0x90, 0x13, 0x17, 0xaa, 0x2f, 0x35, 0x29, 0xe8, 0x8f, 0x14, 0xb8, 0x0e, 0x4b, 0x65,
	0x40, 0x9e, 0x75, 0x50, 0x43, 0x42, 0x6e, 0xc0, 0xd5, 0x32, 0xa4, 0x48, 0x71, 0xa8, 0xb9, 0xd4,
	0xfc, 0x2e, 0x81, 0x16, 0xe4, 0xcb, 0x5e, 0x23, 0x50, 0xab, 0xdc, 0x40, 0xc9, 0x34, 0x08, 0x0b,
	0xe8, 0xd0, 0x2b, 0x30, 0x48, 0x8f, 0x05, 0xd4, 0x16, 0x2c, 0x0a, 0x56, 0xca, 0x00, 0xaa, 0x60,
	0x52, 0x90, 0x22, 0xeb, 0x22, 0xa3, 0x8e, 0x60, 0x71, 0x13, 0x2e, 0xcb, 0x08, 0xa9, 0xa7, 0x8a,
	0xba, 0x02, 0x72, 0x0d, 0xb4, 0x9c, 0x27, 0x59, 0xf1, 0x8b, 0xe6, 0xc5, 0xec, 0x5a, 0x5e, 0x4e,
	0xf9, 0xc2, 0x83, 0xd0, 0x52, 0x9d, 0x62, 0x9a, 0x8a, 0x76, 0x1d, 0x2e, 0xe6, 0x2c, 0x27, 0x9e,
	0xd7, 0xd1, 0x82, 0x10, 0xf4, 0x36, 0x5c, 0x2b, 0x44, 0x52, 0xee, 0x31, 0x09, 0x69, 0x29, 0xae,
	0x57, 0x8a, 0xdb, 0xb2, 0x8f, 0xd1, 0x05, 0xee, 0xa9, 0xdf, 0x95, 0xc8, 0xcc, 0x1f, 0x97, 0xd0,
	0xc5, 0x72, 0xbb, 0xa5, 0xe5, 0x2b, 0x5a, 0x14, 0xcb, 0x5c, 0x85, 0x85, 0x3c, 0x80, 0xf2, 0xbf,
	0x94, 0x6a, 0x9c, 0x8b, 0x97, 0x7c, 0xcf, 0x00, 0x5d, 0x16, 0xa8, 0x82, 0xff, 0xe4, 0x57, 0x59,
	0xd4, 0x13, 0x98, 0xd5, 0x7c, 0x88, 0xe6, 0x1e, 0x6a, 0xd1, 0x95, 0x72, 0x50, 0xee, 0x11, 0x0f,
	0x2d, 0x09, 0x81, 0x57, 0xf3, 0x1a, 0xa5, 0x4f, 0x77, 0xe8, 0xaa, 0x64, 0x94, 0x42, 0x34, 0x64,
	0xaf, 0xb1, 0xe8, 0x5a, 0xf9, 0xae, 0xca, 0x1e, 0x49, 0xd0, 0x72, 0x79, 0xd4, 0x26, 0xd3, 0xd7,
	0xd3, 0xa8, 0xcd, 0xf9, 0x39, 0x39, 0x81, 0xd1, 0x8a, 0xb4, 0x8b, 0x0a, 0x96, 0x91, 0xdb, 0xd2,
	0x48, 0x2f, 0xb7, 0x71, 0xbe, 0x55, 0x8d, 0x56, 0xcb, 0xc3, 0x3b, 0x6b, 0x5f, 0xa3, 0xb5, 0xf2,
	0xf0, 0x96, 0xea, 0x7b, 0x74, 0xbb, 0xdc, 0xbe, 0xb9, 0xa2, 0x1d, 0xdd, 0x11, 0xa0, 0x42, 0x7c,
	0x16, 0xcb, 0x6d, 0xb4, 0x2e, 0x24, 0xba, 0x03, 0xcb, 0xb9, 0xf8, 0x2c, 0x3e, 0x65, 0xa2, 0x8d,
	0x14, 0x78, 0xa5, 0x1c, 0x48, 0xa5, 0xbf, 0x2b, 0x39, 0xed, 0x76, 0xc1, 0x12, 0xb9, 0x56, 0x0d,
	0xba, 0x27, 0xed, 0x30, 0x2d, 0x1f, 0xb2, 0x6c, 0xfe, 0xad, 0xa5, 0xfa, 0x77, 0x7c, 0xbe, 0x60,
	0xd1, 0x7c, 0x07, 0x1f, 0xbd, 0x5d, 0x6e, 0x2f, 0xa9, 0x15, 0x8d, 0xfa, 0xe5, 0x99, 0x5b, 0x34,
	0xa5, 0xd1, 0xfd, 0x72, 0x4b, 0x15, 0x9b, 0x50, 0xe8, 0x9d, 0x74, 0x27, 0x17, 0x3c, 0x2c, 0x77,
	0x0d, 0xd1, 0xbb, 0xa9, 0x5e, 0xeb, 0x79, 0x7e, 0xc5, 0xae, 0x25, 0xda, 0x4c, 0x35, 0x2c, 0x70,
	0xcc, 0xf7, 0x21, 0xd1, 0x7b, 0xb3, 0x38, 0x16, 0x9b, 0x87, 0xe8, 0xfd, 0x94, 0xa3, 0x5e, 0xcc,
	0x6d, 0xd9, 0xbd, 0x08, 0x7d, 0x50, 0x1e, 0xa9, 0xf9, 0x0b, 0x08, 0xfa, 0x50, 0x68, 0x5b, 0xb0,
	0xab, 0xf4, 0xef, 0x46, 0xe8, 0x9f, 0x05, 0xa3, 0x75, 0xb8, 0x9e, 0x53, 0xf4, 0xcc, 0x43, 0x25,
	0xfa, 0x48, 0x20, 0x6f, 0xe5, 0x8f, 0xa1, 0xc2, 0xbb, 0x22, 0xfa, 0x17, 0xb1, 0x66, 0x71, 0x0f,
	0xe5, 0x9a, 0x17, 0xe8, 0x41, 0x7a, 0x4c, 0x2e, 0x97, 0xa1, 0xb2, 0x9c, 0xf8, 0xaf, 0x69, 0x8a,
	0xb9, 0x52, 0x0e, 0xa4, 0xde, 0xff, 0xb7, 0x72, 0x6e, 0x67, 0x2e, 0x49, 0xe8, 0xe3, 0x19, 0x1b,
	0x3c, 0x8f, 0xfa, 0xa4, 0x7c, 0xcd, 0xdc, 0x75, 0x05, 0x7d, 0x2a, 0x58, 0x6d, 0xc0, 0x8d, 0x59,
	0x7a, 0x26, 0x2e, 0xfd, 0x4c, 0x40, 0xef, 0xc1, 0xcd, 0x32, 0x68, 0x7e, 0xcf, 0x6f, 0x09, 0x70,
	0x1f, 0xd6, 0xca, 0xc0, 0x67, 0xf6, 0xfe, 0x43, 0x21, 0xec, 0xbd, 0xbc, 0xee, 0x67, 0xee, 0x15,
	0xc8, 0x59, 0x6a, 0x7e, 0x9f, 0x6c, 0xeb, 0x3b, 0x33, 0xc0, 0xc9, 0xc5, 0x02, 0xe1, 0xa5, 0xda,
	0xf7, 0x25, 0x86, 0xca, 0xdf, 0x35, 0xd0, 0xd1, 0x52, 0xed, 0x87, 0x12, 0x43, 0xe5, 0xaa, 0x65,
	0x34, 0x14, 0xac, 0x0a, 0xe1, 0x2c, 0x57, 0xd0, 0x68, 0x24, 0x18, 0x15, 0x8c, 0x59, 0x52, 0x13,
	0x23, 0x57, 0xb0, 0x2b, 0x84, 0x61, 0x01, 0x8a, 0x3c, 0xc1, 0xf1, 0x2e, 0xac, 0x9c, 0x03, 0x63,
	0x15, 0x2f, 0xf2, 0x05, 0xcb, 0x59, 0xab, 0x67, 0xd5, 0x2b, 0xfa, 0x9a, 0x43, 0x1f, 0xbe, 0x0f,
	0xab, 0xb6, 0x37, 0xe9, 0x87, 0x56, 0xe4, 0x85, 0x23, 0x32, 0xb6, 0x0e, 0xc3, 0x7e, 0x14, 0xe0,
	0x97, 0x5e, 0xd0, 0x1f, 0x93, 0x43, 0xfe, 0x6f, 0x7e, 0x87, 0xf1, 0xd1, 0xc3, 0xce, 0x01, 0x23,
	0x0a, 0xae, 0x7f, 0x0f, 0x00, 0x00, 0xff, 0xff, 0x2a, 0xe4, 0xc0, 0x85, 0x16, 0x28, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = mathPtr.Inf

// *
// Mapping between Trezor wire identifier (uint) and a protobuf message
type MessagesType int32

const (
	MessagesType_MessagesType_Initialize               MessagesType = 0
	MessagesType_MessagesType_Ping                     MessagesType = 1
	MessagesType_MessagesType_Success                  MessagesType = 2
	MessagesType_MessagesType_Failure                  MessagesType = 3
	MessagesType_MessagesType_ChangePin                MessagesType = 4
	MessagesType_MessagesType_WipeDevice               MessagesType = 5
	MessagesType_MessagesType_FirmwareErase            MessagesType = 6
	MessagesType_MessagesType_FirmwareUpload           MessagesType = 7
	MessagesType_MessagesType_FirmwareRequest          MessagesType = 8
	MessagesType_MessagesType_GetEntropy               MessagesType = 9
	MessagesType_MessagesType_Entropy                  MessagesType = 10
	MessagesType_MessagesType_GetPublicKey             MessagesType = 11
	MessagesType_MessagesType_PublicKey                MessagesType = 12
	MessagesType_MessagesType_LoadDevice               MessagesType = 13
	MessagesType_MessagesType_ResetDevice              MessagesType = 14
	MessagesType_MessagesType_SignTx                   MessagesType = 15
	MessagesType_MessagesType_SimpleSignTx             MessagesType = 16
	MessagesType_MessagesType_Features                 MessagesType = 17
	MessagesType_MessagesType_PinMatrixRequest         MessagesType = 18
	MessagesType_MessagesType_PinMatrixAck             MessagesType = 19
	MessagesType_MessagesType_Cancel                   MessagesType = 20
	MessagesType_MessagesType_TxRequest                MessagesType = 21
	MessagesType_MessagesType_TxAck                    MessagesType = 22
	MessagesType_MessagesType_CipherKeyValue           MessagesType = 23
	MessagesType_MessagesType_ClearSession             MessagesType = 24
	MessagesType_MessagesType_ApplySettings            MessagesType = 25
	MessagesType_MessagesType_ButtonRequest            MessagesType = 26
	MessagesType_MessagesType_ButtonAck                MessagesType = 27
	MessagesType_MessagesType_ApplyFlags               MessagesType = 28
	MessagesType_MessagesType_GetAddress               MessagesType = 29
	MessagesType_MessagesType_Address                  MessagesType = 30
	MessagesType_MessagesType_SelfTest                 MessagesType = 32
	MessagesType_MessagesType_BackupDevice             MessagesType = 34
	MessagesType_MessagesType_EntropyRequest           MessagesType = 35
	MessagesType_MessagesType_EntropyAck               MessagesType = 36
	MessagesType_MessagesType_SignMessage              MessagesType = 38
	MessagesType_MessagesType_VerifyMessage            MessagesType = 39
	MessagesType_MessagesType_MessageSignature         MessagesType = 40
	MessagesType_MessagesType_PassphraseRequest        MessagesType = 41
	MessagesType_MessagesType_PassphraseAck            MessagesType = 42
	MessagesType_MessagesType_EstimateTxSize           MessagesType = 43
	MessagesType_MessagesType_TxSize                   MessagesType = 44
	MessagesType_MessagesType_RecoveryDevice           MessagesType = 45
	MessagesType_MessagesType_WordRequest              MessagesType = 46
	MessagesType_MessagesType_WordAck                  MessagesType = 47
	MessagesType_MessagesType_CipheredKeyValue         MessagesType = 48
	MessagesType_MessagesType_EncryptMessage           MessagesType = 49
	MessagesType_MessagesType_EncryptedMessage         MessagesType = 50
	MessagesType_MessagesType_DecryptMessage           MessagesType = 51
	MessagesType_MessagesType_DecryptedMessage         MessagesType = 52
	MessagesType_MessagesType_SignIdentity             MessagesType = 53
	MessagesType_MessagesType_SignedIdentity           MessagesType = 54
	MessagesType_MessagesType_GetFeatures              MessagesType = 55
	MessagesType_MessagesType_BgmchainGetAddress       MessagesType = 56
	MessagesType_MessagesType_BgmchainAddress          MessagesType = 57
	MessagesType_MessagesType_BgmchainSignTx           MessagesType = 58
	MessagesType_MessagesType_BgmchainTxRequest        MessagesType = 59
	MessagesType_MessagesType_BgmchainTxAck            MessagesType = 60
	MessagesType_MessagesType_GetECDHSessionKey        MessagesType = 61
	MessagesType_MessagesType_ECDHSessionKey           MessagesType = 62
	MessagesType_MessagesType_SetU2FCounter            MessagesType = 63
	MessagesType_MessagesType_BgmchainSignMessage      MessagesType = 64
	MessagesType_MessagesType_BgmchainVerifyMessage    MessagesType = 65
	MessagesType_MessagesType_BgmchainMessageSignature MessagesType = 66
	MessagesType_MessagesType_DebugLinkDecision        MessagesType = 100
	MessagesType_MessagesType_DebugLinkGetState        MessagesType = 101
	MessagesType_MessagesType_DebugLinkState           MessagesType = 102
	MessagesType_MessagesType_DebugLinkStop            MessagesType = 103
	MessagesType_MessagesType_DebugLinkbgmlogs             MessagesType = 104
	MessagesType_MessagesType_DebugLinkMemoryRead      MessagesType = 110
	MessagesType_MessagesType_DebugLinkMemory          MessagesType = 111
	MessagesType_MessagesType_DebugLinkMemoryWrite     MessagesType = 112
	MessagesType_MessagesType_DebugLinkFlashErase      MessagesType = 113
)

var MessagesType_name = map[int32]string{
	0:   "MessagesType_Initialize",
	1:   "MessagesType_Ping",
	2:   "MessagesType_Success",
	3:   "MessagesType_Failure",
	4:   "MessagesType_ChangePin",
	5:   "MessagesType_WipeDevice",
	6:   "MessagesType_FirmwareErase",
	7:   "MessagesType_FirmwareUpload",
	8:   "MessagesType_FirmwareRequest",
	9:   "MessagesType_GetEntropy",
	10:  "MessagesType_Entropy",
	11:  "MessagesType_GetPublicKey",
	12:  "MessagesType_PublicKey",
	13:  "MessagesType_LoadDevice",
	14:  "MessagesType_ResetDevice",
	15:  "MessagesType_SignTx",
	16:  "MessagesType_SimpleSignTx",
	17:  "MessagesType_Features",
	18:  "MessagesType_PinMatrixRequest",
	19:  "MessagesType_PinMatrixAck",
	20:  "MessagesType_Cancel",
	21:  "MessagesType_TxRequest",
	22:  "MessagesType_TxAck",
	23:  "MessagesType_CipherKeyValue",
	24:  "MessagesType_ClearSession",
	25:  "MessagesType_ApplySettings",
	26:  "MessagesType_ButtonRequest",
	27:  "MessagesType_ButtonAck",
	28:  "MessagesType_ApplyFlags",
	29:  "MessagesType_GetAddress",
	30:  "MessagesType_Address",
	32:  "MessagesType_SelfTest",
	34:  "MessagesType_BackupDevice",
	35:  "MessagesType_EntropyRequest",
	36:  "MessagesType_EntropyAck",
	38:  "MessagesType_SignMessage",
	39:  "MessagesType_VerifyMessage",
	40:  "MessagesType_MessageSignature",
	41:  "MessagesType_PassphraseRequest",
	42:  "MessagesType_PassphraseAck",
	43:  "MessagesType_EstimateTxSize",
	44:  "MessagesType_TxSize",
	45:  "MessagesType_RecoveryDevice",
	46:  "MessagesType_WordRequest",
	47:  "MessagesType_WordAck",
	48:  "MessagesType_CipheredKeyValue",
	49:  "MessagesType_EncryptMessage",
	50:  "MessagesType_EncryptedMessage",
	51:  "MessagesType_DecryptMessage",
	52:  "MessagesType_DecryptedMessage",
	53:  "MessagesType_SignIdentity",
	54:  "MessagesType_SignedIdentity",
	55:  "MessagesType_GetFeatures",
	56:  "MessagesType_BgmchainGetAddress",
	57:  "MessagesType_BgmchainAddress",
	58:  "MessagesType_BgmchainSignTx",
	59:  "MessagesType_BgmchainTxRequest",
	60:  "MessagesType_BgmchainTxAck",
	61:  "MessagesType_GetECDHSessionKey",
	62:  "MessagesType_ECDHSessionKey",
	63:  "MessagesType_SetU2FCounter",
	64:  "MessagesType_BgmchainSignMessage",
	65:  "MessagesType_BgmchainVerifyMessage",
	66:  "MessagesType_BgmchainMessageSignature",
	100: "MessagesType_DebugLinkDecision",
	101: "MessagesType_DebugLinkGetState",
	102: "MessagesType_DebugLinkState",
	103: "MessagesType_DebugLinkStop",
	104: "MessagesType_DebugLinkbgmlogs",
	110: "MessagesType_DebugLinkMemoryRead",
	111: "MessagesType_DebugLinkMemory",
	112: "MessagesType_DebugLinkMemoryWrite",
	113: "MessagesType_DebugLinkFlashErase",
}
var MessagesType_value = map[string]int32{
	"MessagesType_Initialize":               0,
	"MessagesType_Ping":                     1,
	"MessagesType_Success":                  2,
	"MessagesType_Failure":                  3,
	"MessagesType_ChangePin":                4,
	"MessagesType_WipeDevice":               5,
	"MessagesType_FirmwareErase":            6,
	"MessagesType_FirmwareUpload":           7,
	"MessagesType_FirmwareRequest":          8,
	"MessagesType_GetEntropy":               9,
	"MessagesType_Entropy":                  10,
	"MessagesType_GetPublicKey":             11,
	"MessagesType_PublicKey":                12,
	"MessagesType_LoadDevice":               13,
	"MessagesType_ResetDevice":              14,
	"MessagesType_SignTx":                   15,
	"MessagesType_SimpleSignTx":             16,
	"MessagesType_Features":                 17,
	"MessagesType_PinMatrixRequest":         18,
	"MessagesType_PinMatrixAck":             19,
	"MessagesType_Cancel":                   20,
	"MessagesType_TxRequest":                21,
	"MessagesType_TxAck":                    22,
	"MessagesType_CipherKeyValue":           23,
	"MessagesType_ClearSession":             24,
	"MessagesType_ApplySettings":            25,
	"MessagesType_ButtonRequest":            26,
	"MessagesType_ButtonAck":                27,
	"MessagesType_ApplyFlags":               28,
	"MessagesType_GetAddress":               29,
	"MessagesType_Address":                  30,
	"MessagesType_SelfTest":                 32,
	"MessagesType_BackupDevice":             34,
	"MessagesType_EntropyRequest":           35,
	"MessagesType_EntropyAck":               36,
	"MessagesType_SignMessage":              38,
	"MessagesType_VerifyMessage":            39,
	"MessagesType_MessageSignature":         40,
	"MessagesType_PassphraseRequest":        41,
	"MessagesType_PassphraseAck":            42,
	"MessagesType_EstimateTxSize":           43,
	"MessagesType_TxSize":                   44,
	"MessagesType_RecoveryDevice":           45,
	"MessagesType_WordRequest":              46,
	"MessagesType_WordAck":                  47,
	"MessagesType_CipheredKeyValue":         48,
	"MessagesType_EncryptMessage":           49,
	"MessagesType_EncryptedMessage":         50,
	"MessagesType_DecryptMessage":           51,
	"MessagesType_DecryptedMessage":         52,
	"MessagesType_SignIdentity":             53,
	"MessagesType_SignedIdentity":           54,
	"MessagesType_GetFeatures":              55,
	"MessagesType_BgmchainGetAddress":       56,
	"MessagesType_BgmchainAddress":          57,
	"MessagesType_BgmchainSignTx":           58,
	"MessagesType_BgmchainTxRequest":        59,
	"MessagesType_BgmchainTxAck":            60,
	"MessagesType_GetECDHSessionKey":        61,
	"MessagesType_ECDHSessionKey":           62,
	"MessagesType_SetU2FCounter":            63,
	"MessagesType_BgmchainSignMessage":      64,
	"MessagesType_BgmchainVerifyMessage":    65,
	"MessagesType_BgmchainMessageSignature": 66,
	"MessagesType_DebugLinkDecision":        100,
	"MessagesType_DebugLinkGetState":        101,
	"MessagesType_DebugLinkState":           102,
	"MessagesType_DebugLinkStop":            103,
	"MessagesType_DebugLinkbgmlogs":             104,
	"MessagesType_DebugLinkMemoryRead":      110,
	"MessagesType_DebugLinkMemory":          111,
	"MessagesType_DebugLinkMemoryWrite":     112,
	"MessagesType_DebugLinkFlashErase":      113,
}

// *
// Request: Reset device to default state and ask for device details
// @next Features
type Initialize struct {
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *Initialize) Reset()                    { *m = Initialize{} }
func (mPtr *Initialize) String() string            { return proto.CompactTextString(m) }
func (*Initialize) ProtoMessage()               {}
func (*Initialize) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

// *
// Request: Ask for device details (no device reset)
// @next Features
type GetFeatures struct {
	XXX_unrecognized []byte `json:"-"`
}

func (mPtr *GetFeatures) Reset()                    { *m = GetFeatures{} }
func (mPtr *GetFeatures) String() string            { return proto.CompactTextString(m) }
func (*GetFeatures) ProtoMessage()               {}
func (*GetFeatures) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

func (x MessagesType) Enum() *MessagesType {
	p := new(MessagesType)
	*p = x
	return p
}
func (x MessagesType) String() string {
	return proto.EnumName(MessagesType_name, int32(x))
}
func (x *MessagesType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(MessagesType_value, data, "MessagesType")
	if err != nil {
		return err
	}
	*x = MessagesType(value)
	return nil
}
func (MessagesType) EnumDescriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

// *
// Response: Reports various information about the device
// @prev Initialize
// @prev GetFeatures
type Features struct {
	Vendor               *string     `protobuf:"bytes,1,opt,name=vendor" json:"vendor,omitempty"`
	MajorVersion         *uint32     `protobuf:"varint,2,opt,name=major_version,json=majorVersion" json:"major_version,omitempty"`
	MinorVersion         *uint32     `protobuf:"varint,3,opt,name=minor_version,json=minorVersion" json:"minor_version,omitempty"`
	PatchVersion         *uint32     `protobuf:"varint,4,opt,name=patch_version,json=patchVersion" json:"patch_version,omitempty"`
	BootloaderMode       *bool       `protobuf:"varint,5,opt,name=bootloader_mode,json=bootloaderMode" json:"bootloader_mode,omitempty"`
	DeviceId             *string     `protobuf:"bytes,6,opt,name=device_id,json=deviceId" json:"device_id,omitempty"`
	PinProtection        *bool       `protobuf:"varint,7,opt,name=pin_protection,json=pinProtection" json:"pin_protection,omitempty"`
	PassphraseProtection *bool       `protobuf:"varint,8,opt,name=passphrase_protection,json=passphraseProtection" json:"passphrase_protection,omitempty"`
	Language             *string     `protobuf:"bytes,9,opt,name=language" json:"language,omitempty"`
	Label                *string     `protobuf:"bytes,10,opt,name=label" json:"label,omitempty"`
	Coins                []*CoinsType `protobuf:"bytes,11,rep,name=coins" json:"coins,omitempty"`
	Initialized          *bool       `protobuf:"varint,12,opt,name=initialized" json:"initialized,omitempty"`
	Revision             []byte      `protobuf:"bytes,13,opt,name=revision" json:"revision,omitempty"`
	BootloaderHash       []byte      `protobuf:"bytes,14,opt,name=bootloader_hash,json=bootloaderHash" json:"bootloader_hash,omitempty"`
	Imported             *bool       `protobuf:"varint,15,opt,name=imported" json:"imported,omitempty"`
	PinCached            *bool       `protobuf:"varint,16,opt,name=pin_cached,json=pinCached" json:"pin_cached,omitempty"`
	PassphraseCached     *bool       `protobuf:"varint,17,opt,name=passphrase_cached,json=passphraseCached" json:"passphrase_cached,omitempty"`
	FirmwarePresent      *bool       `protobuf:"varint,18,opt,name=firmware_present,json=firmwarePresent" json:"firmware_present,omitempty"`
	NeedsBackup          *bool       `protobuf:"varint,19,opt,name=needs_backup,json=needsBackup" json:"needs_backup,omitempty"`
	Flags                *uint32     `protobuf:"varint,20,opt,name=flags" json:"flags,omitempty"`
	XXX_unrecognized     []byte      `json:"-"`
}

func (mPtr *Features) Reset()                    { *m = Features{} }
func (mPtr *Features) String() string            { return proto.CompactTextString(m) }
func (*Features) ProtoMessage()               {}
func (*Features) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{2} }

func (mPtr *Features) GetMinorVersion() uint32 {
	if m != nil && mPtr.MinorVersion != nil {
		return *mPtr.MinorVersion
	}
	return 0
}

func (mPtr *Features) GetPatchVersion() uint32 {
	if m != nil && mPtr.PatchVersion != nil {
		return *mPtr.PatchVersion
	}
	return 0
}