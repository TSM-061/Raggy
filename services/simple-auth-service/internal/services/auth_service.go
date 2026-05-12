package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/TSM-061/Raggy/shared/auth"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/password"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/session"
	"github.com/TSM-061/Raggy/simple-auth-service/internal/user"
)

type AuthService struct {
	users user.Repo

	hasher         *password.Argon2Hasher
	signer         *auth.TokenSigner
	sessionManager *session.Manager
}

const UsernameRegexStr = `^[\p{L}0-9][\p{L}0-9._-]*[\p{L}0-9]$`

var usernameRegex = regexp.MustCompile(UsernameRegexStr)

func NewAuthService(
	userRepo user.Repo,
	hasher *password.Argon2Hasher,
	signer *auth.TokenSigner,
	sm *session.Manager) *AuthService {

	return &AuthService{
		users:          userRepo,
		hasher:         hasher,
		sessionManager: sm,
		signer:         signer,
	}
}

func (s *AuthService) Register(
	ctx context.Context, username string, password string) (*user.User, error) {

	if len(username) < user.MinUsernameLength ||
		len(username) > user.MaxUsernameLength {
		return nil,
			fmt.Errorf(
				"%w: 'username' wanted length %d-%d, got %d",
				serviceerr.InvalidInput,
				user.MinUsernameLength,
				user.MaxUsernameLength,
				len(username),
			)
	}

	if len(password) < user.MinPasswordLength ||
		len(password) > user.MaxPasswordLength {
		return nil,
			fmt.Errorf(
				"%w: 'password' wanted length %d-%d, got %d",
				serviceerr.InvalidInput,
				user.MinPasswordLength,
				user.MaxPasswordLength,
				len(password),
			)
	}

	if !usernameRegex.MatchString(username) {
		return nil, fmt.Errorf(
			"%w: must match regex %s",
			serviceerr.InvalidInput,
			UsernameRegexStr,
		)
	}

	userExists, err := s.users.Exists(ctx, username)
	if err != nil {
		return nil, err
	}
	if userExists {
		return nil, fmt.Errorf("%w: username in use", serviceerr.Conflict)
	}

	createdUser := &user.User{
		Username:     username,
		PasswordHash: s.hasher.Hash(password),
	}

	if err := s.users.Create(ctx, createdUser); err != nil {
		return nil, err
	}

	return createdUser, nil
}

type AuthResult struct {
	Tokens *TokenPair
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) Signin(
	ctx context.Context, username string, plaintextPassword string) (*AuthResult, error) {

	foundUser, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		// TODO reflect this in test cases, no notfound and instead unauthorized
		if errors.Is(err, serviceerr.NotFound) {
			return nil, fmt.Errorf("%w: invalid credentials", serviceerr.Unauthorized)
		}
		return nil, err
	}

	isMatch, err := s.hasher.Verify(plaintextPassword, foundUser.PasswordHash)
	if !isMatch {
		return nil, fmt.Errorf("%w: invalid credentials", serviceerr.Unauthorized)
	}

	refreshToken, err := s.sessionManager.Create(ctx, foundUser.ID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.signer.Sign(foundUser.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Tokens: &TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, token string) (*AuthResult, error) {
	refreshResult, err := s.sessionManager.Refresh(ctx, token)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.signer.Sign(refreshResult.UserID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Tokens: &TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshResult.RefreshToken,
		},
	}, nil
}

func (s *AuthService) Signout(ctx context.Context, token string) error {
	return s.sessionManager.Delete(ctx, token)
}
