package service

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) List(ctx context.Context) ([]domain.User, error) {
	return s.repo.List(ctx)
}

func (s *UserService) Get(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.FindByID(ctx, id)
}

type UserInput struct {
	Name     string
	Email    string
	Password string
	Role     string
	IsActive *bool
}

func (s *UserService) Create(ctx context.Context, in UserInput) (*domain.User, error) {
	name := strings.TrimSpace(in.Name)
	email := strings.ToLower(strings.TrimSpace(in.Email))
	role := strings.ToLower(strings.TrimSpace(in.Role))
	if name == "" || email == "" || in.Password == "" {
		return nil, domain.ErrInvalidInput
	}
	if role != string(domain.RoleAdmin) && role != string(domain.RoleStaff) {
		return nil, domain.ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	return s.repo.Create(ctx, repository.UserFields{
		Name: name, Email: email, PasswordHash: string(hash),
		Role: role, IsActive: active,
	})
}

type UserPatchInput struct {
	Name     *string
	Email    *string
	Password *string
	Role     *string
	IsActive *bool
}

func (s *UserService) Update(ctx context.Context, id string, in UserPatchInput) (*domain.User, error) {
	patch := repository.UserPatch{IsActive: in.IsActive}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, domain.ErrInvalidInput
		}
		patch.Name = &name
	}
	if in.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*in.Email))
		if email == "" {
			return nil, domain.ErrInvalidInput
		}
		patch.Email = &email
	}
	if in.Role != nil {
		role := strings.ToLower(strings.TrimSpace(*in.Role))
		if role != string(domain.RoleAdmin) && role != string(domain.RoleStaff) {
			return nil, domain.ErrInvalidInput
		}
		patch.Role = &role
	}
	if in.Password != nil && *in.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*in.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		s := string(hash)
		patch.PasswordHash = &s
	}
	return s.repo.Update(ctx, id, patch)
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *UserService) ChangePassword(ctx context.Context, userID, current, next string) error {
	if userID == "" {
		return domain.ErrUnauthorized
	}
	if len(next) < 8 {
		return domain.ErrInvalidInput
	}
	u, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(current)); err != nil {
		return domain.ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	nh := string(hash)
	_, err = s.repo.Update(ctx, userID, repository.UserPatch{PasswordHash: &nh})
	return err
}
