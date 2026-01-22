package service

import (
	"context"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/google/uuid"
	authdomain "github.com/smallbiznis/railzway/internal/auth/domain"
	"github.com/smallbiznis/railzway/internal/auth/repository"
	"github.com/smallbiznis/railzway/pkg/db"
	"go.uber.org/zap"
)

func newTestService(t *testing.T) authdomain.Service {
	t.Helper()

	dbConn, err := db.NewTest()
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := dbConn.AutoMigrate(&authdomain.User{}, &authdomain.Session{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo, sessionRepo := repository.New(dbConn)
	node, err := snowflake.NewNode(1)
	if err != nil {
		t.Fatalf("failed to create snowflake node: %v", err)
	}

	return New(zap.NewNop(), repo, sessionRepo, node)
}

func TestLoginWrongPassword(t *testing.T) {
	svc := newTestService(t)

	user, err := svc.CreateUser(context.Background(), authdomain.CreateUserRequest{
		Email:    "alice@example.com",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if user == nil {
		t.Fatal("expected user")
	}

	_, err = svc.Login(context.Background(), authdomain.LoginRequest{
		Email:    "alice@example.com",
		Password: "wrong-password",
	})
	if err != authdomain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestCreateUserLocalExternalIDUUID(t *testing.T) {
	svc := newTestService(t)

	user, err := svc.CreateUser(context.Background(), authdomain.CreateUserRequest{
		Email:    "bob@example.com",
		Password: "strong-password",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if user.Provider != "local" {
		t.Fatalf("expected provider local, got %s", user.Provider)
	}
	if user.ExternalID == "" {
		t.Fatalf("expected external id")
	}
	if _, err := uuid.Parse(user.ExternalID); err != nil {
		t.Fatalf("expected external id UUID, got %v", err)
	}
}
