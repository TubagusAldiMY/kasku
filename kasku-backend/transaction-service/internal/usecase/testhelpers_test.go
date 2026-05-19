package usecase_test

import "github.com/google/uuid"

const testTenant = "tenant_550e8400_e29b_41d4_a716_446655440000"

func testUserID() string    { return uuid.New().String() }
func testAccountID() string { return uuid.New().String() }
