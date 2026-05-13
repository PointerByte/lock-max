package cmk

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	jwtservice "github.com/PointerByte/GoForge/security/auth/jwt"
	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
)

type jwtTokenService interface {
	CreateWithContext(ctx context.Context, claims any) (string, error)
	ReadWithContext(ctx context.Context, token string, destination any) error
}

var newJWTServiceFn = newJWTService

func newJWTService(input jwtservice.ConfigServiceInput) (jwtTokenService, error) {
	if strings.ToUpper(strings.TrimSpace(input.Algorithm)) == "HS256" {
		secret := ""
		if input.HMACSecretKey != nil {
			secret = *input.HMACSecretKey
		}
		return jwtservice.NewHMACService(jwtservice.HMACServiceInput{
			Secret:    secret,
			Validator: input.Validator,
		})
	}
	return jwtservice.NewConfiguredService(input)
}

func newJWTConfig(algorithm, key, publicKey string) jwtservice.ConfigServiceInput {
	return jwtservice.ConfigServiceInput{
		Algorithm:          strings.ToLower(algorithm),
		Validator:          validateJWTExpirationClaim,
		HMACSecretKey:      &key,
		RSAPrivateKeyKey:   &key,
		RSAPublicKeyKey:    &publicKey,
		EdDSAPrivateKeyKey: &key,
		EdDSAPublicKeyKey:  &publicKey,
	}
}

func validateJWTExpirationClaim(ctx context.Context, token jwtservice.Token) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var claims map[string]json.RawMessage
	if err := json.Unmarshal(token.Claims, &claims); err != nil {
		return nil
	}

	expRaw, ok := claims["exp"]
	if !ok {
		return nil
	}

	var exp float64
	if err := json.Unmarshal(expRaw, &exp); err != nil {
		return errors.New("invalid jwt expiration claim")
	}

	if exp <= float64(time.Now().Unix()) {
		return errors.New("jwt has expired")
	}
	return nil
}

func (r *Repository) CreateJWT(
	ctx context.Context,
	secretCmkKey string,
	algorithm string,
	claims any,
) (string, error) {
	material, err := r.loadKeyMaterial(secretCmkKey)
	if err != nil {
		return "", err
	}

	if err := requirePurpose(material.cmkKey.Purpose, commonEntity.KeyPurposeSign, errMsgKeyPurposeNotAllowJWTSigning); err != nil {
		return "", err
	}

	service, err := newJWTServiceFn(
		newJWTConfig(algorithm, material.key, material.keyVersion.KID),
	)
	if err != nil {
		return "", err
	}
	jwtStr, err := service.CreateWithContext(ctx, claims)
	if err != nil {
		return "", err
	}
	var decodedClaims any
	if err := service.ReadWithContext(ctx, jwtStr, &decodedClaims); err != nil {
		return "", err
	}
	return jwtStr, nil
}

func (r *Repository) VerifyJWT(
	ctx context.Context,
	secretCmkKey string,
	algorithm string,
	token string,
) error {
	material, err := r.loadKeyMaterial(secretCmkKey)
	if err != nil {
		return err
	}

	if err := requirePurpose(material.cmkKey.Purpose, commonEntity.KeyPurposeSign, errMsgKeyPurposeNotAllowJWTSigning); err != nil {
		return err
	}

	service, err := newJWTServiceFn(
		newJWTConfig(algorithm, material.key, material.keyVersion.KID),
	)
	if err != nil {
		return err
	}
	timeoutCtx, cancel := contextWithTimeout(ctx)
	defer cancel()
	var claims any
	return service.ReadWithContext(timeoutCtx, token, &claims)
}

func (r *Repository) ReadJWT(
	ctx context.Context,
	secretCmkKey string,
	algorithm string,
	token string,
	claims any,
) error {
	material, err := r.loadKeyMaterial(secretCmkKey)
	if err != nil {
		return err
	}

	if err := requirePurpose(material.cmkKey.Purpose, commonEntity.KeyPurposeSign, errMsgKeyPurposeNotAllowJWTSigning); err != nil {
		return err
	}

	service, err := newJWTServiceFn(
		newJWTConfig(algorithm, material.key, material.keyVersion.KID),
	)
	if err != nil {
		return err
	}
	return service.ReadWithContext(ctx, token, &claims)
}
