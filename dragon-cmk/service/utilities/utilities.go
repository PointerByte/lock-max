package utilities

import (
	"encoding/base64"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var errInvalidSecretCmkKey = errors.New("invalid secret cmk key")

func GetSecretCmkKey(secretCmkKey string) (idCmkKey *uuid.UUID, idCmkKeyVersion *uuid.UUID, _ error) {
	cmkKey, err := base64.StdEncoding.DecodeString(secretCmkKey)
	if err != nil {
		cmkKey = []byte(secretCmkKey)
	}

	dataCmkKey := strings.Split(string(cmkKey), ".")
	if len(dataCmkKey) != 2 {
		return nil, nil, errInvalidSecretCmkKey
	}

	_idCmkKey, err := uuid.Parse(dataCmkKey[0])
	if err != nil {
		return nil, nil, err
	}

	_idCmkKeyVersion, err := uuid.Parse(dataCmkKey[1])
	if err != nil {
		return nil, nil, err
	}
	return &_idCmkKey, &_idCmkKeyVersion, nil
}
