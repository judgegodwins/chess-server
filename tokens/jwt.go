package tokens

import (
	"errors"
	"fmt"
	"time"

	// "time"

	"github.com/golang-jwt/jwt/v5"
)

type Payload struct {
	ID  string `json:"id"`
	Username string `json:"username"`
}

func NewJWTToken(claims jwt.MapClaims, secret []byte) (string, error) {
	claims["exp"] = time.Now().Add(30 * 24 * time.Hour).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(secret)
}

func ParseJWTToken(tokenString string, secret []byte) (*Payload, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok || !token.Valid {
		return nil, errors.New("invalid jwt token")
	}

	username, ok1 := claims["username"].(string)
	id, ok2 := claims["id"].(string)

	ok = ok1 && ok2

	if (!ok || username == "" || id == "") {
		return nil, errors.New("invalid token")
	}

	payload := &Payload{
		Username: username,
		ID: id,
	}

	return payload, nil
}
