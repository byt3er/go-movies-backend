// most of the code realated to user authentication
// *****************************
// we going to need all of these in order to issue tokens,
// in order to validate, and in order to issue token-cookies
// and things like that
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Auth struct {
	Issuer      string // who is issuing this token //i.e company.com, example.com
	Audience    string // who should be able to use these tokens
	Secret      string // this is our secret key, a strong secret which we use to sign out tokens
	TokenExpiry time.Duration
	// the refresh token has less information but which can be used to
	// reauthenticate the user and it typically has a much longer expiry time,
	// sometimes as long as year, sometimes as short as two weeks
	// it's entirely up to you
	RefreshExpiry time.Duration // when does my refresh token expires
	//we're goin to give our refresh tokens to users as cookie as HTTP only
	// secure cookie, which is not accessible from javascript, but which
	// will be included in any request made to our backend
	// that's how we will get refresh token from the user
	// so that means we need some cookie specifications, some parameters for
	// our cookies.
	CookieDomain string // i.e example.com
	CookiePath   string // path to the cookie
	CookieName   string
}

// this type will only contain enough information
// the minimal amount of information we need in order to be able to
// issue a token

type jwtUser struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// this type is we're going to issue our token as a pair
type TokenPairs struct {
	Token        string `json:"access_token"` // actual JWT token we issue
	RefreshToken string `json:refresh_token"` // the refresh token

}

// everytime you have a jwt issue, that jwt has certain things that are
// called claims and you might clam that this token is only for this audience
// you might claim that this token has the user_id 1 associated with it
// whatever information you want, there are things you have to have in there
// which will be adding, and the there things you can add that aren't necessary
// they're optional
// you're not going to put too much information in that,
// but you can put other information in.
type Claims struct {
	jwt.RegisteredClaims
}

// generate token pair and that will generate a JWT and the refresh token
func (j *Auth) GenerateTokenPair(user *jwtUser) (TokenPairs, error) {
	// Create a token (that will be an empty token object)
	token := jwt.New(jwt.SigningMethodHS256) // signing Method

	// Set the claims( So what does this token clain to be?)
	// It will have names and a subject an issuser, the audience,
	// all kinds of claim
	claims := token.Claims.(jwt.MapClaims) // is a Map
	claims["name"] = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	//*******NOTE****** the rest of the claims are all going to be in form of three lowercase
	// characters and you have to use these, you can't make up your own here
	claims["sub"] = fmt.Sprint(user.ID)     // subject: the userid in the database
	claims["aud"] = j.Audience              // audience
	claims["iss"] = j.Issuer                // issuer
	claims["iat"] = time.Now().UTC().Unix() // when was this issued?
	claims["typ"] = "JWT"

	// Set the expiry for JWT
	claims["exp"] = time.Now().UTC().Add(j.TokenExpiry).Unix()

	// Create a signed token
	signedAccessToken, err := token.SignedString([]byte(j.Secret))
	if err != nil {
		log.Println("error in Create a signed token")
		return TokenPairs{}, err
	}

	// Create a refresh token and set claims
	refreshToken := jwt.New(jwt.SigningMethodHS256)
	refreshTokenClaims := refreshToken.Claims.(jwt.MapClaims)
	refreshTokenClaims["sub"] = fmt.Sprint(user.ID) // userid in Database
	refreshTokenClaims["iat"] = time.Now().UTC().Unix()

	// Set the expiry for the refresh token
	refreshTokenClaims["exp"] = time.Now().UTC().Add(j.RefreshExpiry).Unix()

	// Create signed refresh token
	signedRefreshToken, err := refreshToken.SignedString([]byte(j.Secret))
	if err != nil {
		log.Println("error in create signed refresh token")
		return TokenPairs{}, err
	}

	// Creae TokenPairs and populate with signed tokens
	var tokenPairs = TokenPairs{
		Token:        signedAccessToken,
		RefreshToken: signedRefreshToken,
	}

	// Return TokenPairs
	return tokenPairs, nil

}

func (j *Auth) GetRefreshCookie(refreshToken string) *http.Cookie {
	return &http.Cookie{
		Name:     j.CookieName,
		Path:     j.CookiePath,
		Value:    refreshToken,
		Expires:  time.Now().Add(j.RefreshExpiry),
		MaxAge:   int(j.RefreshExpiry.Seconds()),
		SameSite: http.SameSiteStrictMode, // make this cookie limited to this site
		Domain:   j.CookieDomain,
		// these make the cookie more secure
		HttpOnly: true, // so javascript will have no access to this cookie in a web browser
		Secure:   true,
	}
}

// Delete refresh cookie from the user browser
func (j *Auth) GetExpiredRefreshCookie() *http.Cookie {
	return &http.Cookie{
		Name:     j.CookieName,
		Path:     j.CookiePath,
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		SameSite: http.SameSiteStrictMode, // make this cookie limited to this site
		Domain:   j.CookieDomain,
		// these make the cookie more secure
		HttpOnly: true, // so javascript will have no access to this cookie in a web browser
		Secure:   true,
	}
}

// GetTokenFromHeaderAndVerify get the token from the header, verify and
// extract the authorization header from a request and validate the token
// and returns the token as string and pointer to our claims and potentially an error
func (j *Auth) GetTokenFromHeaderAndVerify(w http.ResponseWriter, r *http.Request) (string, *Claims, error) {
	// add header to the response
	w.Header().Add("Vary", "Authorization") // it would probably work without it but its is a good practice

	// get auth header from request
	// (if the value doesn't exists it will return empty string)
	authHeader := r.Header.Get("Authorization")

	// sanity check
	if authHeader == "" {
		// there is no header at all
		return "", nil, errors.New("no auth header")
	}

	// split the header on spaces
	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 {
		return "", nil, errors.New("invalid auth header")
	}

	// if we get here, we know we have a authorization header that consists
	// of two things separated by a space

	// the first thing in headerParts should be the word "Bearer"

	// check to see if we have the word "Bearer"
	if headerParts[0] != "Bearer" {
		// invalid authorization header
		return "", nil, errors.New("invalid auth header")
	}

	// the next thing in the headerParts must be the token(Bearer-token)
	token := headerParts[1]

	// declare an empty claims to store any claims that Token might make
	claims := &Claims{} // that's we're going to read our claims into.

	// parse the token
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		//need to validate the signing method and make sure that it's what we expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method:%v", token.Header["alg"])
		}
		return []byte(j.Secret), nil
	})

	if err != nil {
		// check if the error has the prefix token is expired then return
		// the error "expired token"
		// otherwise return whatever the errors happens to be
		if strings.HasPrefix(err.Error(), "token is expired by") {
			return "", nil, errors.New("expired token")
		}
		return "", nil, err
	}

	// if we pass this then we have a valid token

	// check wether we issue this token
	if claims.Issuer != j.Issuer {
		return "", nil, errors.New("invalid issuer")
	}

	// if we pass that, then we have a valid non-expired token that
	// we actually issued
	return token, claims, nil

}
