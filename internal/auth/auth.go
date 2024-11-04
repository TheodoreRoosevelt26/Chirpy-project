package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashString, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashString), nil
}

func CheckPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims(claims))
	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	subject, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.UUID{}, err
	}
	return subject, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	check := []rune{'B', 'e', 'a', 'r', 'e', 'r', ' '}
	bearer := headers.Get("Authorization")
	if len(bearer) < 8 {
		return "", fmt.Errorf("401 unauthorized")
	}
	partBearer := bearer[:7]
	runes := []rune(partBearer)
	for i := 0; i < len(check); i++ {
		if check[i] != runes[i] {
			return "", fmt.Errorf("401 unauthorized")
		}
	}
	stringToken := bearer[7:]
	return stringToken, nil
}
