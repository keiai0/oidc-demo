package scim

import (
	"time"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

const (
	userSchema = "urn:ietf:params:scim:schemas:core:2.0:User"
)

// SCIMUser は RFC 7643 のユーザーリソース表現。
type SCIMUser struct {
	Schemas    []string    `json:"schemas"`
	ID         string      `json:"id"`
	ExternalID string      `json:"externalId,omitempty"`
	UserName   string      `json:"userName"`
	Name       *SCIMName   `json:"name,omitempty"`
	Emails     []SCIMEmail `json:"emails,omitempty"`
	Active     bool        `json:"active"`
	Meta       SCIMMeta    `json:"meta"`
}

// SCIMName は SCIM の name 属性。
type SCIMName struct {
	Formatted string `json:"formatted,omitempty"`
}

// SCIMEmail は SCIM の email 属性。
type SCIMEmail struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

// SCIMMeta は SCIM のメタ情報。
type SCIMMeta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created"`
	LastModified string `json:"lastModified"`
	Location     string `json:"location,omitempty"`
}

// SCIMListResponse は RFC 7644 Section 3.4.2 のリストレスポンス。
type SCIMListResponse struct {
	Schemas      []string   `json:"schemas"`
	TotalResults int64      `json:"totalResults"`
	StartIndex   int        `json:"startIndex"`
	ItemsPerPage int        `json:"itemsPerPage"`
	Resources    []SCIMUser `json:"Resources"`
}

// ToSCIMUser は model.User を SCIM User リソースに変換する。
func ToSCIMUser(user *model.User, baseURL string) SCIMUser {
	scimUser := SCIMUser{
		Schemas:  []string{userSchema},
		ID:       user.ID.String(),
		UserName: user.LoginID,
		Active:   user.Status == "active",
		Meta: SCIMMeta{
			ResourceType: "User",
			Created:      user.CreatedAt.Format(time.RFC3339),
			LastModified: user.UpdatedAt.Format(time.RFC3339),
			Location:     baseURL + "/" + user.ID.String(),
		},
	}

	if user.ExternalID != nil {
		scimUser.ExternalID = *user.ExternalID
	}

	if user.Name != nil && *user.Name != "" {
		scimUser.Name = &SCIMName{Formatted: *user.Name}
	}

	if user.Email != "" {
		scimUser.Emails = []SCIMEmail{
			{Value: user.Email, Primary: true},
		}
	}

	return scimUser
}

// SCIMUserCreateRequest は SCIM ユーザー作成リクエスト。
type SCIMUserCreateRequest struct {
	Schemas    []string    `json:"schemas"`
	ExternalID string      `json:"externalId,omitempty"`
	UserName   string      `json:"userName"`
	Name       *SCIMName   `json:"name,omitempty"`
	Emails     []SCIMEmail `json:"emails,omitempty"`
	Active     *bool       `json:"active,omitempty"`
	Password   string      `json:"password,omitempty"`
}

// FromSCIMUserCreate は SCIM 作成リクエストから model.User を生成する。
func FromSCIMUserCreate(req SCIMUserCreateRequest, tenantID uuid.UUID) *model.User {
	user := &model.User{
		TenantID: tenantID,
		LoginID:  req.UserName,
		Status:   "active",
	}

	if req.ExternalID != "" {
		user.ExternalID = &req.ExternalID
	}

	if req.Name != nil && req.Name.Formatted != "" {
		user.Name = &req.Name.Formatted
	}

	if len(req.Emails) > 0 {
		user.Email = req.Emails[0].Value
	}

	if req.Active != nil && !*req.Active {
		user.Status = "disabled"
	}

	return user
}

// SCIMPatchRequest は RFC 7644 Section 3.5.2 の PATCH リクエスト。
type SCIMPatchRequest struct {
	Schemas    []string        `json:"schemas"`
	Operations []SCIMOperation `json:"Operations"`
}

// SCIMOperation は PATCH の個別操作。
type SCIMOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"`
}
