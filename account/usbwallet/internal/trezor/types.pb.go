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
import google_protobuf "github.com/golang/protobuf/protoc-gen-go/descriptor"

func (x *FailuresType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(FailuresType_value, data, "FailuresType")
	if err != nil {
		return err
	}
	*x = FailuresType(value)
	return nil
}
func (FailuresType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

// *
// Type of script which will be used for transaction output
// @used_in TxOutputType
type OutputScriptType int32

const (
	OutputScriptType_PAYTOADDRESS     OutputScriptType = 0
	OutputScriptType_PAYTOSCRIPTHASH  OutputScriptType = 1
	OutputScriptType_PAYTOMULTISIG    OutputScriptType = 2
	OutputScriptType_PAYTOOPRETURN    OutputScriptType = 3
	OutputScriptType_PAYTOWITNESS     OutputScriptType = 4
	OutputScriptType_PAYTOP2SHWITNESS OutputScriptType = 5
)

var OutputScriptType_name = map[int32]string{
	0: "PAYTOADDRESS",
	1: "PAYTOSCRIPTHASH",
	2: "PAYTOMULTISIG",
	3: "PAYTOOPRETURN",
	4: "PAYTOWITNESS",
	5: "PAYTOP2SHWITNESS",
}
var OutputScriptType_value = map[string]int32{
	"PAYTOADDRESS":     0,
	"PAYTOSCRIPTHASH":  1,
	"PAYTOMULTISIG":    2,
	"PAYTOOPRETURN":    3,
	"PAYTOWITNESS":     4,
	"PAYTOP2SHWITNESS": 5,
}

func (x OutputScriptType) Enum() *OutputScriptType {
	p := new(OutputScriptType)
	*p = x
	return p
}
func (x OutputScriptType) String() string {
	return proto.EnumName(OutputScriptType_name, int32(x))
}
func (x *OutputScriptType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(OutputScriptType_value, data, "OutputScriptType")
	if err != nil {
		return err
	}
	*x = OutputScriptType(value)
	return nil
}
func (OutputScriptType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

// *
// Type of script which will be used for transaction output
// @used_in TxInputType
type InputScriptType int32

const (
	InputScriptType_SPENDADDRESS     InputScriptType = 0
	InputScriptType_SPENDMULTISIG    InputScriptType = 1
	InputScriptType_EXTERNAL         InputScriptType = 2
	InputScriptType_SPENDWITNESS     InputScriptType = 3
	InputScriptType_SPENDP2SHWITNESS InputScriptType = 4
)

var InputScriptType_name = map[int32]string{
	0: "SPENDADDRESS",
	1: "SPENDMULTISIG",
	2: "EXTERNAL",
	3: "SPENDWITNESS",
	4: "SPENDP2SHWITNESS",
}
var InputScriptType_value = map[string]int32{
	"SPENDADDRESS":     0,
	"SPENDMULTISIG":    1,
	"EXTERNAL":         2,
	"SPENDWITNESS":     3,
	"SPENDP2SHWITNESS": 4,
}

func (x InputScriptType) Enum() *InputScriptType {
	p := new(InputScriptType)
	*p = x
	return p
}
func (x InputScriptType) String() string {
	return proto.EnumName(InputScriptType_name, int32(x))
}
func (x *InputScriptType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(InputScriptType_value, data, "InputScriptType")
	if err != nil {
		return err
	}
	*x = InputScriptType(value)
	return nil
}
func (InputScriptType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

// *
// Type of information required by transaction signing process
// @used_in TxRequest
type RequestType int32

const (
	RequestType_TXINPUT     RequestType = 0
	RequestType_TXOUTPUT    RequestType = 1
	RequestType_TXMETA      RequestType = 2
	RequestType_TXFINISHED  RequestType = 3
	RequestType_TXEXTRADATA RequestType = 4
)

var RequestType_name = map[int32]string{
	0: "TXINPUT",
	1: "TXOUTPUT",
	2: "TXMETA",
	3: "TXFINISHED",
	4: "TXEXTRADATA",
}
var RequestType_value = map[string]int32{
	"TXINPUT":     0,
	"TXOUTPUT":    1,
	"TXMETA":      2,
	"TXFINISHED":  3,
	"TXEXTRADATA": 4,
}

func (x RequestType) Enum() *RequestType {
	p := new(RequestType)
	*p = x
	return p
}
func (x RequestType) String() string {
	return proto.EnumName(RequestType_name, int32(x))
}
func (x *RequestType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(RequestType_value, data, "RequestType")
	if err != nil {
		return err
	}
	*x = RequestType(value)
	return nil
}
func (RequestType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

// *
// Type of button request
// @used_in ButtonRequest
type ButtonRequestType int32

const (
	ButtonRequestType_ButtonRequest_Other            ButtonRequestType = 1
	ButtonRequestType_ButtonRequest_FeeOverThreshold ButtonRequestType = 2
	ButtonRequestType_ButtonRequest_ConfirmOutput    ButtonRequestType = 3
	ButtonRequestType_ButtonRequest_ResetDevice      ButtonRequestType = 4
	ButtonRequestType_ButtonRequest_ConfirmWord      ButtonRequestType = 5
	ButtonRequestType_ButtonRequest_WipeDevice       ButtonRequestType = 6
	ButtonRequestType_ButtonRequest_ProtectCall      ButtonRequestType = 7
	ButtonRequestType_ButtonRequest_SignTx           ButtonRequestType = 8
	ButtonRequestType_ButtonRequest_FirmwareCheck    ButtonRequestType = 9
	ButtonRequestType_ButtonRequest_Address          ButtonRequestType = 10
	ButtonRequestType_ButtonRequest_PublicKey        ButtonRequestType = 11
)

var ButtonRequestType_name = map[int32]string{
	1:  "ButtonRequest_Other",
	2:  "ButtonRequest_FeeOverThreshold",
	3:  "ButtonRequest_ConfirmOutput",
	4:  "ButtonRequest_ResetDevice",
	5:  "ButtonRequest_ConfirmWord",
	6:  "ButtonRequest_WipeDevice",
	7:  "ButtonRequest_ProtectCall",
	8:  "ButtonRequest_SignTx",
	9:  "ButtonRequest_FirmwareCheck",
	10: "ButtonRequest_Address",
	11: "ButtonRequest_PublicKey",
}
var ButtonRequestType_value = map[string]int32{
	"ButtonRequest_Other":            1,
	"ButtonRequest_FeeOverThreshold": 2,
	"ButtonRequest_ConfirmOutput":    3,
	"ButtonRequest_ResetDevice":      4,
	"ButtonRequest_ConfirmWord":      5,
	"ButtonRequest_WipeDevice":       6,
	"ButtonRequest_ProtectCall":      7,
	"ButtonRequest_SignTx":           8,
	"ButtonRequest_FirmwareCheck":    9,
	"ButtonRequest_Address":          10,
	"ButtonRequest_PublicKey":        11,
}

func (x ButtonRequestType) Enum() *ButtonRequestType {
	p := new(ButtonRequestType)
	*p = x
	return p
}
func (x ButtonRequestType) String() string {
	return proto.EnumName(ButtonRequestType_name, int32(x))
}
func (x *ButtonRequestType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(ButtonRequestType_value, data, "ButtonRequestType")
	if err != nil {
		return err
	}
	*x = ButtonRequestType(value)
	return nil
}
func (ButtonRequestType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

// *
// Type of PIN request
// @used_in PinMatrixRequest
type PinMatrixRequestType int32

const (
	PinMatrixRequestType_PinMatrixRequestType_Current   PinMatrixRequestType = 1
	PinMatrixRequestType_PinMatrixRequestType_NewFirst  PinMatrixRequestType = 2
	PinMatrixRequestType_PinMatrixRequestType_NewSecond PinMatrixRequestType = 3
)

var PinMatrixRequestType_name = map[int32]string{
	1: "PinMatrixRequestType_Current",
	2: "PinMatrixRequestType_NewFirst",
	3: "PinMatrixRequestType_NewSecond",
}
var PinMatrixRequestType_value = map[string]int32{
	"PinMatrixRequestType_Current":   1,
	"PinMatrixRequestType_NewFirst":  2,
	"PinMatrixRequestType_NewSecond": 3,
}

func (x PinMatrixRequestType) Enum() *PinMatrixRequestType {
	p := new(PinMatrixRequestType)
	*p = x
	return p
}
func (x PinMatrixRequestType) String() string {
	return proto.EnumName(PinMatrixRequestType_name, int32(x))
}
func (x *PinMatrixRequestType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(PinMatrixRequestType_value, data, "PinMatrixRequestType")
	if err != nil {
		return err
	}
	*x = PinMatrixRequestType(value)
	return nil
}
func (PinMatrixRequestType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

// *
// Type of recovery procedure. These should be used as bitmask, e.g.,
// `RecoveryDeviceType_ScrambledWords | RecoveryDeviceType_Matrix`
// listing every method supported by the host computer.
//
// Note that ScrambledWords must be supported by every implementation
// for backward compatibility; there is no way to not support it.
//
// @used_in RecoveryDevice
type RecoveryDeviceType int32

const (
	// use powers of two when extending this field
	RecoveryDeviceType_RecoveryDeviceType_ScrambledWords RecoveryDeviceType = 0
	RecoveryDeviceType_RecoveryDeviceType_Matrix         RecoveryDeviceType = 1
)

var RecoveryDeviceType_name = map[int32]string{
	0: "RecoveryDeviceType_ScrambledWords",
	1: "RecoveryDeviceType_Matrix",
}
var RecoveryDeviceType_value = map[string]int32{
	"RecoveryDeviceType_ScrambledWords": 0,
	"RecoveryDeviceType_Matrix":         1,
}

func (x RecoveryDeviceType) Enum() *RecoveryDeviceType {
	p := new(RecoveryDeviceType)
	*p = x
	return p
}
func (x RecoveryDeviceType) String() string {
	return proto.EnumName(RecoveryDeviceType_name, int32(x))
}
func (x *RecoveryDeviceType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(RecoveryDeviceType_value, data, "RecoveryDeviceType")
	if err != nil {
		return err
	}
	*x = RecoveryDeviceType(value)
	return nil
}
func (RecoveryDeviceType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

// *
// Type of Recovery Word request
// @used_in WordRequest
type WordRequestType int32

const (
	WordRequestType_WordRequestType_Plain   WordRequestType = 0
	WordRequestType_WordRequestType_Matrix9 WordRequestType = 1
	WordRequestType_WordRequestType_Matrix6 WordRequestType = 2
)

var WordRequestType_name = map[int32]string{
	0: "WordRequestType_Plain",
	1: "WordRequestType_Matrix9",
	2: "WordRequestType_Matrix6",
}
var WordRequestType_value = map[string]int32{
	"WordRequestType_Plain":   0,
	"WordRequestType_Matrix9": 1,
	"WordRequestType_Matrix6": 2,
}

func (x WordRequestType) Enum() *WordRequestType {
	p := new(WordRequestType)
	*p = x
	return p
}
func (x WordRequestType) String() string {
	return proto.EnumName(WordRequestType_name, int32(x))
}
func (x *WordRequestType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(WordRequestType_value, data, "WordRequestType")
	if err != nil {
		return err
	}
	*x = WordRequestType(value)
	return nil
}
func (WordRequestType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

// *
// Structure representing BIP32 (hierarchical deterministic) node
// Used for imports of private key into the device and exporting public key out of device
// @used_in PublicKey
// @used_in LoadDevice
// @used_in DebugLinkState
// @used_in Storage
type HDNodeType struct {
	Depth            *uint32 `protobuf:"varint,1,req,name=depth" json:"depth,omitempty"`
	Fingerprint      *uint32 `protobuf:"varint,2,req,name=fingerprint" json:"fingerprint,omitempty"`
	ChildNum         *uint32 `protobuf:"varint,3,req,name=child_num,json=childNum" json:"child_num,omitempty"`
	ChainCode        []byte  `protobuf:"bytes,4,req,name=chain_code,json=chainCode" json:"chain_code,omitempty"`
	PrivateKey       []byte  `protobuf:"bytes,5,opt,name=private_key,json=privateKey" json:"private_key,omitempty"`
	PublicKey        []byte  `protobuf:"bytes,6,opt,name=public_key,json=publicKey" json:"public_key,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *HDNodeType) Reset()                    { *m = HDNodeType{} }
func (mPtr *HDNodeType) String() string            { return proto.CompactTextString(m) }
func (*HDNodeType) ProtoMessage()               {}
func (*HDNodeType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (mPtr *HDNodeType) GetDepth() uint32 {
	if m != nil && mPtr.Depth != nil {
		return *mPtr.Depth
	}
	return 0
}

func (mPtr *HDNodeType) GetFingerprint() uint32 {
	if m != nil && mPtr.Fingerprint != nil {
		return *mPtr.Fingerprint
	}
	return 0
}

func (mPtr *HDNodeType) GetChildNum() uint32 {
	if m != nil && mPtr.ChildNum != nil {
		return *mPtr.ChildNum
	}
	return 0
}

func (mPtr *HDNodeType) GetChainCode() []byte {
	if m != nil {
		return mPtr.ChainCode
	}
	return nil
}

func (mPtr *HDNodeType) GetPrivateKey() []byte {
	if m != nil {
		return mPtr.PrivateKey
	}
	return nil
}

func (mPtr *HDNodeType) GetPublicKey() []byte {
	if m != nil {
		return mPtr.PublicKey
	}
	return nil
}

type HDNodePathType struct {
	Node             *HDNodeType `protobuf:"bytes,1,req,name=node" json:"node,omitempty"`
	AddressN         []uint32    `protobuf:"varint,2,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	XXX_unrecognized []byte      `json:"-"`
}

func (mPtr *HDNodePathType) Reset()                    { *m = HDNodePathType{} }
func (mPtr *HDNodePathType) String() string            { return proto.CompactTextString(m) }
func (*HDNodePathType) ProtoMessage()               {}
func (*HDNodePathType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (mPtr *HDNodePathType) GetNode() *HDNodeType {
	if m != nil {
		return mPtr.Node
	}
	return nil
}

func (mPtr *HDNodePathType) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

// *
// Structure representing Coin
// @used_in Features
type CoinsType struct {
	CoinName            *string `protobuf:"bytes,1,opt,name=coin_name,json=coinName" json:"coin_name,omitempty"`
	CoinShortcut        *string `protobuf:"bytes,2,opt,name=coin_shortcut,json=coinShortcut" json:"coin_shortcut,omitempty"`
	AddressType         *uint32 `protobuf:"varint,3,opt,name=address_type,json=addressType,def=0" json:"address_type,omitempty"`
	MaxfeeKb            *uint64 `protobuf:"varint,4,opt,name=maxfee_kb,json=maxfeeKb" json:"maxfee_kb,omitempty"`
	AddressTypeP2Sh     *uint32 `protobuf:"varint,5,opt,name=address_type_p2sh,json=addressTypeP2sh,def=5" json:"address_type_p2sh,omitempty"`
	SignedMessageheaderPtr *string `protobuf:"bytes,8,opt,name=signed_message_headerPtr,json=signedMessageHeader" json:"signed_message_headerPtr,omitempty"`
	XpubMagic           *uint32 `protobuf:"varint,9,opt,name=xpub_magic,json=xpubMagic,def=76067358" json:"xpub_magic,omitempty"`
	XprvMagic           *uint32 `protobuf:"varint,10,opt,name=xprv_magic,json=xprvMagic,def=76066276" json:"xprv_magic,omitempty"`
	Segwit              *bool   `protobuf:"varint,11,opt,name=segwit" json:"segwit,omitempty"`
	Forkid              *uint32 `protobuf:"varint,12,opt,name=forkid" json:"forkid,omitempty"`
	XXX_unrecognized    []byte  `json:"-"`
}

func (mPtr *CoinsType) Reset()                    { *m = CoinsType{} }
func (mPtr *CoinsType) String() string            { return proto.CompactTextString(m) }
func (*CoinsType) ProtoMessage()               {}
func (*CoinsType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

const Default_CoinsType_AddressType uint32 = 0
const Default_CoinsType_AddressTypeP2Sh uint32 = 5
const Default_CoinsType_XpubMagic uint32 = 76067358
const Default_CoinsType_XprvMagic uint32 = 76066276

func (mPtr *CoinsType) GetCoinName() string {
	if m != nil && mPtr.CoinName != nil {
		return *mPtr.CoinName
	}
	return ""
}

func (mPtr *CoinsType) GetCoinShortcut() string {
	if m != nil && mPtr.CoinShortcut != nil {
		return *mPtr.CoinShortcut
	}
	return ""
}

func (mPtr *CoinsType) GetAddressType() uint32 {
	if m != nil && mPtr.AddressType != nil {
		return *mPtr.AddressType
	}
	return Default_CoinsType_AddressType
}

func (mPtr *CoinsType) GetMaxfeeKb() uint64 {
	if m != nil && mPtr.MaxfeeKb != nil {
		return *mPtr.MaxfeeKb
	}
	return 0
}

func (mPtr *CoinsType) GetAddressTypeP2Sh() uint32 {
	if m != nil && mPtr.AddressTypeP2Sh != nil {
		return *mPtr.AddressTypeP2Sh
	}
	return Default_CoinsType_AddressTypeP2Sh
}

func (mPtr *CoinsType) GetSignedMessageHeader() string {
	if m != nil && mPtr.SignedMessageHeader != nil {
		return *mPtr.SignedMessageHeader
	}
	return ""
}

func (mPtr *CoinsType) GetXpubMagic() uint32 {
	if m != nil && mPtr.XpubMagic != nil {
		return *mPtr.XpubMagic
	}
	return Default_CoinsType_XpubMagic
}

func (mPtr *CoinsType) GetXprvMagic() uint32 {
	if m != nil && mPtr.XprvMagic != nil {
		return *mPtr.XprvMagic
	}
	return Default_CoinsType_XprvMagic
}

func (mPtr *CoinsType) GetSegwit() bool {
	if m != nil && mPtr.Segwit != nil {
		return *mPtr.Segwit
	}
	return false
}

func (mPtr *CoinsType) GetForkid() uint32 {
	if m != nil && mPtr.Forkid != nil {
		return *mPtr.Forkid
	}
	return 0
}

// *
// Type of redeem script used in input
// @used_in TxInputType
type MultisigRedeemScriptTypes struct {
	Pubkeys          []*HDNodePathType `protobuf:"bytes,1,rep,name=pubkeys" json:"pubkeys,omitempty"`
	Signatures       [][]byte          `protobuf:"bytes,2,rep,name=signatures" json:"signatures,omitempty"`
	M                *uint32           `protobuf:"varint,3,opt,name=m" json:"m,omitempty"`
	XXX_unrecognized []byte            `json:"-"`
}

func (mPtr *MultisigRedeemScriptTypes) Reset()                    { *m = MultisigRedeemScriptTypes{} }
func (mPtr *MultisigRedeemScriptTypes) String() string            { return proto.CompactTextString(m) }
func (*MultisigRedeemScriptTypes) ProtoMessage()               {}
func (*MultisigRedeemScriptTypes) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (mPtr *MultisigRedeemScriptTypes) GetPubkeys() []*HDNodePathType {
	if m != nil {
		return mPtr.Pubkeys
	}
	return nil
}

func (mPtr *MultisigRedeemScriptTypes) GetSignatures() [][]byte {
	if m != nil {
		return mPtr.Signatures
	}
	return nil
}

func (mPtr *MultisigRedeemScriptTypes) GetM() uint32 {
	if m != nil && mPtr.M != nil {
		return *mPtr.M
	}
	return 0
}

// *
// Structure representing transaction input
// @used_in SimpleSignTx
// @used_in TransactionType
type TxInputType struct {
	AddressN         []uint32                  `protobuf:"varint,1,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	PrevHash         []byte                    `protobuf:"bytes,2,req,name=prev_hash,json=prevHash" json:"prev_hash,omitempty"`
	PrevIndex        *uint32                   `protobuf:"varint,3,req,name=prev_index,json=prevIndex" json:"prev_index,omitempty"`
	ScriptSig        []byte                    `protobuf:"bytes,4,opt,name=script_sig,json=scriptSig" json:"script_sig,omitempty"`
	Sequence         *uint32                   `protobuf:"varint,5,opt,name=sequence,def=4294967295" json:"sequence,omitempty"`
	ScriptType       *InputScriptType          `protobuf:"varint,6,opt,name=script_type,json=scriptType,enum=InputScriptType,def=0" json:"script_type,omitempty"`
	Multisig         *MultisigRedeemScriptTypes `protobuf:"bytes,7,opt,name=multisig" json:"multisig,omitempty"`
	Amount           *uint64                   `protobuf:"varint,8,opt,name=amount" json:"amount,omitempty"`
	XXX_unrecognized []byte                    `json:"-"`
}

func (mPtr *TxInputType) Reset()                    { *m = TxInputType{} }
func (mPtr *TxInputType) String() string            { return proto.CompactTextString(m) }
func (*TxInputType) ProtoMessage()               {}
func (*TxInputType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

const Default_TxInputType_Sequence uint32 = 4294967295
const Default_TxInputType_ScriptType InputScriptType = InputScriptType_SPENDADDRESS

func (mPtr *TxInputType) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

func (mPtr *TxInputType) GetPrevHash() []byte {
	if m != nil {
		return mPtr.PrevHash
	}
	return nil
}

func (mPtr *TxInputType) GetPrevIndex() uint32 {
	if m != nil && mPtr.PrevIndex != nil {
		return *mPtr.PrevIndex
	}
	return 0
}

func (mPtr *TxInputType) GetScriptSig() []byte {
	if m != nil {
		return mPtr.ScriptSig
	}
	return nil
}

func (mPtr *TxInputType) GetSequence() uint32 {
	if m != nil && mPtr.Sequence != nil {
		return *mPtr.Sequence
	}
	return Default_TxInputType_Sequence
}

func (mPtr *TxInputType) GetScriptType() InputScriptType {
	if m != nil && mPtr.ScriptType != nil {
		return *mPtr.ScriptType
	}
	return Default_TxInputType_ScriptType
}

func (mPtr *TxInputType) GetMultisig() *MultisigRedeemScriptTypes {
	if m != nil {
		return mPtr.Multisig
	}
	return nil
}

func (mPtr *TxInputType) GetAmount() uint64 {
	if m != nil && mPtr.Amount != nil {
		return *mPtr.Amount
	}
	return 0
}

// *
// Structure representing transaction output
// @used_in SimpleSignTx
// @used_in TransactionType
type TxOutputType struct {
	Address          *string                   `protobuf:"bytes,1,opt,name=address" json:"address,omitempty"`
	AddressN         []uint32                  `protobuf:"varint,2,rep,name=address_n,json=addressN" json:"address_n,omitempty"`
	Amount           *uint64                   `protobuf:"varint,3,req,name=amount" json:"amount,omitempty"`
	ScriptType       *OutputScriptType         `protobuf:"varint,4,req,name=script_type,json=scriptType,enum=OutputScriptType" json:"script_type,omitempty"`
	Multisig         *MultisigRedeemScriptTypes `protobuf:"bytes,5,opt,name=multisig" json:"multisig,omitempty"`
	OpReturnData     []byte                    `protobuf:"bytes,6,opt,name=op_return_data,json=opReturnData" json:"op_return_data,omitempty"`
	XXX_unrecognized []byte                    `json:"-"`
}

func (mPtr *TxOutputType) Reset()                    { *m = TxOutputType{} }
func (mPtr *TxOutputType) String() string            { return proto.CompactTextString(m) }
func (*TxOutputType) ProtoMessage()               {}
func (*TxOutputType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (mPtr *TxOutputType) GetAddress() string {
	if m != nil && mPtr.Address != nil {
		return *mPtr.Address
	}
	return ""
}

func (mPtr *TxOutputType) GetAddressN() []uint32 {
	if m != nil {
		return mPtr.AddressN
	}
	return nil
}

func (mPtr *TxOutputType) GetAmount() uint64 {
	if m != nil && mPtr.Amount != nil {
		return *mPtr.Amount
	}
	return 0
}

func (mPtr *TxOutputType) GetScriptType() OutputScriptType {
	if m != nil && mPtr.ScriptType != nil {
		return *mPtr.ScriptType
	}
	return OutputScriptType_PAYTOADDRESS
}

func (mPtr *TxOutputType) GetMultisig() *MultisigRedeemScriptTypes {
	if m != nil {
		return mPtr.Multisig
	}
	return nil
}

func (mPtr *TxOutputType) GetOpReturnData() []byte {
	if m != nil {
		return mPtr.OpReturnData
	}
	return nil
}

// *
// Structure representing compiled transaction output
// @used_in TransactionType
type TxOutputBinType struct {
	Amount           *uint64 `protobuf:"varint,1,req,name=amount" json:"amount,omitempty"`
	ScriptPubkey     []byte  `protobuf:"bytes,2,req,name=script_pubkey,json=scriptPubkey" json:"script_pubkey,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *TxOutputBinType) Reset()                    { *m = TxOutputBinType{} }
func (mPtr *TxOutputBinType) String() string            { return proto.CompactTextString(m) }
func (*TxOutputBinType) ProtoMessage()               {}
func (*TxOutputBinType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (mPtr *TxOutputBinType) GetAmount() uint64 {
	if m != nil && mPtr.Amount != nil {
		return *mPtr.Amount
	}
	return 0
}

func (mPtr *TxOutputBinType) GetScriptPubkey() []byte {
	if m != nil {
		return mPtr.ScriptPubkey
	}
	return nil
}

// *
// Structure representing transaction
// @used_in SimpleSignTx
type TransactionType struct {
	Version          *uint32            `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	Inputs           []*TxInputType     `protobuf:"bytes,2,rep,name=inputs" json:"inputs,omitempty"`
	BinOutputs       []*TxOutputBinType `protobuf:"bytes,3,rep,name=bin_outputs,json=binOutputs" json:"bin_outputs,omitempty"`
	Outputs          []*TxOutputType    `protobuf:"bytes,5,rep,name=outputs" json:"outputs,omitempty"`
	LockTime         *uint32            `protobuf:"varint,4,opt,name=lock_time,json=lockTime" json:"lock_time,omitempty"`
	InputsCnt        *uint32            `protobuf:"varint,6,opt,name=inputs_cnt,json=inputsCnt" json:"inputs_cnt,omitempty"`
	OutputsCnt       *uint32            `protobuf:"varint,7,opt,name=outputs_cnt,json=outputsCnt" json:"outputs_cnt,omitempty"`
	ExtraData        []byte             `protobuf:"bytes,8,opt,name=extra_data,json=extraData" json:"extra_data,omitempty"`
	ExtraDataLen     *uint32            `protobuf:"varint,9,opt,name=extra_data_len,json=extraDataLen" json:"extra_data_len,omitempty"`
	XXX_unrecognized []byte             `json:"-"`
}

func (mPtr *TransactionType) Reset()                    { *m = TransactionType{} }
func (mPtr *TransactionType) String() string            { return proto.CompactTextString(m) }
func (*TransactionType) ProtoMessage()               {}
func (*TransactionType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (mPtr *TransactionType) GetVersion() uint32 {
	if m != nil && mPtr.Version != nil {
		return *mPtr.Version
	}
	return 0
}

func (mPtr *TransactionType) GetInputs() []*TxInputType {
	if m != nil {
		return mPtr.Inputs
	}
	return nil
}

func (mPtr *TransactionType) GetBinOutputs() []*TxOutputBinType {
	if m != nil {
		return mPtr.BinOutputs
	}
	return nil
}

func (mPtr *TransactionType) GetOutputs() []*TxOutputType {
	if m != nil {
		return mPtr.Outputs
	}
	return nil
}

func (mPtr *TransactionType) GetLockTime() uint32 {
	if m != nil && mPtr.LockTime != nil {
		return *mPtr.LockTime
	}
	return 0
}

func (mPtr *TransactionType) GetInputsCnt() uint32 {
	if m != nil && mPtr.InputsCnt != nil {
		return *mPtr.InputsCnt
	}
	return 0
}

func (mPtr *TransactionType) GetOutputsCnt() uint32 {
	if m != nil && mPtr.OutputsCnt != nil {
		return *mPtr.OutputsCnt
	}
	return 0
}

func (mPtr *TransactionType) GetExtraData() []byte {
	if m != nil {
		return mPtr.ExtraData
	}
	return nil
}

func (mPtr *TransactionType) GetExtraDataLen() uint32 {
	if m != nil && mPtr.ExtraDataLen != nil {
		return *mPtr.ExtraDataLen
	}
	return 0
}

// *
// Structure representing request details
// @used_in TxRequest
type TxRequestDetailsType struct {
	RequestIndex     *uint32 `protobuf:"varint,1,opt,name=request_index,json=requestIndex" json:"request_index,omitempty"`
	TxHash           []byte  `protobuf:"bytes,2,opt,name=tx_hash,json=txHash" json:"tx_hash,omitempty"`
	ExtraDataLen     *uint32 `protobuf:"varint,3,opt,name=extra_data_len,json=extraDataLen" json:"extra_data_len,omitempty"`
	ExtraDataOffset  *uint32 `protobuf:"varint,4,opt,name=extra_data_offset,json=extraDataOffset" json:"extra_data_offset,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *TxRequestDetailsType) Reset()                    { *m = TxRequestDetailsType{} }
func (mPtr *TxRequestDetailsType) String() string            { return proto.CompactTextString(m) }
func (*TxRequestDetailsType) ProtoMessage()               {}
func (*TxRequestDetailsType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

func (mPtr *TxRequestDetailsType) GetRequestIndex() uint32 {
	if m != nil && mPtr.RequestIndex != nil {
		return *mPtr.RequestIndex
	}
	return 0
}

func (mPtr *TxRequestDetailsType) GetTxHash() []byte {
	if m != nil {
		return mPtr.TxHash
	}
	return nil
}

func (mPtr *TxRequestDetailsType) GetExtraDataLen() uint32 {
	if m != nil && mPtr.ExtraDataLen != nil {
		return *mPtr.ExtraDataLen
	}
	return 0
}

func (mPtr *TxRequestDetailsType) GetExtraDataOffset() uint32 {
	if m != nil && mPtr.ExtraDataOffset != nil {
		return *mPtr.ExtraDataOffset
	}
	return 0
}

// *
// Structure representing serialized data
// @used_in TxRequest
type TxRequestSerializedType struct {
	SignatureIndex   *uint32 `protobuf:"varint,1,opt,name=signature_index,json=signatureIndex" json:"signature_index,omitempty"`
	Signature        []byte  `protobuf:"bytes,2,opt,name=signature" json:"signature,omitempty"`
	SerializedTx     []byte  `protobuf:"bytes,3,opt,name=serialized_tx,json=serializedTx" json:"serialized_tx,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *TxRequestSerializedType) Reset()                    { *m = TxRequestSerializedType{} }
func (mPtr *TxRequestSerializedType) String() string            { return proto.CompactTextString(m) }
func (*TxRequestSerializedType) ProtoMessage()               {}
func (*TxRequestSerializedType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

func (mPtr *TxRequestSerializedType) GetSignatureIndex() uint32 {
	if m != nil && mPtr.SignatureIndex != nil {
		return *mPtr.SignatureIndex
	}
	return 0
}

func (mPtr *TxRequestSerializedType) GetSignature() []byte {
	if m != nil {
		return mPtr.Signature
	}
	return nil
}

func (mPtr *TxRequestSerializedType) GetSerializedTx() []byte {
	if m != nil {
		return mPtr.SerializedTx
	}
	return nil
}

// *
// Structure representing identity data
// @used_in IdentityType
type IdentityType struct {
	Proto            *string `protobuf:"bytes,1,opt,name=proto" json:"proto,omitempty"`
	User             *string `protobuf:"bytes,2,opt,name=user" json:"user,omitempty"`
	Host             *string `protobuf:"bytes,3,opt,name=host" json:"host,omitempty"`
	Port             *string `protobuf:"bytes,4,opt,name=port" json:"port,omitempty"`
	Path             *string `protobuf:"bytes,5,opt,name=path" json:"path,omitempty"`
	Index            *uint32 `protobuf:"varint,6,opt,name=index,def=0" json:"index,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (mPtr *IdentityType) Reset()                    { *m = IdentityType{} }
func (mPtr *IdentityType) String() string            { return proto.CompactTextString(m) }
func (*IdentityType) ProtoMessage()               {}
func (*IdentityType) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{10} }

const Default_IdentityType_Index uint32 = 0

func (mPtr *IdentityType) GetProto() string {
	if m != nil && mPtr.Proto != nil {
		return *mPtr.Proto
	}
	return ""
}

func (mPtr *IdentityType) GetUser() string {
	if m != nil && mPtr.User != nil {
		return *mPtr.User
	}
	return ""
}

func (mPtr *IdentityType) GetHost() string {
	if m != nil && mPtr.Host != nil {
		return *mPtr.Host
	}
	return ""
}

func (mPtr *IdentityType) GetPort() string {
	if m != nil && mPtr.Port != nil {
		return *mPtr.Port
	}
	return ""
}

func (mPtr *IdentityType) GetPath() string {
	if m != nil && mPtr.Path != nil {
		return *mPtr.Path
	}
	return ""
}

func (mPtr *IdentityType) GetIndex() uint32 {
	if m != nil && mPtr.Index != nil {
		return *mPtr.Index
	}
	return Default_IdentityType_Index
}

var E_WireIn = &proto.ExtensionDesc{
	ExtendedType:  (*google_protobuf.EnumValueOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         50002,
	Name:          "wire_in",
	Tag:           "varint,50002,opt,name=wire_in,json=wireIn",
	Filename:      "types.proto",
}

var E_WireOut = &proto.ExtensionDesc{
	ExtendedType:  (*google_protobuf.EnumValueOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         50003,
	Name:          "wire_out",
	Tag:           "varint,50003,opt,name=wire_out,json=wireOut",
	Filename:      "types.proto",
}

var E_WireDebugIn = &proto.ExtensionDesc{
	ExtendedType:  (*google_protobuf.EnumValueOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         50004,
	Name:          "wire_debug_in",
	Tag:           "varint,50004,opt,name=wire_debug_in,json=wireDebugIn",
	Filename:      "types.proto",
}

var E_WireDebugOut = &proto.ExtensionDesc{
	ExtendedType:  (*google_protobuf.EnumValueOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         50005,
	Name:          "wire_debug_out",
	Tag:           "varint,50005,opt,name=wire_debug_out,json=wireDebugOut",
	Filename:      "types.proto",
}

var E_WireTiny = &proto.ExtensionDesc{
	ExtendedType:  (*google_protobuf.EnumValueOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         50006,
	Name:          "wire_tiny",
	Tag:           "varint,50006,opt,name=wire_tiny,json=wireTiny",
	Filename:      "types.proto",
}

var E_WireBootloader = &proto.ExtensionDesc{
	ExtendedType:  (*google_protobuf.EnumValueOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         50007,
	Name:          "wire_bootloader",
	Tag:           "varint,50007,opt,name=wire_bootloader,json=wireBootloader",
	Filename:      "types.proto",
}

func init() {
	proto.RegisterType((*HDNodeType)(nil), "HDNodeType")
	proto.RegisterType((*HDNodePathType)(nil), "HDNodePathType")
	proto.RegisterType((*CoinsType)(nil), "CoinsType")
	proto.RegisterType((*MultisigRedeemScriptTypes)(nil), "MultisigRedeemScriptTypes")
	proto.RegisterType((*TxInputType)(nil), "TxInputType")
	proto.RegisterType((*TxOutputType)(nil), "TxOutputType")
	proto.RegisterType((*TxOutputBinType)(nil), "TxOutputBinType")
	proto.RegisterType((*TransactionType)(nil), "TransactionType")
	proto.RegisterType((*TxRequestDetailsType)(nil), "TxRequestDetailsType")
	proto.RegisterType((*TxRequestSerializedType)(nil), "TxRequestSerializedType")
	proto.RegisterType((*IdentityType)(nil), "IdentityType")
	proto.RegisterEnum("FailuresType", FailuresType_name, FailuresType_value)
	proto.RegisterEnum("OutputScriptType", OutputScriptType_name, OutputScriptType_value)
	proto.RegisterEnum("InputScriptType", InputScriptType_name, InputScriptType_value)
	proto.RegisterEnum("RequestType", RequestType_name, RequestType_value)
	proto.RegisterEnum("ButtonRequestType", ButtonRequestType_name, ButtonRequestType_value)
	proto.RegisterEnum("PinMatrixRequestType", PinMatrixRequestType_name, PinMatrixRequestType_value)
	proto.RegisterEnum("RecoveryDeviceType", RecoveryDeviceType_name, RecoveryDeviceType_value)
	proto.RegisterEnum("WordRequestType", WordRequestType_name, WordRequestType_value)
	proto.RegisterExtension(E_WireIn)
	proto.RegisterExtension(E_WireOut)
	proto.RegisterExtension(E_WireDebugIn)
	proto.RegisterExtension(E_WireDebugOut)
	proto.RegisterExtension(E_WireTiny)
	proto.RegisterExtension(E_WireBootloader)
}

func init() { proto.RegisterFile("types.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 1899 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x57, 0xdb, 0x72, 0x1a, 0xc9,
	0x19, 0xf6, 0x00, 0x92, 0xe0, 0x07, 0xc4, 0xa8, 0x7d, 0xd0, 0x78, 0x6d, 0xaf, 0x31, 0x76, 0x62,
	0x45, 0x55, 0x61, 0x77, 0xc9, 0x5a, 0x8e, 0x55, 0xa9, 0x24, 0x3a, 0xa0, 0x15, 0x65, 0x0b, 0x51,
	0xc3, 0x28, 0x56, 0x72, 0x33, 0x35, 0xcc, 0xb4, 0xa0, 0x4b, 0x43, 0x37, 0xe9, 0xe9, 0x91, 0xd1,
	0xde, 0xe4, 0x2a, 0xc9, 0x55, 0x5e, 0x23, 0x6f, 0x91, 0xaa, 0xbc, 0x41, 0xaa, 0x36, 0xa7, 0xcb,
	0xbc, 0x41, 0xae, 0xf2, 0x00, 0xa9, 0x3e, 0x0c, 0x02, 0xc9, 0xde, 0xd2, 0x1d, 0xfd, 0x7d, 0xff,
	0xf9, 0xd0, 0x3d, 0x40, 0x59, 0x5c, 0x4e, 0x70, 0xd2, 0x9c, 0x70, 0x26, 0xd8, 0x67, 0xf5, 0x21,
	0x63, 0xc3, 0x18, 0x7f, 0xa1, 0x4e, 0x83, 0xf4, 0xec, 0x8b, 0x08, 0x27, 0x21, 0x27, 0x13, 0xc1,
	0xb8, 0x96, 0x68, 0xfc, 0xd5, 0x02, 0x38, 0xdc, 0xef, 0xb2, 0x08, 0x7b, 0x97, 0x13, 0x8c, 0xee,
	0xc1, 0x52, 0x84, 0x27, 0x62, 0xe4, 0x58, 0xf5, 0xdc, 0x46, 0xd5, 0xd5, 0x07, 0x54, 0x87, 0xf2,
	0x19, 0xa1, 0x43, 0xcc, 0x27, 0x9c, 0x50, 0xe1, 0xe4, 0x14, 0x37, 0x0f, 0xa1, 0x47, 0x50, 0x0a,
	0x47, 0x24, 0x8e, 0x7c, 0x9a, 0x8e, 0x9d, 0xbc, 0xe2, 0x8b, 0x0a, 0xe8, 0xa6, 0x63, 0xf4, 0x04,
	0x20, 0x1c, 0x05, 0x84, 0xfa, 0x21, 0x8b, 0xb0, 0x53, 0xa8, 0xe7, 0x36, 0x2a, 0x6e, 0x49, 0x21,
	0x7b, 0x2c, 0xc2, 0xe8, 0x29, 0x94, 0x27, 0x9c, 0x5c, 0x04, 0x02, 0xfb, 0xe7, 0xf8, 0xd2, 0x59,
	0xaa, 0x5b, 0x1b, 0x15, 0x17, 0x0c, 0xf4, 0x16, 0x5f, 0x4a, 0xfd, 0x49, 0x3a, 0x88, 0x49, 0xa8,
	0xf8, 0x65, 0xc5, 0x97, 0x34, 0xf2, 0x16, 0x5f, 0x36, 0xba, 0xb0, 0xaa, 0x33, 0xe8, 0x05, 0x62,
	0xa4, 0xb2, 0x78, 0x0a, 0x05, 0x2a, 0x5d, 0xc9, 0x24, 0xca, 0xad, 0x72, 0xf3, 0x2a, 0x41, 0x57,
	0x11, 0x32, 0xdc, 0x20, 0x8a, 0x38, 0x4e, 0x12, 0x9f, 0x3a, 0xb9, 0x7a, 0x5e, 0x86, 0x6b, 0x80,
	0x6e, 0xe3, 0x7f, 0x39, 0x28, 0xee, 0x31, 0x42, 0x95, 0x29, 0x99, 0x18, 0x23, 0xd4, 0xa7, 0xc1,
	0x58, 0xda, 0xb3, 0x36, 0x4a, 0x6e, 0x51, 0x02, 0xdd, 0x60, 0x8c, 0xd1, 0x73, 0xa8, 0x2a, 0x32,
	0x19, 0x31, 0x2e, 0xc2, 0x54, 0x56, 0x46, 0x0a, 0x54, 0x24, 0xd8, 0x37, 0x18, 0x7a, 0x01, 0x95,
	0xcc, 0x97, 0x6c, 0x8d, 0x93, 0xaf, 0x5b, 0x1b, 0xd5, 0x6d, 0xeb, 0x4b, 0xb7, 0x6c, 0xe0, 0xcc,
	0xcf, 0x38, 0x98, 0x9e, 0x61, 0xec, 0x9f, 0x0f, 0x9c, 0x42, 0xdd, 0xda, 0x28, 0xb8, 0x45, 0x0d,
	0xbc, 0x1d, 0xa0, 0x1f, 0xc3, 0xda, 0xbc, 0x09, 0x7f, 0xd2, 0x4a, 0x46, 0xaa, 0x4e, 0xd5, 0x6d,
	0xeb, 0x95, 0x5b, 0x9b, 0xb3, 0xd3, 0x6b, 0x25, 0x23, 0xd4, 0x82, 0xfb, 0x09, 0x19, 0x52, 0x1c,
	0xf9, 0x63, 0x9c, 0x24, 0xc1, 0x10, 0xfb, 0x23, 0x1c, 0x44, 0x98, 0x3b, 0x45, 0x15, 0xde, 0x5d,
	0x4d, 0x1e, 0x69, 0xee, 0x50, 0x51, 0xe8, 0x25, 0xc0, 0x74, 0x92, 0x0e, 0xfc, 0x71, 0x30, 0x24,
	0xa1, 0x53, 0x52, 0xb6, 0x8b, 0xaf, 0xb7, 0xbe, 0xdc, 0x7a, 0xfd, 0x93, 0x57, 0x3f, 0x75, 0x4b,
	0x92, 0x3b, 0x92, 0x94, 0x16, 0xe4, 0x17, 0x46, 0x10, 0xae, 0x04, 0xb7, 0x5a, 0xaf, 0xb7, 0xa4,
	0x20, 0xbf, 0xd0, 0x82, 0x0f, 0x60, 0x39, 0xc1, 0xc3, 0x0f, 0x44, 0x38, 0xe5, 0xba, 0xb5, 0x51,
	0x74, 0xcd, 0x49, 0xe2, 0x67, 0x8c, 0x9f, 0x93, 0xc8, 0xa9, 0x48, 0x65, 0xd7, 0x9c, 0x1a, 0x09,
	0x38, 0x47, 0x69, 0x2c, 0x48, 0x42, 0x86, 0x2e, 0x8e, 0x30, 0x1e, 0xf7, 0xd5, 0xa4, 0xaa, 0xea,
	0xfc, 0x08, 0x56, 0x26, 0xe9, 0xe0, 0x1c, 0x5f, 0x26, 0x8e, 0x55, 0xcf, 0x6f, 0x94, 0x5b, 0xb5,
	0xe6, 0x62, 0xcb, 0xdd, 0x8c, 0x47, 0x9f, 0x03, 0xc8, 0xfc, 0x02, 0x91, 0x72, 0x9c, 0xa8, 0xde,
	0x56, 0xdc, 0x39, 0x04, 0x55, 0xc0, 0x1a, 0xeb, 0x1e, 0xb8, 0xd6, 0xb8, 0xf1, 0x97, 0x1c, 0x94,
	0xbd, 0x69, 0x87, 0x4e, 0x52, 0x91, 0xb5, 0xe1, 0x6a, 0x30, 0xac, 0xc5, 0xc1, 0x90, 0xe4, 0x84,
	0xe3, 0x0b, 0x7f, 0x14, 0x24, 0x23, 0xb5, 0x04, 0x15, 0xb7, 0x28, 0x81, 0xc3, 0x20, 0x19, 0xa9,
	0x21, 0x95, 0x24, 0xa1, 0x11, 0x9e, 0x9a, 0x15, 0x50, 0xe2, 0x1d, 0x09, 0x48, 0x5a, 0x6f, 0x9e,
	0x9f, 0x90, 0xa1, 0x6a, 0x70, 0xc5, 0x2d, 0x69, 0xa4, 0x4f, 0x86, 0xe8, 0x87, 0x50, 0x4c, 0xf0,
	0x6f, 0x53, 0x4c, 0x43, 0x6c, 0x1a, 0x0b, 0x5f, 0xb7, 0xde, 0x7c, 0xfd, 0x66, 0xeb, 0x75, 0xeb,
	0xcd, 0x2b, 0x77, 0xc6, 0xa1, 0x5f, 0x40, 0xd9, 0x98, 0x51, 0xb3, 0x24, 0x77, 0x61, 0xb5, 0x65,
	0x37, 0x55, 0x02, 0x57, 0xf5, 0xda, 0xae, 0xf4, 0x7b, 0xed, 0xee, 0xfe, 0xce, 0xfe, 0xbe, 0xdb,
	0xee, 0xf7, 0x5d, 0xe3, 0x59, 0x25, 0xf8, 0x0a, 0x8a, 0x63, 0x53, 0x65, 0x67, 0xa5, 0x6e, 0x6d,
	0x94, 0x5b, 0x0f, 0x9b, 0x9f, 0x2a, 0xbb, 0x3b, 0x13, 0x95, 0x4d, 0x0b, 0xc6, 0x2c, 0xa5, 0x42,
	0xcd, 0x50, 0xc1, 0x35, 0xa7, 0xc6, 0x7f, 0x2d, 0xa8, 0x78, 0xd3, 0xe3, 0x54, 0x64, 0x05, 0x74,
	0x60, 0xc5, 0xd4, 0xcb, 0x6c, 0x4b, 0x76, 0xfc, 0xde, 0x9d, 0x9b, 0xb3, 0x2f, 0x2b, 0x37, 0xb3,
	0x8f, 0x5a, 0x8b, 0xf9, 0xca, 0xbb, 0x63, 0xb5, 0xb5, 0xd6, 0xd4, 0x0e, 0xe7, 0x22, 0xfd, 0x54,
	0x8a, 0x4b, 0xb7, 0x4f, 0xf1, 0x05, 0xac, 0xb2, 0x89, 0xcf, 0xb1, 0x48, 0x39, 0xf5, 0xa3, 0x40,
	0x04, 0xe6, 0xa6, 0xa9, 0xb0, 0x89, 0xab, 0xc0, 0xfd, 0x40, 0x04, 0x8d, 0x2e, 0xd4, 0xb2, 0x7c,
	0x77, 0xcd, 0x15, 0x71, 0x15, 0xbb, 0xb5, 0x10, 0xfb, 0x73, 0xa8, 0x9a, 0xd8, 0xf5, 0x6c, 0x9a,
	0x91, 0xa9, 0x68, 0xb0, 0xa7, 0xb0, 0xc6, 0xdf, 0x72, 0x50, 0xf3, 0x78, 0x40, 0x93, 0x20, 0x14,
	0x84, 0xd1, 0xac, 0x86, 0x17, 0x98, 0x27, 0x84, 0x51, 0x55, 0xc3, 0xaa, 0x9b, 0x1d, 0xd1, 0x0b,
	0x58, 0x26, 0xb2, 0xd5, 0x7a, 0xb0, 0xcb, 0xad, 0x4a, 0x73, 0x6e, 0x78, 0x5d, 0xc3, 0xa1, 0xaf,
	0xa0, 0x3c, 0x20, 0xd4, 0x67, 0x2a, 0xca, 0xc4, 0xc9, 0x2b, 0x51, 0xbb, 0x79, 0x2d, 0x6e, 0x17,
	0x06, 0x84, 0x6a, 0x24, 0x41, 0x2f, 0x61, 0x25, 0x13, 0x5f, 0x52, 0xe2, 0xd5, 0xe6, 0x7c, 0x5b,
	0xdd, 0x8c, 0x95, 0x5d, 0x8c, 0x59, 0x78, 0xee, 0x0b, 0x32, 0xc6, 0x6a, 0x8c, 0xab, 0x6e, 0x51,
	0x02, 0x1e, 0x19, 0x63, 0x39, 0xe4, 0x3a, 0x04, 0x3f, 0xa4, 0x42, 0x95, 0xaf, 0xea, 0x96, 0x34,
	0xb2, 0x47, 0x85, 0xbc, 0xe8, 0x8d, 0x19, 0xc5, 0xaf, 0x28, 0x1e, 0x0c, 0x24, 0x05, 0x9e, 0x00,
	0xe0, 0xa9, 0xe0, 0x81, 0x2e, 0x7f, 0x51, 0x2f, 0x89, 0x42, 0x64, 0xed, 0x65, 0x87, 0xae, 0x68,
	0x3f, 0xc6, 0x54, 0xdf, 0x53, 0x6e, 0x65, 0x26, 0xf2, 0x0e, 0xd3, 0xc6, 0x9f, 0x2d, 0xb8, 0xe7,
	0x4d, 0x5d, 0xb9, 0x31, 0x89, 0xd8, 0xc7, 0x22, 0x20, 0xb1, 0xbe, 0x62, 0x9f, 0x43, 0x95, 0x6b,
	0xd4, 0x2c, 0xa9, 0x2e, 0x6e, 0xc5, 0x80, 0x7a, 0x4f, 0xd7, 0x61, 0x45, 0x4c, 0xb3, 0x0d, 0x97,
	0xfe, 0x97, 0xc5, 0x54, 0xed, 0xf7, 0x4d, 0xe7, 0xf9, 0x9b, 0xce, 0xd1, 0x26, 0xac, 0xcd, 0x49,
	0xb1, 0xb3, 0xb3, 0x04, 0x0b, 0x53, 0xa6, 0xda, 0x4c, 0xf0, 0x58, 0xc1, 0x8d, 0xdf, 0x5b, 0xb0,
	0x3e, 0x0b, 0xb4, 0x8f, 0x39, 0x09, 0x62, 0xf2, 0x2d, 0x8e, 0x54, 0xac, 0x2f, 0xa1, 0x36, 0xbb,
	0xb3, 0x16, 0xa2, 0x5d, 0x9d, 0xc1, 0x3a, 0xde, 0xc7, 0x50, 0x9a, 0x21, 0x26, 0xe2, 0x2b, 0x40,
	0x8d, 0xe0, 0xcc, 0xb0, 0x2f, 0xa6, 0x2a, 0x66, 0x39, 0x82, 0x57, 0xde, 0xa6, 0x8d, 0x3f, 0x59,
	0x50, 0xe9, 0x44, 0x98, 0x0a, 0x22, 0x2e, 0xb3, 0x8f, 0x00, 0xf5, 0x71, 0x60, 0x36, 0x58, 0x1f,
	0x10, 0x82, 0x42, 0x9a, 0x60, 0x6e, 0xde, 0x38, 0xf5, 0x5b, 0x62, 0x23, 0x96, 0x08, 0x65, 0xb6,
	0xe4, 0xaa, 0xdf, 0x12, 0x9b, 0x30, 0xae, 0xb3, 0x2e, 0xb9, 0xea, 0xb7, 0xc2, 0x02, 0xa1, 0xdf,
	0x2c, 0x89, 0x05, 0x62, 0x84, 0xd6, 0x61, 0x49, 0x27, 0xb6, 0x9c, 0x3d, 0x88, 0xfa, 0xbc, 0xf9,
	0x5d, 0x0e, 0xca, 0x07, 0x01, 0x89, 0x53, 0xae, 0xbf, 0x49, 0x9e, 0xc0, 0x43, 0x73, 0xf4, 0x4f,
	0x28, 0x9e, 0x4e, 0x70, 0x28, 0x66, 0xaf, 0x97, 0x6d, 0xa1, 0xcf, 0xe0, 0x41, 0x46, 0xef, 0xa6,
	0x42, 0x30, 0xda, 0x36, 0x22, 0x76, 0x0e, 0xdd, 0x87, 0xb5, 0x8c, 0x93, 0x85, 0x6f, 0x73, 0xce,
	0xb8, 0x9d, 0x47, 0x8f, 0x60, 0x3d, 0x83, 0x77, 0xd4, 0xda, 0xed, 0x05, 0x34, 0xc4, 0x71, 0x8c,
	0x23, 0xbb, 0x80, 0xd6, 0xe1, 0x6e, 0x46, 0xf6, 0xc8, 0x95, 0xb1, 0x25, 0xe4, 0xc0, 0xbd, 0x39,
	0xe2, 0x4a, 0x65, 0x19, 0x3d, 0x00, 0x34, 0xc7, 0x74, 0xe8, 0x45, 0x10, 0x93, 0xc8, 0x5e, 0x41,
	0x8f, 0xc1, 0xc9, 0x70, 0x03, 0xf6, 0xb3, 0xd6, 0xd8, 0xc5, 0x05, 0x7b, 0x9c, 0x85, 0x38, 0x49,
	0x74, 0x7c, 0xa5, 0xf9, 0x94, 0xba, 0x4c, 0xb4, 0x29, 0x4b, 0x87, 0xa3, 0x83, 0x94, 0x46, 0x89,
	0x0d, 0xd7, 0xb8, 0x0e, 0x25, 0xc2, 0x74, 0xd2, 0x2e, 0xa3, 0x87, 0x70, 0x3f, 0xe3, 0x0e, 0x08,
	0x1f, 0x7f, 0x08, 0x38, 0xd6, 0x26, 0xc3, 0xcd, 0x3f, 0x5a, 0x60, 0x5f, 0xbf, 0x35, 0x91, 0x0d,
	0x95, 0xde, 0xce, 0xaf, 0xbd, 0x63, 0xf3, 0x50, 0xd8, 0x77, 0xd0, 0x5d, 0xa8, 0x29, 0xa4, 0xbf,
	0xe7, 0x76, 0x7a, 0xde, 0xe1, 0x4e, 0xff, 0xd0, 0xb6, 0xd0, 0x1a, 0x54, 0x15, 0x78, 0x74, 0xf2,
	0xce, 0xeb, 0xf4, 0x3b, 0xdf, 0xd8, 0xb9, 0x19, 0x74, 0xdc, 0x73, 0xdb, 0xde, 0x89, 0xdb, 0xb5,
	0xf3, 0x33, 0x63, 0xef, 0x3b, 0x5e, 0x57, 0x1a, 0x2b, 0xa0, 0x7b, 0x60, 0x2b, 0xa4, 0xd7, 0xea,
	0x1f, 0x66, 0xe8, 0xd2, 0x66, 0x0c, 0xb5, 0x6b, 0xcf, 0x95, 0x54, 0x9d, 0x7f, 0xb0, 0xec, 0x3b,
	0xd2, 0xbe, 0x42, 0x66, 0x2e, 0x2d, 0x54, 0x81, 0x62, 0xfb, 0xd4, 0x6b, 0xbb, 0xdd, 0x9d, 0x77,
	0x76, 0x6e, 0xa6, 0x92, 0xd9, 0xcd, 0x4b, 0x6f, 0x0a, 0x99, 0xf7, 0x56, 0xd8, 0x3c, 0x81, 0xb2,
	0xd9, 0x30, 0xe5, 0xa9, 0x0c, 0x2b, 0xde, 0x69, 0xa7, 0xdb, 0x3b, 0xf1, 0xec, 0x3b, 0xd2, 0xa2,
	0x77, 0x7a, 0x7c, 0xe2, 0xc9, 0x93, 0x85, 0x00, 0x96, 0xbd, 0xd3, 0xa3, 0xb6, 0xb7, 0x63, 0xe7,
	0xd0, 0x2a, 0x80, 0x77, 0x7a, 0xd0, 0xe9, 0x76, 0xfa, 0x87, 0xed, 0x7d, 0x3b, 0x8f, 0x6a, 0x50,
	0xf6, 0x4e, 0xdb, 0xa7, 0x9e, 0xbb, 0xb3, 0xbf, 0xe3, 0xed, 0xd8, 0x85, 0xcd, 0xff, 0xe4, 0x60,
	0x4d, 0x4f, 0xdb, 0xbc, 0xf5, 0x75, 0xb8, 0xbb, 0x00, 0xfa, 0xc7, 0x62, 0x84, 0xb9, 0x6d, 0xa1,
	0x06, 0x7c, 0xbe, 0x48, 0x1c, 0x60, 0x7c, 0x7c, 0x81, 0xb9, 0x37, 0xe2, 0x38, 0x19, 0xb1, 0x58,
	0xce, 0xea, 0x53, 0x78, 0xb4, 0x28, 0xb3, 0xc7, 0xe8, 0x19, 0xe1, 0x63, 0xdd, 0x35, 0x3b, 0x2f,
	0xf7, 0x60, 0x51, 0xc0, 0xc5, 0x09, 0x16, 0xfb, 0xf8, 0x82, 0x84, 0xd8, 0x2e, 0xdc, 0xa4, 0x8d,
	0xfe, 0x7b, 0xc6, 0xe5, 0xf4, 0x3e, 0x06, 0x67, 0x91, 0x7e, 0x4f, 0x26, 0xd8, 0x28, 0x2f, 0xdf,
	0x54, 0xee, 0x71, 0x26, 0x70, 0x28, 0xf6, 0x82, 0x38, 0xb6, 0x57, 0xe4, 0xa8, 0x2e, 0xd2, 0x72,
	0x8e, 0xbd, 0xa9, 0x5d, 0xbc, 0x19, 0x75, 0x36, 0x78, 0x7b, 0x23, 0x1c, 0x9e, 0xdb, 0x25, 0x39,
	0x93, 0x8b, 0x02, 0x3b, 0xfa, 0xcd, 0xb7, 0x41, 0xae, 0xe1, 0x35, 0xa7, 0xd9, 0x37, 0xbd, 0x5d,
	0xde, 0xfc, 0x1d, 0xdc, 0xeb, 0x11, 0x7a, 0x14, 0x08, 0x4e, 0xa6, 0xf3, 0x35, 0xae, 0xc3, 0xe3,
	0x8f, 0xe1, 0xfe, 0x5e, 0xca, 0x39, 0xa6, 0xc2, 0xb6, 0xd0, 0x33, 0x78, 0xf2, 0x51, 0x89, 0x2e,
	0xfe, 0x70, 0x40, 0x78, 0x22, 0xec, 0x9c, 0xec, 0xc7, 0xa7, 0x44, 0xfa, 0x38, 0x64, 0x34, 0xb2,
	0xf3, 0x9b, 0xbf, 0x01, 0xe4, 0xe2, 0x90, 0x5d, 0x60, 0x7e, 0xa9, 0xcb, 0xa4, 0xdc, 0xff, 0x00,
	0x9e, 0xdd, 0x44, 0xfd, 0x7e, 0xc8, 0x83, 0xf1, 0x20, 0xc6, 0x91, 0x2c, 0x76, 0x62, 0xdf, 0x91,
	0xf5, 0xfc, 0x88, 0x98, 0x76, 0x68, 0x5b, 0x9b, 0x67, 0x50, 0x93, 0x92, 0xf3, 0x79, 0x3d, 0x84,
	0xfb, 0xd7, 0x20, 0xbf, 0x17, 0x07, 0x84, 0xda, 0x77, 0x64, 0x9d, 0xae, 0x53, 0xda, 0xd2, 0x1b,
	0xdb, 0xfa, 0x34, 0xb9, 0x65, 0xe7, 0xb6, 0x7f, 0x06, 0x2b, 0x1f, 0x88, 0x7a, 0x41, 0xd0, 0xb3,
	0xa6, 0xfe, 0x2f, 0xd8, 0xcc, 0xfe, 0x0b, 0x36, 0xdb, 0x34, 0x1d, 0xff, 0x2a, 0x88, 0x53, 0x7c,
	0x3c, 0x91, 0x77, 0x60, 0xe2, 0x7c, 0xf7, 0x87, 0xbc, 0xfe, 0x52, 0x97, 0x3a, 0x1d, 0xba, 0xfd,
	0x73, 0x28, 0x2a, 0x6d, 0x96, 0x8a, 0xdb, 0xa8, 0xff, 0xdd, 0xa8, 0x2b, 0x97, 0xc7, 0xa9, 0xd8,
	0xfe, 0x06, 0xaa, 0x4a, 0x3f, 0xc2, 0x83, 0x74, 0x78, 0xcb, 0x18, 0xfe, 0x61, 0x8c, 0x94, 0xa5,
	0xe6, 0xbe, 0x54, 0xec, 0xd0, 0xed, 0x0e, 0xac, 0xce, 0x19, 0xba, 0x65, 0x38, 0xff, 0x34, 0x96,
	0x2a, 0x33, 0x4b, 0x32, 0xa6, 0x5f, 0x42, 0x49, 0x99, 0x12, 0x84, 0x5e, 0xde, 0xc6, 0xca, 0xbf,
	0x8c, 0x15, 0x55, 0x09, 0x8f, 0xd0, 0xcb, 0xed, 0x77, 0x50, 0x53, 0x16, 0x06, 0x8c, 0x89, 0x98,
	0xa9, 0x3f, 0x4f, 0xb7, 0xb0, 0xf3, 0x6f, 0x63, 0x47, 0x25, 0xb2, 0x3b, 0x53, 0xdd, 0xfd, 0x0a,
	0x9e, 0x87, 0x6c, 0xdc, 0x4c, 0x02, 0xc1, 0x92, 0x11, 0x89, 0x83, 0x41, 0xd2, 0x14, 0x1c, 0x7f,
	0xcb, 0x78, 0x33, 0x26, 0x83, 0x99, 0xbd, 0x5d, 0xf0, 0x14, 0x28, 0xdb, 0xfb, 0xff, 0x00, 0x00,
	0x00, 0xff, 0xff, 0x70, 0x88, 0xcd, 0x71, 0xe2, 0x0f, 0x00, 0x00,
}
// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = mathPtr.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// *
// Type of failures returned by Failure message
// @used_in Failure
type FailuresType int32

const (
	FailuresType_Failures_UnexpectedMessage FailuresType = 1
	FailuresType_Failures_ButtonExpected    FailuresType = 2
	FailuresType_Failures_DataError         FailuresType = 3
	FailuresType_Failures_ActionCancelled   FailuresType = 4
	FailuresType_Failures_PinExpected       FailuresType = 5
	FailuresType_Failures_PinCancelled      FailuresType = 6
	FailuresType_Failures_PinInvalid        FailuresType = 7
	FailuresType_Failures_InvalidSignature  FailuresType = 8
	FailuresType_Failures_ProcessError      FailuresType = 9
	FailuresType_Failures_NotEnoughFunds    FailuresType = 10
	FailuresType_Failures_NotInitialized    FailuresType = 11
	FailuresType_Failures_FirmwareError     FailuresType = 99
)

var FailuresType_name = map[int32]string{
	1:  "Failures_UnexpectedMessage",
	2:  "Failures_ButtonExpected",
	3:  "Failures_DataError",
	4:  "Failures_ActionCancelled",
	5:  "Failures_PinExpected",
	6:  "Failures_PinCancelled",
	7:  "Failures_PinInvalid",
	8:  "Failures_InvalidSignature",
	9:  "Failures_ProcessError",
	10: "Failures_NotEnoughFunds",
	11: "Failures_NotInitialized",
	99: "Failures_FirmwareError",
}
var FailuresType_value = map[string]int32{
	"Failures_UnexpectedMessage": 1,
	"Failures_ButtonExpected":    2,
	"Failures_DataError":         3,
	"Failures_ActionCancelled":   4,
	"Failures_PinExpected":       5,
	"Failures_PinCancelled":      6,
	"Failures_PinInvalid":        7,
	"Failures_InvalidSignature":  8,
	"Failures_ProcessError":      9,
	"Failures_NotEnoughFunds":    10,
	"Failures_NotInitialized":    11,
	"Failures_FirmwareError":     99,
}

func (x FailuresType) Enum() *FailuresType {
	p := new(FailuresType)
	*p = x
	return p
}
func (x FailuresType) String() string {
	return proto.EnumName(FailuresType_name, int32(x))
}