package auth

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash"
	"unicode"

	"golang.org/x/crypto/pbkdf2"
)

const identityIterations = 100_000

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	subkey := pbkdf2.Key([]byte(password), salt, identityIterations, 32, sha512.New)
	payload := make([]byte, 13+len(salt)+len(subkey))
	payload[0] = 0x01
	binary.BigEndian.PutUint32(payload[1:5], 2) // KeyDerivationPrf.HMACSHA512
	binary.BigEndian.PutUint32(payload[5:9], identityIterations)
	binary.BigEndian.PutUint32(payload[9:13], uint32(len(salt)))
	copy(payload[13:], salt)
	copy(payload[13+len(salt):], subkey)
	return base64.StdEncoding.EncodeToString(payload), nil
}

func verifyPassword(encoded, password string) bool {
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(payload) < 2 {
		return false
	}
	if payload[0] == 0x00 {
		if len(payload) != 49 {
			return false
		}
		expected := pbkdf2.Key([]byte(password), payload[1:17], 1000, 32, sha1.New)
		return subtle.ConstantTimeCompare(expected, payload[17:]) == 1
	}
	if payload[0] != 0x01 || len(payload) < 14 {
		return false
	}
	prf := binary.BigEndian.Uint32(payload[1:5])
	iterations := binary.BigEndian.Uint32(payload[5:9])
	saltLength := binary.BigEndian.Uint32(payload[9:13])
	if iterations == 0 || iterations > 10_000_000 || saltLength < 16 || uint64(13+saltLength) >= uint64(len(payload)) {
		return false
	}
	var digest func() hash.Hash
	switch prf {
	case 0:
		digest = sha1.New
	case 1:
		digest = sha256.New
	case 2:
		digest = sha512.New
	default:
		return false
	}
	saltEnd := 13 + int(saltLength)
	expected := pbkdf2.Key([]byte(password), payload[13:saltEnd], int(iterations), len(payload)-saltEnd, digest)
	return subtle.ConstantTimeCompare(expected, payload[saltEnd:]) == 1
}

func validatePassword(password string) error {
	if len([]rune(password)) < 12 || len([]rune(password)) > 128 {
		return errors.New("password must contain between 12 and 128 characters")
	}
	var lower, upper, digit, symbol bool
	unique := make(map[rune]struct{})
	for _, character := range password {
		unique[character] = struct{}{}
		lower = lower || unicode.IsLower(character)
		upper = upper || unicode.IsUpper(character)
		digit = digit || unicode.IsDigit(character)
		symbol = symbol || (!unicode.IsLetter(character) && !unicode.IsDigit(character))
	}
	if !lower || !upper || !digit || !symbol || len(unique) < 6 {
		return errors.New("password must include upper, lower, number, symbol, and six unique characters")
	}
	return nil
}
