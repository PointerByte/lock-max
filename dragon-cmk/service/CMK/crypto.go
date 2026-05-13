package cmk

import (
	"errors"
	"strconv"

	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
)

func (r *Repository) Encrypt(
	secretCmkKey string,
	plaintext string,
	additional *string,
) (ciphertext string, annotations map[string][]byte, err error) {
	material, err := r.loadKeyMaterial(secretCmkKey)
	if err != nil {
		return "", nil, err
	}

	if err := requirePurpose(material.cmkKey.Purpose, commonEntity.KeyPurposeEncrypt, errMsgKeyPurposeNotAllowEncryption); err != nil {
		return "", nil, err
	}

	publicKey := material.keyVersion.KID
	size := strconv.Itoa(material.keyVersion.Size)
	ctx, cancel := r.contextWithTimeout()
	defer cancel()

	switch material.cmkKey.Algorithm {
	case commonEntity.KeySymmetricDefault:
		ciphertext, err = r.security.EncryptAES(ctx, material.key, plaintext, additional)
	case commonEntity.KeyTypeRSAOAEP:
		ciphertext, err = r.security.RSA_OAEP_Encode(ctx, publicKey, plaintext)
	case commonEntity.KeyTypeECDH:
		ciphertext, err = r.security.ECC_Encode(ctx, publicKey, plaintext)
	default:
		return "", nil, errors.New(errMsgUnsupportedAlgorithm)
	}

	annotations = make(map[string][]byte)
	annotations["PublicKey"] = []byte(publicKey)
	annotations["Size"] = []byte(size)
	return
}

func (r *Repository) Decrypt(
	secretCmkKey string,
	ciphertext string,
	additional *string,
) (plaintext string, annotations map[string][]byte, err error) {
	material, err := r.loadKeyMaterial(secretCmkKey)
	if err != nil {
		return "", nil, err
	}

	if err := requirePurpose(material.cmkKey.Purpose, commonEntity.KeyPurposeEncrypt, errMsgKeyPurposeNotAllowDecryption); err != nil {
		return "", nil, err
	}

	publicKey := material.keyVersion.KID
	size := strconv.Itoa(material.keyVersion.Size)
	ctx, cancel := r.contextWithTimeout()
	defer cancel()

	switch material.cmkKey.Algorithm {
	case commonEntity.KeySymmetricDefault:
		plaintext, err = r.security.DecryptAES(ctx, material.key, ciphertext, additional)
	case commonEntity.KeyTypeRSAOAEP:
		plaintext, err = r.security.RSA_OAEP_Decode(ctx, material.key, ciphertext)
	case commonEntity.KeyTypeECDH:
		plaintext, err = r.security.ECC_Decode(ctx, material.key, ciphertext)
	default:
		return "", nil, errors.New(errMsgUnsupportedAlgorithm)
	}

	annotations = make(map[string][]byte)
	annotations["PublicKey"] = []byte(publicKey)
	annotations["Size"] = []byte(size)
	return plaintext, nil, err
}

func (r *Repository) Sing(secretCmkKey string, message string) (signature string, err error) {
	material, err := r.loadKeyMaterial(secretCmkKey)
	if err != nil {
		return "", err
	}

	if err := requirePurpose(material.cmkKey.Purpose, commonEntity.KeyPurposeSign, errMsgKeyPurposeNotAllowSigning); err != nil {
		return "", err
	}

	switch material.cmkKey.Algorithm {
	case commonEntity.KeyTypeRSAOAEP:
		ctx, cancel := r.contextWithTimeout()
		defer cancel()
		signature, err = r.security.SignRSAPSS(ctx, material.key, message)
	case commonEntity.KeyTypeRSAPKCS1v15SHA256:
		ctx, cancel := r.contextWithTimeout()
		defer cancel()
		signature, err = r.security.Sign_RSA_PKCS1v15_SHA256(ctx, material.key, message)
	case commonEntity.KeyTypeEdDSA:
		ctx, cancel := r.contextWithTimeout()
		defer cancel()
		signature, err = r.security.SignEd25519(ctx, material.key, message)
	default:
		return "", errors.New(errMsgUnsupportedAlgorithm)
	}
	return signature, err
}

func (r *Repository) Verify(secretCmkKey string, message, signature string) (valid bool, err error) {
	material, err := r.loadKeyMaterial(secretCmkKey)
	if err != nil {
		return false, err
	}

	if err := requirePurpose(material.cmkKey.Purpose, commonEntity.KeyPurposeSign, errMsgKeyPurposeNotAllowSigning); err != nil {
		return false, err
	}

	switch material.cmkKey.Algorithm {
	case commonEntity.KeyTypeRSAOAEP:
		ctx, cancel := r.contextWithTimeout()
		defer cancel()
		err = r.security.VerifyRSAPSS(ctx, material.key, message, signature)
	case commonEntity.KeyTypeRSAPKCS1v15SHA256:
		ctx, cancel := r.contextWithTimeout()
		defer cancel()
		err = r.security.Verify_RSA_PKCS1v15_SHA256(ctx, material.key, message, signature)
	case commonEntity.KeyTypeEdDSA:
		ctx, cancel := r.contextWithTimeout()
		defer cancel()
		err = r.security.VerifyEd25519(ctx, material.key, message, signature)
	default:
		return false, errors.New(errMsgUnsupportedAlgorithm)
	}
	return err == nil, err
}
