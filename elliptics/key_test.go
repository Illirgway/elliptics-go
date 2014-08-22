package elliptics

import (
	"errors"
	"testing"
)

/*
	Key
*/

const (
	badKeyCreationArg  = 9999
	goodKeyCreationArg = "some_key"
)

func TestKeyDefaultCreationAndFree(t *testing.T) {
	key, err := NewKey()
	if err != nil {
		t.Errorf("%v", key)
	}

	if key.ById() {
		t.Errorf("%s", "Create key without ID")
	}

	key.Free()
}

func TestKeyCreationAndFree(t *testing.T) {
	_, err := NewKey(badKeyCreationArg)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	key, err := NewKey(goodKeyCreationArg)
	if err != nil {
		t.Fatalf("Error in a key creation, got %v", err)
	}
	key.Free()
}

/*
	Error
*/

func TestDnetError(t *testing.T) {
	const (
		dnet_code = 100
		dnet_flag = 16
		dnet_msg  = "dummy_dnet_error_message"
	)
	derr := DnetError{
		Code:    dnet_code,
		Flags:   dnet_flag,
		Message: dnet_msg,
	}

	dummy_err := errors.New("dummy_err")

	if msg := ErrorData(&derr); msg != dnet_msg {
		t.Errorf("ErroData: expected %s, got %s", dnet_msg, msg)
	}

	if msg := ErrorData(dummy_err); msg != dummy_err.Error() {
		t.Errorf("ErroData: expected %s, got %s", dummy_err.Error(), msg)
	}

	if errcode := ErrorStatus(&derr); errcode != dnet_code {
		t.Errorf("ErroStatus: expected %d, got %d", dnet_code, errcode)
	}

	if errcode := ErrorStatus(dummy_err); errcode != -22 {
		t.Errorf("ErroStatus: expected %d, got %d", -22, errcode)
	}

	if len(derr.Error()) == 0 {
		t.Errorf("DnetError: a malformed error representation")
	}

}
