package magic

import (
	"runtime"
	"strconv"

	"github.com/jsipprell/keyctl"
)

// Keyring is a magic storage using linux thread keyring as the backend.
type Keyring struct {
	commChan chan keyringPacket
}

type KeyringOpts struct {
	TimeoutSeconds uint
}

func (k Keyring) Set(magic uint64) error {
	packet := keyringPacket{cmdtype: keyringCmdSet, magic: magic, out: make(chan keyringPacket)}
	k.commChan <- packet
	packet = <-packet.out
	return packet.err
}

func (k Keyring) Get() (uint64, error) {
	packet := keyringPacket{cmdtype: keyringCmdGet, out: make(chan keyringPacket)}
	k.commChan <- packet
	packet = <-packet.out
	return packet.magic, packet.err
}

func (k Keyring) Clear() error {
	packet := keyringPacket{cmdtype: keyringCmdClr, out: make(chan keyringPacket)}
	k.commChan <- packet
	packet = <-packet.out
	return packet.err
}

type keyringCmd uint

const (
	keyringCmdSet keyringCmd = 1
	keyringCmdGet keyringCmd = 2
	keyringCmdClr keyringCmd = 3
)

type keyringPacket struct {
	cmdtype keyringCmd
	err     error
	magic   uint64

	out chan keyringPacket
}

type keyringHandle struct {
	keyring keyctl.Keyring
	name    string

	key *keyctl.Key
}

func (k *keyringHandle) keyname() string {
	return "apparmor_magic_" + k.name
}

func (k *keyringHandle) Set(magic uint64) error {
	key, err := k.keyring.Add(k.keyname(), []byte(strconv.FormatUint(magic, 16)))
	if err != nil {
		return err
	}
	k.key = key
	return nil
}

func (k *keyringHandle) Get() (uint64, error) {
	if k.key != nil {
		keyBytes, err := k.key.Get()
		if err != nil {
			return 0, err
		}
		return strconv.ParseUint(string(keyBytes), 16, 64)
	}

	key, err := k.keyring.Search(k.keyname())
	if err != nil {
		return 0, err
	}
	keyBytes, err := key.Get()
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(string(keyBytes), 16, 64)
}

func (k *keyringHandle) Clear() error {
	if k.key != nil {
		return k.key.Unlink()
	}
	if key, err := k.keyring.Search(k.keyname()); err == nil {
		return key.Unlink()
	}
	return nil
}

// NewKeyring returns a new Keyring (kernel keyring backed) magic storage instance.
//
// Default options is no timeout (0 seconds)
func NewKeyring(opts *KeyringOpts) (Store, error) {
	if opts == nil {
		opts = &KeyringOpts{TimeoutSeconds: 0}
	}

	commChan := make(chan keyringPacket)
	initErr := make(chan error)
	go func() {
		runtime.LockOSThread()
		keyring, err := keyctl.ThreadKeyring()
		keyring.SetDefaultTimeout(opts.TimeoutSeconds)
		handle := &keyringHandle{keyring: keyring}

		initErr <- err
		for {
			packet, ok := <-commChan
			if !ok {
				return
			}
			switch packet.cmdtype {
			case keyringCmdSet:
				packet.err = handle.Set(packet.magic)
			case keyringCmdGet:
				packet.magic, packet.err = handle.Get()
			case keyringCmdClr:
				packet.err = handle.Clear()
			}
			packet.out <- packet
		}
	}()
	if err := <-initErr; err != nil {
		return nil, err
	}
	return &Keyring{commChan: commChan}, nil
}
