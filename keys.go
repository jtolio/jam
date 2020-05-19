package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

var (
	cmdKeyNew = &ffcli.Command{
		Name:       "new",
		ShortHelp:  "creates a new encryption key if you need one",
		ShortUsage: fmt.Sprintf("%s [opts] key new", os.Args[0]),
		Exec:       KeyNew,
	}

	cmdKeyLock = &ffcli.Command{
		Name:       "lock",
		ShortHelp:  "locks an encryption key with a passphrase (see new for creation)",
		ShortUsage: fmt.Sprintf("%s [opts] key lock", os.Args[0]),
		Exec:       KeyLock,
	}

	cmdKeyUnlock = &ffcli.Command{
		Name:       "unlock",
		ShortHelp:  "unlocks an encryption key with a passphrase",
		ShortUsage: fmt.Sprintf("%s [opts] key unlock", os.Args[0]),
		Exec:       KeyUnlock,
	}

	cmdKeys = &ffcli.Command{
		Name:       "key",
		ShortHelp:  "encryption key utilities",
		ShortUsage: fmt.Sprintf("%s [opts] key <subcommand> [opts]", os.Args[0]),
		Subcommands: []*ffcli.Command{
			cmdKeyLock,
			cmdKeyNew,
			cmdKeyUnlock,
		},
		Exec: help,
	}
)

func newKey() ([]byte, error) {
	var key [keySize]byte
	_, err := rand.Read(key[:])
	return key[:], err
}

func KeyNew(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return flag.ErrHelp
	}

	key, err := newKey()
	if err != nil {
		return err
	}

	fmt.Println("new key:", hex.EncodeToString(key))
	return nil
}

func parseUnlockedKey(hexKey string) (key []byte, err error) {
	key, err = hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("encryption key invalid hex: %w", err)
	}
	if len(key) != keySize {
		return nil, fmt.Errorf("encryption key invalid length")
	}
	return key, nil
}

func parseKey(output io.Writer, input *bufio.Reader, lockedKey string) (key []byte, err error) {
	idx := strings.Index(lockedKey, "-")
	if idx < 0 {
		return parseUnlockedKey(lockedKey)
	}
	if lockedKey[:idx] != "lock" {
		return nil, fmt.Errorf("invalid locked key")
	}
	data, err := hex.DecodeString(lockedKey[idx+1:])
	if err != nil {
		return nil, err
	}

	passphrase, err := readLine(output, input, "input passphrase: ")
	if err != nil {
		return nil, err
	}

	salt := data[:keySize]
	encrypted := data[keySize:]

	nonce, keyKey, err := lockKey(passphrase, salt)
	if err != nil {
		return nil, err
	}

	key, success := secretbox.Open(nil, encrypted, nonce, keyKey)
	if !success {
		return nil, fmt.Errorf("failed decrypting")
	}

	return key, nil
}

func readLine(output io.Writer, input *bufio.Reader, message string) (string, error) {
	_, err := fmt.Fprint(output, message)
	if err != nil {
		return "", err
	}
	result, err := input.ReadString('\n')
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return strings.TrimSuffix(result, "\n"), err
}

const nonceSize = 24
const keySize = sha256.Size

func lockKey(passphrase string, salt []byte) (*[nonceSize]byte, *[keySize]byte, error) {
	buf := argon2.IDKey([]byte(passphrase), salt, 10, 64*1024, 4, keySize)

	var key [keySize]byte
	n := copy(key[:], buf)
	if n != keySize {
		return nil, nil, fmt.Errorf("unexpected argon2 key length")
	}

	var nonce [nonceSize]byte
	n = copy(nonce[:], salt)
	if n != nonceSize {
		return nil, nil, fmt.Errorf("unexpected nonce length")
	}

	return &nonce, &key, nil
}

func KeyLock(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return flag.ErrHelp
	}

	input := bufio.NewReader(os.Stdin)

	encKeyHex, err := readLine(os.Stdout, input, "input 32 byte hex-encoded encryption key: ")
	if err != nil {
		return err
	}
	encKey, err := parseUnlockedKey(encKeyHex)
	if err != nil {
		return err
	}

	passphrase, err := readLine(os.Stdout, input, "input passphrase: ")
	if err != nil {
		return err
	}

	salt, err := newKey()
	if err != nil {
		return err
	}

	nonce, keyKey, err := lockKey(passphrase, salt)
	if err != nil {
		return err
	}

	_, err = fmt.Printf("lock-%x\n",
		append(
			append([]byte(nil), salt...),
			secretbox.Seal(nil, encKey, nonce, keyKey)...))
	return err
}

func KeyUnlock(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return flag.ErrHelp
	}

	input := bufio.NewReader(os.Stdin)

	lockedKey, err := readLine(os.Stdout, input, "input locked key: ")
	if err != nil {
		return err
	}

	encKey, err := parseKey(os.Stdout, input, lockedKey)
	if err != nil {
		return err
	}

	_, err = fmt.Printf("%x\n", encKey)
	return err
}
