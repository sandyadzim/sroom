package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/dto"
	"identity/internal/identity/model"
	"identity/internal/identity/password"
)

func (r *accountUsers) ListCustomers(_ context.Context, limit, offset int, _, status string) ([]model.User, int64, error) {
	var users []model.User
	for _, user := range r.users {
		if user.Role == model.RoleCustomer && (status == "" || user.Status == status) {
			users = append(users, user)
		}
	}
	total := int64(len(users))
	if offset >= len(users) {
		return nil, total, nil
	}
	return users[offset:min(offset+limit, len(users))], total, nil
}

func (r *accountUsers) FindCustomerByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user.Role != model.RoleCustomer {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (r *accountUsers) UpdateCustomerStatus(ctx context.Context, id uuid.UUID, status string) (*model.User, error) {
	user, err := r.FindCustomerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Status = status
	r.users[id] = *user
	r.writes++
	return user, nil
}

func TestCustomerManagement(t *testing.T) {
	app, users, _ := accountTestApp()
	const root = "/api/v1/identity/customers"
	const jsonType = "application/json"

	var customer, admin model.User
	for _, user := range users.users {
		if user.Role == model.RoleCustomer {
			customer = user
		}
		if user.Role == model.RoleAdmin {
			admin = user
		}
	}
	hash, err := password.Hash("synthetic-password")
	if err != nil {
		t.Fatal(err)
	}
	customer.PasswordHash = hash
	users.users[customer.ID] = customer
	detailURL := root + "/" + customer.ID.String()
	statusURL := detailURL + "/status"

	for _, bearer := range []string{"admin-token", "owner-token"} {
		body := request(t, app, "GET", root+"?status=active", "", "", bearer, 200)
		var list dto.CustomerListResponse
		if err := json.Unmarshal(body, &list); err != nil || len(list.Data) != 1 ||
			list.Data[0].ID != customer.ID.String() || list.Page.Total != 1 || list.Page.Limit != 20 {
			t.Fatal("list must contain only the customer with default pagination")
		}
		body = request(t, app, "GET", detailURL, "", "", bearer, 200)
		var detail struct{ Data map[string]any }
		if err := json.Unmarshal(body, &detail); err != nil || len(detail.Data) != 6 ||
			detail.Data["id"] != customer.ID.String() || detail.Data["email"] != customer.Email ||
			detail.Data["status"] != model.StatusActive || detail.Data["password_hash"] != nil {
			t.Fatal("detail must return only the customer account fields")
		}
	}

	for _, action := range []struct{ method, path, body string }{
		{"GET", root, ""},
		{"GET", detailURL, ""},
		{"PATCH", statusURL, `{"status":"inactive"}`},
	} {
		request(t, app, action.method, action.path, action.body, jsonType, "", 401)
		request(t, app, action.method, action.path, action.body, jsonType, "customer-token", 403)
	}
	request(t, app, "GET", root+"/invalid-id", "", "", "admin-token", 400)
	request(t, app, "GET", root+"?status=unknown", "", "", "admin-token", 400)
	request(t, app, "GET", root+"/"+admin.ID.String(), "", "", "owner-token", 404)
	request(t, app, "PATCH", root+"/"+admin.ID.String()+"/status", `{"status":"inactive"}`, jsonType, "owner-token", 404)
	for _, body := range []string{`{}`, `{"status":null}`, `{"status":"enabled"}`, `{"status":"inactive","full_name":"Forged"}`, `{"status":"inactive","role":"OWNER"}`} {
		request(t, app, "PATCH", statusURL, body, jsonType, "admin-token", 400)
	}
	if users.writes != 0 {
		t.Fatal("rejected requests must not update accounts")
	}

	existing := accountLogin(t, app, customer.Email, "synthetic-password", 200)
	request(t, app, "PATCH", statusURL, `{"status":"inactive"}`, jsonType, "admin-token", 200)
	accountLogin(t, app, customer.Email, "synthetic-password", 401)
	request(t, app, "GET", "/api/v1/identity/me", "", "", existing, 200)
	request(t, app, "PATCH", statusURL, `{"status":"active"}`, jsonType, "owner-token", 200)
	accountLogin(t, app, customer.Email, "synthetic-password", 200)
}
