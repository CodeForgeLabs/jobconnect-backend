package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	jwt.RegisteredClaims
}

func CreateToken(userId uint, userRole string) (string, error) {
	config := LoadJWTConfig()

	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprint(userId),
			Issuer:    "jobconnect-app",
			Audience:  []string{userRole},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(60 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.SECRET_KEY))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func VerifyToken(tokenString string) (*jwt.Token, *CustomClaims, error) {
	config := LoadJWTConfig()

	claims := &CustomClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.SECRET_KEY), nil
	})
	if err != nil {
		return nil, nil, err
	}

	if !token.Valid {
		return nil, nil, errors.New("invalid token")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, nil, errors.New("token expired")
	}

	return token, claims, nil
}

/*

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request){
	user := new(entities.User)
	err := json.NewDecoder(r.Body).Decode(&user)
	if err !=nil{
		util.WriteError(w,http.StatusBadRequest,err)
		return
	}
	existingUser,err := h.iuserusecase.Login(user)
	if err != nil{
		util.WriteError(w,http.StatusBadRequest,err)
		return
	}

	token, err :=auth.CreateToken(existingUser.ID,string(existingUser.Role))
	if err != nil{
		util.WriteError(w,http.StatusBadRequest,err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	})
	util.WriteJSON(w,http.StatusOK,map[string]string{"token":token})
}

*/
