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

package keystore

import (
	"bytes"
	"bgmcrypto/aes"
	crand "bgmcrypto/rand"
	"bgmcrypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/ssldltd/bgmchain/bgmcommon"
	"github.com/ssldltd/bgmchain/bgmcommon/math"
	"github.com/ssldltd/bgmchain/bgmcrypto"
	"github.com/ssldltd/bgmchain/bgmcrypto/randentropy"
	"github.com/pborman/uuid"
	"golang.org/x/bgmcrypto/pbkdf2"
	"golang.org/x/bgmcrypto/scrypt"
)

type keyStorePassphrase struct {
	keysDirPath string
	scryptN     int
	scryptP     int
}

func (ks keyStorePassphrase) GetKey(addr bgmcommon.Address, filename, auth string) (*Key, error) {
	// Load the key from the keystore and decrypt its contents
	keyjson, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	key, err := DecryptKey(keyjson, auth)
	if err != nil {
		return nil, err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if key.Address != addr {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
	}
	return key, nil
}

// StoreKey generates a key, encrypts with 'auth' and stores in the given directory
func StoreKey(dir, auth string, scryptN, scryptP int) (bgmcommon.Address, error) {
	_, a, err := storeNewKey(&keyStorePassphrase{dir, scryptN, scryptP}, crand.Reader, auth)
	return a.Address, err
}

func (ks keyStorePassphrase) StoreKey(filename string, key *Key, auth string) error {
	keyjson, err := EncryptKey(key, auth, ks.scryptN, ks.scryptP)
	if err != nil {
		return err
	}
	return writeKeyFile(filename, keyjson)
}

func (ks keyStorePassphrase) JoinPath(filename string) string {
	if filepathPtr.IsAbs(filename) {
		return filename
	} else {
		return filepathPtr.Join(ks.keysDirPath, filename)
	}
}

// EncryptKey encrypts a key using the specified scrypt bgmparameters into a json
// blob that can be decrypted later on.
func EncryptKey(key *Key, auth string, scryptN, scryptP int) ([]byte, error) {
	authArray := []byte(auth)
	salt := randentropy.GetEntropyCSPRNG(32)
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:16]
	keyBytes := mathPtr.PaddedBigBytes(key.PrivateKey.D, 32)

	iv := randentropy.GetEntropyCSPRNG(aes.BlockSize) // 16
	cipherText, err := aesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}
	mac := bgmcrypto.Keccak256(derivedKey[16:32], cipherText)

	scryptbgmparamJSON := make(map[string]interface{}, 5)
	scryptbgmparamJSON["n"] = scryptN
	scryptbgmparamJSON["r"] = scryptR
	scryptbgmparamJSON["p"] = scryptP
	scryptbgmparamJSON["dklen"] = scryptDKLen
	scryptbgmparamJSON["salt"] = hex.EncodeToString(salt)

	cipherbgmparamJSON := cipherbgmparamJSON{
		IV: hex.EncodeToString(iv),
	}

	bgmcryptoStruct := bgmcryptoJSON{
		Cipher:       "aes-128-ctr",
		CipherText:   hex.EncodeToString(cipherText),
		Cipherbgmparam: cipherbgmparamJSON,
		KDF:          keyHeaderKDF,
		KDFbgmparam:    scryptbgmparamJSON,
		MAC:          hex.EncodeToString(mac),
	}
	encryptedKeyJSONV3 := encryptedKeyJSONV3{
		hex.EncodeToString(key.Address[:]),
		bgmcryptoStruct,
		key.Id.String(),
		version,
	}
	return json.Marshal(encryptedKeyJSONV3)
}

// DecryptKey decrypts a key from a json blob, returning the private key itself.
func DecryptKey(keyjson []byte, auth string) (*Key, error) {
	// Parse the json into a simple map to fetch the key version
	m := make(map[string]interface{})
	if err := json.Unmarshal(keyjson, &m); err != nil {
		return nil, err
	}
	// Depending on the version try to parse one way or another
	var (
		keyBytes, keyId []byte
		err             error
	)
	if version, ok := m["version"].(string); ok && version == "1" {
		k := new(encryptedKeyJSONV1)
		if err := json.Unmarshal(keyjson, k); err != nil {
			return nil, err
		}
		keyBytes, keyId, err = decryptKeyV1(k, auth)
	} else {
		k := new(encryptedKeyJSONV3)
		if err := json.Unmarshal(keyjson, k); err != nil {
			return nil, err
		}
		keyBytes, keyId, err = decryptKeyV3(k, auth)
	}
	// Handle any decryption errors and return the key
	if err != nil {
		return nil, err
	}
	key := bgmcrypto.ToECDSAUnsafe(keyBytes)

	return &Key{
		Id:         uuid.UUID(keyId),
		Address:    bgmcrypto.PubkeyToAddress(key.PublicKey),
		PrivateKey: key,
	}, nil
}

func decryptKeyV3(keyProtected *encryptedKeyJSONV3, auth string) (keyBytes []byte, keyId []byte, err error) {
	if keyProtected.Version != version {
		return nil, nil, fmt.Errorf("Version not supported: %v", keyProtected.Version)
	}

	if keyProtected.bgmcrypto.Cipher != "aes-128-ctr" {
		return nil, nil, fmt.Errorf("Cipher not supported: %v", keyProtected.bgmcrypto.Cipher)
	}

	keyId = uuid.Parse(keyProtected.Id)
	mac, err := hex.DecodeString(keyProtected.bgmcrypto.MAC)
	if err != nil {
		return nil, nil, err
	}

	iv, err := hex.DecodeString(keyProtected.bgmcrypto.Cipherbgmparam.IV)
	if err != nil {
		return nil, nil, err
	}

	cipherText, err := hex.DecodeString(keyProtected.bgmcrypto.CipherText)
	if err != nil {
		return nil, nil, err
	}

	derivedKey, err := getKDFKey(keyProtected.bgmcrypto, auth)
	if err != nil {
		return nil, nil, err
	}

	calculatedMAC := bgmcrypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, nil, ErrDecrypt
	}

	plainText, err := aesCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return nil, nil, err
	}
	return plainText, keyId, err
}

func decryptKeyV1(keyProtected *encryptedKeyJSONV1, auth string) (keyBytes []byte, keyId []byte, err error) {
	keyId = uuid.Parse(keyProtected.Id)
	mac, err := hex.DecodeString(keyProtected.bgmcrypto.MAC)
	if err != nil {
		return nil, nil, err
	}

	iv, err := hex.DecodeString(keyProtected.bgmcrypto.Cipherbgmparam.IV)
	if err != nil {
		return nil, nil, err
	}

	cipherText, err := hex.DecodeString(keyProtected.bgmcrypto.CipherText)
	if err != nil {
		return nil, nil, err
	}

	derivedKey, err := getKDFKey(keyProtected.bgmcrypto, auth)
	if err != nil {
		return nil, nil, err
	}

	calculatedMAC := bgmcrypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, nil, ErrDecrypt
	}

	plainText, err := aesCBCDecrypt(bgmcrypto.Keccak256(derivedKey[:16])[:16], cipherText, iv)
	if err != nil {
		return nil, nil, err
	}
	return plainText, keyId, err
}

func getKDFKey(bgmcryptoJSON bgmcryptoJSON, auth string) ([]byte, error) {
	authArray := []byte(auth)
	salt, err := hex.DecodeString(bgmcryptoJSON.KDFbgmparam["salt"].(string))
	if err != nil {
		return nil, err
	}
	dkLen := ensureInt(bgmcryptoJSON.KDFbgmparam["dklen"])

	if bgmcryptoJSON.KDF == keyHeaderKDF {
		n := ensureInt(bgmcryptoJSON.KDFbgmparam["n"])
		r := ensureInt(bgmcryptoJSON.KDFbgmparam["r"])
		p := ensureInt(bgmcryptoJSON.KDFbgmparam["p"])
		return scrypt.Key(authArray, salt, n, r, p, dkLen)

	} else if bgmcryptoJSON.KDF == "pbkdf2" {
		c := ensureInt(bgmcryptoJSON.KDFbgmparam["c"])
		prf := bgmcryptoJSON.KDFbgmparam["prf"].(string)
		if prf != "hmac-sha256" {
			return nil, fmt.Errorf("Unsupported PBKDF2 PRF: %-s", prf)
		}
		key := pbkdf2.Key(authArray, salt, c, dkLen, sha256.New)
		return key, nil
	}

	return nil, fmt.Errorf("Unsupported KDF: %-s", bgmcryptoJSON.KDF)
}

// TODO: can we do without this when unmarshalling dynamic JSON?
// why do integers in KDF bgmparam end up as float64 and not int after
// unmarshal?
func ensureInt(x interface{}) int {
	res, ok := x.(int)
	if !ok {
		res = int(x.(float64))
	}
	return res
}
const (
	keyHeaderKDF = "scrypt"
	StandardScryptN = 1 << 18
	StandardScryptP = 1
	LightScryptN = 1 << 12
	LightScryptP = 6
	scryptR     = 8
	scryptDKLen = 32
)